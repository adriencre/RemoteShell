package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"remoteshell/internal/common"
)

// LogManager gère les logs de l'agent et du système
type LogManager struct {
	agentLogBuffer []string
	bufferMutex    sync.RWMutex
	maxBufferSize  int
}

// NewLogManager crée un nouveau gestionnaire de logs
func NewLogManager(maxBufferSize int) *LogManager {
	if maxBufferSize <= 0 {
		maxBufferSize = 1000
	}

	return &LogManager{
		agentLogBuffer: make([]string, 0, maxBufferSize),
		maxBufferSize:  maxBufferSize,
	}
}

// AddAgentLog ajoute une entrée au buffer de logs de l'agent
func (lm *LogManager) AddAgentLog(message string) {
	lm.bufferMutex.Lock()
	defer lm.bufferMutex.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s", timestamp, message)

	lm.agentLogBuffer = append(lm.agentLogBuffer, entry)

	// Limiter la taille du buffer
	if len(lm.agentLogBuffer) > lm.maxBufferSize {
		lm.agentLogBuffer = lm.agentLogBuffer[len(lm.agentLogBuffer)-lm.maxBufferSize:]
	}
}

// ListLogSources liste les sources de logs disponibles
func (lm *LogManager) ListLogSources() ([]*common.LogSource, error) {
	sources := []*common.LogSource{
		{
			Name:        "agent",
			Type:        "agent",
			Description: "Logs de l'agent RemoteShell",
		},
	}

	// Vérifier si journalctl est disponible (systemd)
	if checkSystemd() {
		sources = append(sources, &common.LogSource{
			Name:        "systemd",
			Type:        "systemd",
			Description: "Logs système via journalctl",
		})
	}

	// Lister les fichiers de logs communs
	commonLogFiles := []string{
		"/var/log/syslog",
		"/var/log/messages",
		"/var/log/auth.log",
		"/var/log/kern.log",
		"/var/log/dmesg",
		"/var/log/apache2/error.log",
		"/var/log/nginx/error.log",
	}

	for _, logPath := range commonLogFiles {
		if _, err := os.Stat(logPath); err == nil {
			sources = append(sources, &common.LogSource{
				Name:        filepath.Base(logPath),
				Type:        "file",
				Path:        logPath,
				Description: fmt.Sprintf("Fichier log: %s", logPath),
			})
		}
	}

	return sources, nil
}

// GetLogs récupère les logs selon la requête
func (lm *LogManager) GetLogs(req *common.LogRequest) ([]*common.LogEntry, error) {
	switch req.Type {
	case "agent":
		return lm.getAgentLogs(req)
	case "systemd":
		return lm.getSystemdLogs(req)
	case "file":
		return lm.getFileLogs(req)
	default:
		return nil, fmt.Errorf("type de log non supporté: %s", req.Type)
	}
}

// getAgentLogs récupère les logs de l'agent
func (lm *LogManager) getAgentLogs(req *common.LogRequest) ([]*common.LogEntry, error) {
	lm.bufferMutex.RLock()
	defer lm.bufferMutex.RUnlock()

	lines := req.Lines
	if lines <= 0 || lines > len(lm.agentLogBuffer) {
		lines = len(lm.agentLogBuffer)
	}

	// Prendre les dernières lignes
	startIdx := len(lm.agentLogBuffer) - lines
	if startIdx < 0 {
		startIdx = 0
	}

	var entries []*common.LogEntry
	for _, line := range lm.agentLogBuffer[startIdx:] {
		entries = append(entries, &common.LogEntry{
			Source:  "agent",
			Message: line,
		})
	}

	return entries, nil
}

// getSystemdLogs récupère les logs via journalctl
func (lm *LogManager) getSystemdLogs(req *common.LogRequest) ([]*common.LogEntry, error) {
	if !checkSystemd() {
		return nil, fmt.Errorf("systemd n'est pas disponible")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Construire la commande journalctl
	args := []string{"journalctl", "--no-pager", "--output=json"}

	// Ajouter les filtres
	if req.Unit != "" {
		args = append(args, "-u", req.Unit)
	}

	if req.Priority != "" {
		args = append(args, "-p", req.Priority)
	}

	if req.Since != "" {
		args = append(args, "--since", req.Since)
	}

	if req.Until != "" {
		args = append(args, "--until", req.Until)
	}

	// Limiter le nombre de lignes
	lines := req.Lines
	if lines <= 0 {
		lines = 100
	}
	args = append(args, "-n", fmt.Sprintf("%d", lines))

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("échec de journalctl: %v", err)
	}

	// Parser la sortie JSON
	var entries []*common.LogEntry
	lines_str := strings.Split(string(output), "\n")

	for _, line := range lines_str {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Pour simplifier, on parse manuellement les champs importants
		entry := &common.LogEntry{
			Source:  "systemd",
			Message: line,
		}

		// Extraire les champs JSON basiques
		if strings.Contains(line, "__REALTIME_TIMESTAMP") {
			// Le parsing JSON complet serait mieux, mais pour simplifier...
			entry.Message = extractJournalMessage(line)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// extractJournalMessage extrait le message d'une ligne JSON de journalctl
func extractJournalMessage(jsonLine string) string {
	// Rechercher le champ MESSAGE dans la ligne JSON
	if idx := strings.Index(jsonLine, `"MESSAGE":"`); idx != -1 {
		start := idx + len(`"MESSAGE":"`)
		rest := jsonLine[start:]
		if endIdx := strings.Index(rest, `"`); endIdx != -1 {
			return rest[:endIdx]
		}
	}
	return jsonLine
}

// getFileLogs récupère les logs d'un fichier
func (lm *LogManager) getFileLogs(req *common.LogRequest) ([]*common.LogEntry, error) {
	if req.Path == "" {
		return nil, fmt.Errorf("chemin du fichier manquant")
	}

	// Vérifier que le fichier existe et est accessible
	fileInfo, err := os.Stat(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fichier inaccessible: %v", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("le chemin est un répertoire, pas un fichier")
	}

	// Sécurité: vérifier que le fichier est dans /var/log ou est un fichier log connu
	if !strings.HasPrefix(req.Path, "/var/log/") && !strings.HasSuffix(req.Path, ".log") {
		return nil, fmt.Errorf("accès non autorisé au fichier")
	}

	lines := req.Lines
	if lines <= 0 {
		lines = 100
	}

	// Utiliser tail pour lire les dernières lignes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tail", "-n", fmt.Sprintf("%d", lines), req.Path)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("échec de la lecture du fichier: %v", err)
	}

	var entries []*common.LogEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		entries = append(entries, &common.LogEntry{
			Source:  filepath.Base(req.Path),
			Message: line,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("erreur de lecture: %v", err)
	}

	return entries, nil
}

// StreamLogs stream les logs en temps réel (pour implémentation future)
func (lm *LogManager) StreamLogs(req *common.LogRequest, writer io.Writer) error {
	if !req.Follow {
		return fmt.Errorf("le streaming nécessite follow=true")
	}

	switch req.Type {
	case "systemd":
		return lm.streamSystemdLogs(req, writer)
	case "file":
		return lm.streamFileLogs(req, writer)
	default:
		return fmt.Errorf("streaming non supporté pour le type: %s", req.Type)
	}
}

// streamSystemdLogs stream les logs systemd en temps réel
func (lm *LogManager) streamSystemdLogs(req *common.LogRequest, writer io.Writer) error {
	if !checkSystemd() {
		return fmt.Errorf("systemd n'est pas disponible")
	}

	args := []string{"journalctl", "-f", "--no-pager"}

	if req.Unit != "" {
		args = append(args, "-u", req.Unit)
	}

	if req.Priority != "" {
		args = append(args, "-p", req.Priority)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = writer
	cmd.Stderr = writer

	return cmd.Run()
}

// streamFileLogs stream les logs d'un fichier en temps réel
func (lm *LogManager) streamFileLogs(req *common.LogRequest, writer io.Writer) error {
	if req.Path == "" {
		return fmt.Errorf("chemin du fichier manquant")
	}

	// Sécurité
	if !strings.HasPrefix(req.Path, "/var/log/") && !strings.HasSuffix(req.Path, ".log") {
		return fmt.Errorf("accès non autorisé au fichier")
	}

	cmd := exec.Command("tail", "-f", req.Path)
	cmd.Stdout = writer
	cmd.Stderr = writer

	return cmd.Run()
}

// GetAgentLogBuffer retourne le buffer complet des logs de l'agent
func (lm *LogManager) GetAgentLogBuffer() []string {
	lm.bufferMutex.RLock()
	defer lm.bufferMutex.RUnlock()

	// Faire une copie pour éviter les modifications concurrentes
	buffer := make([]string, len(lm.agentLogBuffer))
	copy(buffer, lm.agentLogBuffer)

	return buffer
}

// ClearAgentLogBuffer vide le buffer de logs de l'agent
func (lm *LogManager) ClearAgentLogBuffer() {
	lm.bufferMutex.Lock()
	defer lm.bufferMutex.Unlock()

	lm.agentLogBuffer = make([]string, 0, lm.maxBufferSize)
}

// LogWriter implémente io.Writer pour capturer les logs
type LogWriter struct {
	manager *LogManager
}

// NewLogWriter crée un nouveau LogWriter
func NewLogWriter(manager *LogManager) *LogWriter {
	return &LogWriter{manager: manager}
}

// Write implémente io.Writer
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	message := string(p)
	lw.manager.AddAgentLog(message)

	// Écrire aussi sur la sortie standard
	log.Print(message)

	return len(p), nil
}
