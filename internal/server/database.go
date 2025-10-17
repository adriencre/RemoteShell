package server

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database gère la base de données
type Database struct {
	db *gorm.DB
}

// AgentRecord représente un enregistrement d'agent en base
type AgentRecord struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Name        string    `json:"name"`
	LastSeen    time.Time `json:"last_seen"`
	Status      string    `json:"status"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CommandLog représente un log de commande
type CommandLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AgentID   string    `json:"agent_id"`
	Command   string    `json:"command"`
	Args      string    `json:"args"`
	WorkingDir string   `json:"working_dir"`
	ExitCode  int       `json:"exit_code"`
	Duration  int64     `json:"duration"`
	Stdout    string    `json:"stdout"`
	Stderr    string    `json:"stderr"`
	CreatedAt time.Time `json:"created_at"`
}

// FileLog représente un log de fichier
type FileLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AgentID   string    `json:"agent_id"`
	Operation string    `json:"operation"` // upload, download, delete, create_dir
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Success   bool      `json:"success"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
}

// PrinterLog représente un log d'imprimante
type PrinterLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AgentID   string    `json:"agent_id"`
	PrinterName string  `json:"printer_name"`
	Status    string    `json:"status"`
	JobCount  int       `json:"job_count"`
	CreatedAt time.Time `json:"created_at"`
}

// SystemLog représente un log système
type SystemLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AgentID   string    `json:"agent_id"`
	Hostname  string    `json:"hostname"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	Uptime    int64     `json:"uptime"`
	MemoryTotal int64   `json:"memory_total"`
	MemoryUsed  int64   `json:"memory_used"`
	DiskTotal   int64   `json:"disk_total"`
	DiskUsed    int64   `json:"disk_used"`
	CreatedAt time.Time `json:"created_at"`
}

// NewDatabase crée une nouvelle instance de base de données
func NewDatabase(dbPath string) (*Database, error) {
	// Configuration GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Connexion SQLite
	db, err := gorm.Open(sqlite.Open(dbPath), config)
	if err != nil {
		return nil, err
	}

	// Auto-migration des modèles
	if err := db.AutoMigrate(
		&AgentRecord{},
		&CommandLog{},
		&FileLog{},
		&PrinterLog{},
		&SystemLog{},
	); err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

// Close ferme la connexion à la base de données
func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SaveAgent sauvegarde ou met à jour un agent
func (d *Database) SaveAgent(agent *AgentRecord) error {
	return d.db.Save(agent).Error
}

// GetAgent récupère un agent par son ID
func (d *Database) GetAgent(agentID string) (*AgentRecord, error) {
	var agent AgentRecord
	err := d.db.Where("id = ?", agentID).First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetAgents récupère tous les agents
func (d *Database) GetAgents() ([]*AgentRecord, error) {
	var agents []*AgentRecord
	err := d.db.Find(&agents).Error
	return agents, err
}

// DeleteAgent supprime un agent
func (d *Database) DeleteAgent(agentID string) error {
	return d.db.Where("id = ?", agentID).Delete(&AgentRecord{}).Error
}

// LogCommand enregistre l'exécution d'une commande
func (d *Database) LogCommand(log *CommandLog) error {
	return d.db.Create(log).Error
}

// GetCommandLogs récupère les logs de commandes
func (d *Database) GetCommandLogs(agentID string, limit int) ([]*CommandLog, error) {
	var logs []*CommandLog
	query := d.db.Order("created_at DESC")
	
	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&logs).Error
	return logs, err
}

// LogFile enregistre une opération sur fichier
func (d *Database) LogFile(log *FileLog) error {
	return d.db.Create(log).Error
}

// GetFileLogs récupère les logs de fichiers
func (d *Database) GetFileLogs(agentID string, limit int) ([]*FileLog, error) {
	var logs []*FileLog
	query := d.db.Order("created_at DESC")
	
	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&logs).Error
	return logs, err
}

// LogPrinter enregistre le statut d'une imprimante
func (d *Database) LogPrinter(log *PrinterLog) error {
	return d.db.Create(log).Error
}

// GetPrinterLogs récupère les logs d'imprimantes
func (d *Database) GetPrinterLogs(agentID string, limit int) ([]*PrinterLog, error) {
	var logs []*PrinterLog
	query := d.db.Order("created_at DESC")
	
	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&logs).Error
	return logs, err
}

// LogSystem enregistre les informations système
func (d *Database) LogSystem(log *SystemLog) error {
	return d.db.Create(log).Error
}

// GetSystemLogs récupère les logs système
func (d *Database) GetSystemLogs(agentID string, limit int) ([]*SystemLog, error) {
	var logs []*SystemLog
	query := d.db.Order("created_at DESC")
	
	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&logs).Error
	return logs, err
}

// GetStats récupère les statistiques générales
func (d *Database) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Nombre d'agents
	var agentCount int64
	if err := d.db.Model(&AgentRecord{}).Count(&agentCount).Error; err != nil {
		return nil, err
	}
	stats["agents"] = agentCount

	// Nombre de commandes exécutées
	var commandCount int64
	if err := d.db.Model(&CommandLog{}).Count(&commandCount).Error; err != nil {
		return nil, err
	}
	stats["commands"] = commandCount

	// Nombre d'opérations sur fichiers
	var fileCount int64
	if err := d.db.Model(&FileLog{}).Count(&fileCount).Error; err != nil {
		return nil, err
	}
	stats["files"] = fileCount

	// Nombre d'imprimantes surveillées
	var printerCount int64
	if err := d.db.Model(&PrinterLog{}).Count(&printerCount).Error; err != nil {
		return nil, err
	}
	stats["printers"] = printerCount

	return stats, nil
}

// CleanupOldLogs nettoie les anciens logs
func (d *Database) CleanupOldLogs(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	
	// Nettoyer les logs de commandes
	if err := d.db.Where("created_at < ?", cutoff).Delete(&CommandLog{}).Error; err != nil {
		return err
	}
	
	// Nettoyer les logs de fichiers
	if err := d.db.Where("created_at < ?", cutoff).Delete(&FileLog{}).Error; err != nil {
		return err
	}
	
	// Nettoyer les logs d'imprimantes
	if err := d.db.Where("created_at < ?", cutoff).Delete(&PrinterLog{}).Error; err != nil {
		return err
	}
	
	// Nettoyer les logs système
	if err := d.db.Where("created_at < ?", cutoff).Delete(&SystemLog{}).Error; err != nil {
		return err
	}
	
	return nil
}


