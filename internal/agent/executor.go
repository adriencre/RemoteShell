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
		// Utiliser bash en mode interactif (-i) pour conserver l'état entre les commandes
		// Le mode interactif permet à bash de conserver le répertoire de travail et les variables
		// On utilise aussi --norc pour éviter que les fichiers de configuration interfèrent
		// mais on garde l'état grâce au mode interactif
		// Alternative: bash sans -i mais avec des variables d'environnement pour forcer l'état
		cmd = exec.Command("sudo", "-n", "bash", "-i")
		// Forcer bash à rester actif et à conserver l'état
		cmd.Env = append(os.Environ(), "SHELL=/bin/bash")
		// S'assurer que le répertoire de travail est préservé
		cmd.Dir = "" // Laisser vide pour hériter du répertoire parent, ou utiliser cmd.Dir = "/"
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

	// Construire la commande complète
	fullCommand := cmdData.Command
	if len(cmdData.Args) > 0 {
		fullCommand += " " + strings.Join(cmdData.Args, " ")
	}

	// Détecter les commandes cd pour mettre à jour le workingDir
	// IMPORTANT: Le shell persistant DOIT conserver l'état entre les commandes
	// Si cd est détecté, on l'exécute dans le shell persistant et on met à jour notre tracking
	if strings.HasPrefix(strings.TrimSpace(fullCommand), "cd ") {
		// Extraire le chemin du cd
		cdParts := strings.Fields(strings.TrimSpace(fullCommand))
		if len(cdParts) >= 2 {
			newDir := cdParts[1]
			// Gérer les cas spéciaux comme cd - ou cd ~
			switch newDir {
			case "-":
				// cd - retourne au répertoire précédent, difficile à tracker
				// Laisser le shell gérer ça - on ne met pas à jour workingDir
			case "~":
				// cd ~ va au home directory
				homeDir := os.Getenv("HOME")
				if homeDir != "" {
					e.workingDir = homeDir
				}
			default:
				// Mettre à jour le workingDir
				// Si c'est un chemin absolu, on l'utilise tel quel
				if strings.HasPrefix(newDir, "/") {
					e.workingDir = newDir
				} else {
					// Chemin relatif - on ne peut pas facilement résoudre sans connaître le pwd actuel
					// On garde le tracking approximatif, mais le shell gère le chemin réel
					// Note: Pour un vrai tracking, il faudrait exécuter 'pwd' après chaque cd relatif
					e.workingDir = newDir
				}
			}
			// Le cd sera exécuté par la commande elle-même (fullCommand contient déjà "cd /home")
			// Le shell persistant conservera cet état pour les commandes suivantes
		}
	} else if cmdData.WorkingDir != "" && cmdData.WorkingDir != "." {
		// Si un workingDir est explicitement spécifié (pas "."), l'utiliser
		// Ignorer "." car cela signifie "utiliser le répertoire courant du shell"
		if cmdData.WorkingDir != e.workingDir {
			// Envoyer le cd AVANT la commande pour que le shell soit dans le bon répertoire
			e.shellIn.Write([]byte(fmt.Sprintf("cd %s\n", cmdData.WorkingDir)))
			e.workingDir = cmdData.WorkingDir
			time.Sleep(100 * time.Millisecond) // Attendre que le cd soit effectué
		}
	}
	// Avec bash -i (mode interactif), le shell devrait conserver l'état
	// Mais on préfixe quand même avec cd pour être sûr que ça fonctionne
	// Si bash -i fonctionne correctement, on pourra retirer ce préfixage plus tard
	commandToExecute := fullCommand
	if e.workingDir != "" && !strings.HasPrefix(strings.TrimSpace(fullCommand), "cd ") {
		// Si on a un workingDir défini et que ce n'est pas un cd, préfixer avec cd
		// Cela garantit que la commande s'exécute dans le bon répertoire
		commandToExecute = fmt.Sprintf("cd %s && %s", e.workingDir, fullCommand)
	}

	// Générer un marqueur unique pour cette commande
	marker := fmt.Sprintf("__CMD_END_%d__", time.Now().UnixNano())

	// Envoyer la commande avec un marqueur de fin
	// NOTE: Si c'est un cd, la commande sera "cd /home; echo 'marker'"
	// Sinon, si workingDir est défini, ce sera "cd /home && ls; echo 'marker'"
	commandWithMarker := fmt.Sprintf("%s; echo '%s'\n", commandToExecute, marker)
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

	// Après l'exécution, si c'était un cd réussi, mettre à jour le workingDir réel
	if strings.HasPrefix(strings.TrimSpace(fullCommand), "cd ") && err == nil {
		// Essayer de récupérer le répertoire actuel avec pwd
		// Pour l'instant, on fait confiance au cd qui a été exécuté
		// Le workingDir a déjà été mis à jour avant
	}

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
