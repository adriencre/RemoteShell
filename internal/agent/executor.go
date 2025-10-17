package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"remoteshell/internal/common"
)

// Executor gère l'exécution de commandes avec un shell persistant
type Executor struct {
	workingDir  string
	env         map[string]string
	shellCmd    *exec.Cmd
	shellIn     io.WriteCloser
	shellOut    io.ReadCloser
	shellMutex  sync.Mutex
	initialized bool
}

// NewExecutor crée un nouvel exécuteur
func NewExecutor(workingDir string) *Executor {
	return &Executor{
		workingDir:  workingDir,
		env:         make(map[string]string),
		initialized: false,
	}
}

// SetEnv définit une variable d'environnement
func (e *Executor) SetEnv(key, value string) {
	e.env[key] = value
}

// initShell initialise le shell persistant
func (e *Executor) initShell() error {
	if e.initialized {
		return nil
	}

	e.shellMutex.Lock()
	defer e.shellMutex.Unlock()

	if e.initialized {
		return nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe")
	} else {
		// Utiliser bash avec sudo pour les privilèges root (sans mode interactif)
		cmd = exec.Command("sudo", "-n", "bash")
	}

	// Configurer les pipes
	var err error
	e.shellIn, err = cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("erreur création stdin pipe: %v", err)
	}

	e.shellOut, err = cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("erreur création stdout pipe: %v", err)
	}

	// Rediriger stderr vers stdout pour capturer les erreurs
	cmd.Stderr = cmd.Stdout

	// Démarrer le shell
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erreur démarrage shell: %v", err)
	}

	e.shellCmd = cmd
	e.initialized = true

	// Attendre que le shell soit prêt
	time.Sleep(200 * time.Millisecond)

	// Définir le répertoire de travail initial
	if e.workingDir != "" {
		e.shellIn.Write([]byte(fmt.Sprintf("cd %s\n", e.workingDir)))
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

// sendCommand envoie une commande au shell
func (e *Executor) sendCommand(command string) error {
	if !e.initialized {
		if err := e.initShell(); err != nil {
			return err
		}
	}

	_, err := e.shellIn.Write([]byte(command + "\n"))
	return err
}

// readOutput lit la sortie du shell avec un marqueur de fin
func (e *Executor) readOutput(marker string, timeout time.Duration) (string, error) {
	if !e.initialized {
		return "", fmt.Errorf("shell non initialisé")
	}

	// Créer un canal pour la lecture
	outputChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(e.shellOut)
		var output strings.Builder

		// Lire jusqu'à ce qu'on trouve le marqueur de fin
		for scanner.Scan() {
			line := scanner.Text()

			// Si on trouve le marqueur, on arrête sans l'inclure
			if strings.Contains(line, marker) {
				break
			}

			output.WriteString(line + "\n")
		}

		if err := scanner.Err(); err != nil {
			errorChan <- err
			return
		}

		outputChan <- output.String()
	}()

	// Attendre avec timeout
	select {
	case output := <-outputChan:
		return output, nil
	case err := <-errorChan:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout de lecture")
	}
}

// Execute exécute une commande et retourne le résultat
func (e *Executor) Execute(ctx context.Context, cmdData *common.CommandData) (*common.CommandOutput, error) {
	start := time.Now()

	// Initialiser le shell si nécessaire
	if err := e.initShell(); err != nil {
		return &common.CommandOutput{
			Stdout:   "",
			Stderr:   fmt.Sprintf("Erreur initialisation shell: %v", err),
			ExitCode: 1,
			Duration: time.Since(start).Milliseconds(),
		}, nil
	}

	e.shellMutex.Lock()
	defer e.shellMutex.Unlock()

	// Changer de répertoire si spécifié
	if cmdData.WorkingDir != "" && cmdData.WorkingDir != e.workingDir {
		e.shellIn.Write([]byte(fmt.Sprintf("cd %s\n", cmdData.WorkingDir)))
		e.workingDir = cmdData.WorkingDir
		time.Sleep(50 * time.Millisecond) // Attendre que le cd soit effectué
	}

	// Construire la commande complète
	fullCommand := cmdData.Command
	if len(cmdData.Args) > 0 {
		fullCommand += " " + strings.Join(cmdData.Args, " ")
	}

	// Générer un marqueur unique pour cette commande
	marker := fmt.Sprintf("__CMD_END_%d__", time.Now().UnixNano())

	// Envoyer la commande avec un marqueur de fin
	commandWithMarker := fmt.Sprintf("%s; echo '%s'\n", fullCommand, marker)
	if _, err := e.shellIn.Write([]byte(commandWithMarker)); err != nil {
		return &common.CommandOutput{
			Stdout:   "",
			Stderr:   fmt.Sprintf("Erreur envoi commande: %v", err),
			ExitCode: 1,
			Duration: time.Since(start).Milliseconds(),
		}, nil
	}

	// Lire la sortie avec timeout
	timeout := 30 * time.Second
	if cmdData.Timeout > 0 {
		timeout = time.Duration(cmdData.Timeout) * time.Second
	}

	output, err := e.readOutput(marker, timeout)
	duration := time.Since(start)

	// Analyser la sortie
	stdout := strings.TrimSpace(output)
	stderr := ""
	exitCode := 0

	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			stderr = "Commande interrompue par timeout"
			exitCode = 124 // Code d'erreur standard pour timeout
		} else {
			stderr = err.Error()
			exitCode = 1
		}
	}

	return &common.CommandOutput{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Duration: duration.Milliseconds(),
	}, nil
}

// ExecuteWithTimeout exécute une commande avec un timeout
func (e *Executor) ExecuteWithTimeout(cmdData *common.CommandData) (*common.CommandOutput, error) {
	timeout := time.Duration(cmdData.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // timeout par défaut
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return e.Execute(ctx, cmdData)
}

// GetWorkingDir retourne le répertoire de travail actuel
func (e *Executor) GetWorkingDir() string {
	return e.workingDir
}

// SetWorkingDir définit le répertoire de travail
func (e *Executor) SetWorkingDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("répertoire inexistant: %s", dir)
	}
	e.workingDir = dir
	return nil
}

// IsCommandSafe vérifie si une commande est sûre à exécuter
func (e *Executor) IsCommandSafe(command string) bool {
	// Liste des commandes dangereuses à interdire
	dangerousCommands := []string{
		"rm -rf /",
		"mkfs",
		"fdisk",
		"dd if=",
		":(){ :|:& };:",
		"sudo rm -rf",
		"shutdown",
		"reboot",
		"halt",
		"poweroff",
	}

	command = strings.ToLower(command)
	for _, dangerous := range dangerousCommands {
		if strings.Contains(command, dangerous) {
			return false
		}
	}

	return true
}

// Close ferme le shell persistant
func (e *Executor) Close() error {
	if e.shellCmd != nil && e.shellCmd.Process != nil {
		return e.shellCmd.Process.Kill()
	}
	return nil
}
