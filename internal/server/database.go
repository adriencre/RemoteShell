package server

import (
	"fmt"
	"strings"
	"time"

	"remoteshell/internal/common"

	"github.com/glebarez/sqlite" // Driver SQLite pur Go (pas besoin de CGO)
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// contains vérifie si une chaîne contient une sous-chaîne
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Database gère la base de données
type Database struct {
	db *gorm.DB
}

// User représente un utilisateur SSO
type User struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Email     string    `gorm:"uniqueIndex;type:varchar(191)" json:"email"`
	Name      string    `gorm:"type:varchar(255)" json:"name"`
	Username  string    `gorm:"type:varchar(255)" json:"username"`
	Role      string    `gorm:"type:varchar(50)" json:"role"`
	Groups    string    `gorm:"type:text" json:"groups"`
	CreatedAt time.Time `gorm:"type:datetime(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime(3)" json:"updated_at"`
}

func (User) TableName() string {
	return "rms_users"
}

// AgentRecord représente un enregistrement d'agent en base
type AgentRecord struct {
	ID        string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name      string    `gorm:"type:varchar(255)" json:"name"`
	LastSeen  time.Time `gorm:"type:datetime(3)" json:"last_seen"`
	Status    string    `gorm:"type:varchar(50)" json:"status"`
	IPAddress string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent string    `gorm:"type:varchar(500)" json:"user_agent"`
	Franchise string    `gorm:"type:varchar(100)" json:"franchise"`
	Category  string    `gorm:"type:varchar(100)" json:"category"`
	UserID    string    `gorm:"type:varchar(191);index" json:"user_id"`
	CreatedAt time.Time `gorm:"type:datetime(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime(3)" json:"updated_at"`
}

func (AgentRecord) TableName() string {
	return "rms_agents"
}

// CommandLog représente un log de commande
type CommandLog struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID    string    `gorm:"type:varchar(191);index" json:"agent_id"`
	Command    string    `gorm:"type:varchar(500)" json:"command"`
	Args       string    `gorm:"type:text" json:"args"`
	WorkingDir string    `gorm:"type:varchar(500)" json:"working_dir"`
	ExitCode   int       `json:"exit_code"`
	Duration   int64     `json:"duration"`
	Stdout     string    `gorm:"type:longtext" json:"stdout"`
	Stderr     string    `gorm:"type:longtext" json:"stderr"`
	CreatedAt  time.Time `gorm:"type:datetime(3);index" json:"created_at"`
}

func (CommandLog) TableName() string {
	return "rms_command_logs"
}

// FileLog représente un log de fichier
type FileLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID   string    `gorm:"type:varchar(191);index" json:"agent_id"`
	Operation string    `gorm:"type:varchar(50)" json:"operation"`
	Path      string    `gorm:"type:varchar(1000)" json:"path"`
	Size      int64     `json:"size"`
	Success   bool      `json:"success"`
	Error     string    `gorm:"type:text" json:"error"`
	CreatedAt time.Time `gorm:"type:datetime(3);index" json:"created_at"`
}

func (FileLog) TableName() string {
	return "rms_file_logs"
}

// PrinterLog représente un log d'imprimante
type PrinterLog struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID     string    `gorm:"type:varchar(191);index" json:"agent_id"`
	PrinterName string    `gorm:"type:varchar(255)" json:"printer_name"`
	Status      string    `gorm:"type:varchar(100)" json:"status"`
	JobCount    int       `json:"job_count"`
	CreatedAt   time.Time `gorm:"type:datetime(3);index" json:"created_at"`
}

func (PrinterLog) TableName() string {
	return "rms_printer_logs"
}

// SystemLog représente un log système
type SystemLog struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID     string    `gorm:"type:varchar(191);index" json:"agent_id"`
	Hostname    string    `gorm:"type:varchar(255)" json:"hostname"`
	OS          string    `gorm:"type:varchar(100)" json:"os"`
	Arch        string    `gorm:"type:varchar(50)" json:"arch"`
	Uptime      int64     `json:"uptime"`
	MemoryTotal int64     `json:"memory_total"`
	MemoryUsed  int64     `json:"memory_used"`
	DiskTotal   int64     `json:"disk_total"`
	DiskUsed    int64     `json:"disk_used"`
	CreatedAt   time.Time `gorm:"type:datetime(3);index" json:"created_at"`
}

func (SystemLog) TableName() string {
	return "rms_system_logs"
}

// NewDatabase crée une nouvelle instance de base de données
// Si MySQL est configuré, utilise MySQL, sinon utilise SQLite
func NewDatabase(config *common.Config) (*Database, error) {
	// Configuration GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	var db *gorm.DB
	var err error

	// Vérifier si MySQL est activé
	if config.MySQLEnabled && config.MySQLHost != "" {
		// Construire la DSN MySQL
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.MySQLUser,
			config.MySQLPassword,
			config.MySQLHost,
			config.MySQLPort,
			config.MySQLDatabase,
		)

		// Connexion MySQL
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
		if err != nil {
			return nil, fmt.Errorf("erreur de connexion MySQL: %w", err)
		}
	} else {
		// Fallback sur SQLite
		dbPath := config.DatabasePath
		if dbPath == "" {
			dbPath = "./remoteshell.db"
		}
		db, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
		if err != nil {
			return nil, fmt.Errorf("erreur de connexion SQLite: %w", err)
		}
	}

	// Auto-migration des modèles
	// Ignorer les erreurs de suppression d'index/clé qui n'existent pas (lors des migrations)
	if err := db.AutoMigrate(
		&User{},
		&AgentRecord{},
		&CommandLog{},
		&FileLog{},
		&PrinterLog{},
		&SystemLog{},
	); err != nil {
		// Les erreurs de type "Can't DROP" sont normales lors des migrations
		// On les ignore car les tables sont déjà créées avec les bons index
		errMsg := err.Error()
		if !contains(errMsg, "Can't DROP") && !contains(errMsg, "check that column/key exists") {
			return nil, fmt.Errorf("erreur de migration: %w", err)
		}
		// Log mais ne pas bloquer si c'est juste une erreur de DROP
		fmt.Printf("[INFO] Migration: erreur ignorée (index/clé déjà supprimé): %v\n", err)
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

// SaveUser sauvegarde ou met à jour un utilisateur
func (d *Database) SaveUser(user *User) error {
	return d.db.Save(user).Error
}

// GetUser récupère un utilisateur par son ID
func (d *Database) GetUser(userID string) (*User, error) {
	var user User
	err := d.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail récupère un utilisateur par son email
func (d *Database) GetUserByEmail(email string) (*User, error) {
	var user User
	err := d.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUsers récupère tous les utilisateurs
func (d *Database) GetUsers() ([]*User, error) {
	var users []*User
	err := d.db.Find(&users).Error
	return users, err
}

// GetAgentsByUser récupère les agents d'un utilisateur
func (d *Database) GetAgentsByUser(userID string) ([]*AgentRecord, error) {
	var agents []*AgentRecord
	err := d.db.Where("user_id = ?", userID).Find(&agents).Error
	return agents, err
}
