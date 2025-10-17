package common

import (
	"encoding/json"
	"time"
)

// MessageType définit les types de messages WebSocket
type MessageType string

const (
	// Messages d'authentification
	MessageTypeAuth        MessageType = "auth"
	MessageTypeAuthSuccess MessageType = "auth_success"
	MessageTypeAuthError   MessageType = "auth_error"

	// Messages de commande
	MessageTypeCommand     MessageType = "command"
	MessageTypeCommandExec MessageType = "command_exec"
	MessageTypeCommandOut  MessageType = "command_out"
	MessageTypeCommandErr  MessageType = "command_err"
	MessageTypeCommandDone MessageType = "command_done"

	// Messages de fichier
	MessageTypeFileUpload    MessageType = "file_upload"
	MessageTypeFileDownload  MessageType = "file_download"
	MessageTypeFileList      MessageType = "file_list"
	MessageTypeFileChunk     MessageType = "file_chunk"
	MessageTypeFileComplete  MessageType = "file_complete"
	MessageTypeFileError     MessageType = "file_error"
	MessageTypeFileDelete    MessageType = "file_delete"
	MessageTypeFileCreateDir MessageType = "file_create_dir"

	// Messages de monitoring
	MessageTypePrinterStatus MessageType = "printer_status"
	MessageTypeSystemInfo    MessageType = "system_info"
	MessageTypeHeartbeat     MessageType = "heartbeat"

	// Messages de gestion des services
	MessageTypeServiceList   MessageType = "service_list"
	MessageTypeServiceStatus MessageType = "service_status"
	MessageTypeServiceAction MessageType = "service_action"
	MessageTypeServiceResult MessageType = "service_result"

	// Messages de gestion des logs
	MessageTypeLogList    MessageType = "log_list"
	MessageTypeLogContent MessageType = "log_content"
	MessageTypeLogStream  MessageType = "log_stream"

	// Messages d'erreur
	MessageTypeError MessageType = "error"
)

// Message représente un message WebSocket
type Message struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	AgentID   string      `json:"agent_id,omitempty"`
}

// AuthData contient les données d'authentification
type AuthData struct {
	Token string `json:"token"`
}

// CommandData contient les données d'exécution de commande
type CommandData struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Timeout    int               `json:"timeout,omitempty"` // en secondes
}

// CommandOutput contient la sortie d'une commande
type CommandOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Duration int64  `json:"duration"` // en millisecondes
}

// FileData contient les informations de fichier
type FileData struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Mode     uint32    `json:"mode"`
	Modified time.Time `json:"modified"`
	IsDir    bool      `json:"is_dir"`
}

// FileChunk contient un chunk de fichier
type FileChunk struct {
	Path     string `json:"path"`
	Offset   int64  `json:"offset"`
	Data     []byte `json:"data"`
	Checksum string `json:"checksum,omitempty"`
	IsLast   bool   `json:"is_last"`
}

// PrinterInfo contient les informations d'une imprimante
type PrinterInfo struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Description string     `json:"description"`
	Location    string     `json:"location"`
	URI         string     `json:"uri"`
	IsDefault   bool       `json:"is_default"`
	Jobs        []PrintJob `json:"jobs,omitempty"`
}

// PrintJob contient les informations d'un travail d'impression
type PrintJob struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	User     string    `json:"user"`
	Size     int64     `json:"size"`
	Status   string    `json:"status"`
	Priority int       `json:"priority"`
	Created  time.Time `json:"created"`
}

// SystemInfo contient les informations système
type SystemInfo struct {
	Hostname    string    `json:"hostname"`
	OS          string    `json:"os"`
	Arch        string    `json:"arch"`
	Uptime      int64     `json:"uptime"` // en secondes
	LoadAvg     []float64 `json:"load_avg,omitempty"`
	MemoryTotal int64     `json:"memory_total"`
	MemoryUsed  int64     `json:"memory_used"`
	DiskTotal   int64     `json:"disk_total"`
	DiskUsed    int64     `json:"disk_used"`
}

// ErrorData contient les informations d'erreur
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ServiceInfo contient les informations d'un service
type ServiceInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "systemd" ou "docker"
	Status      string `json:"status"`
	State       string `json:"state"` // active, inactive, failed, etc.
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
	// Pour Docker
	ContainerID string `json:"container_id,omitempty"`
	Image       string `json:"image,omitempty"`
}

// ServiceAction contient une action à effectuer sur un service
type ServiceAction struct {
	Name   string `json:"name"`
	Type   string `json:"type"`   // "systemd" ou "docker"
	Action string `json:"action"` // "start", "stop", "restart", "enable", "disable"
}

// ServiceResult contient le résultat d'une action sur un service
type ServiceResult struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Action  string `json:"action"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Output  string `json:"output,omitempty"`
}

// LogSource contient les informations d'une source de logs
type LogSource struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "agent", "systemd", "file"
	Path        string `json:"path,omitempty"`
	Description string `json:"description,omitempty"`
}

// LogRequest contient une demande de logs
type LogRequest struct {
	Source   string            `json:"source"`
	Type     string            `json:"type"`
	Lines    int               `json:"lines,omitempty"`    // nombre de lignes (tail)
	Follow   bool              `json:"follow,omitempty"`   // streaming en temps réel
	Filters  map[string]string `json:"filters,omitempty"`  // filtres (service, priority, since, until)
	Path     string            `json:"path,omitempty"`     // pour les fichiers logs
	Unit     string            `json:"unit,omitempty"`     // pour journalctl
	Priority string            `json:"priority,omitempty"` // pour journalctl
	Since    string            `json:"since,omitempty"`    // pour journalctl
	Until    string            `json:"until,omitempty"`    // pour journalctl
}

// LogEntry contient une entrée de log
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level,omitempty"`
	Source    string `json:"source,omitempty"`
	Message   string `json:"message"`
	Unit      string `json:"unit,omitempty"`
}

// NewMessage crée un nouveau message
func NewMessage(msgType MessageType, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewMessageWithID crée un nouveau message avec ID
func NewMessageWithID(msgType MessageType, id string, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		ID:        id,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// ToJSON convertit le message en JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON crée un message à partir de JSON
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// IsValid vérifie si le message est valide
func (m *Message) IsValid() bool {
	return m.Type != "" && m.Timestamp.IsZero() == false
}
