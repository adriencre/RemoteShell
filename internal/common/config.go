package common

import (
	"os"
	"strconv"
	"time"
)

// Config contient la configuration de l'application
type Config struct {
	// Configuration serveur
	ServerHost string
	ServerPort int
	ServerTLS  bool
	CertFile   string
	KeyFile    string

	// Configuration agent
	AgentID       string
	AgentName     string
	ReconnectDelay time.Duration
	HeartbeatInterval time.Duration

	// Configuration authentification
	AuthToken string
	TokenFile string

	// Configuration base de données
	DatabasePath string

	// Configuration logs
	LogLevel string
	LogFile  string

	// Configuration fichiers
	MaxFileSize int64
	ChunkSize   int
}

// DefaultConfig retourne une configuration par défaut
func DefaultConfig() *Config {
	return &Config{
		ServerHost:        "localhost",
		ServerPort:        8080,
		ServerTLS:         false,
		ReconnectDelay:    5 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		DatabasePath:      "remoteshell.db",
		LogLevel:          "info",
		MaxFileSize:       100 * 1024 * 1024, // 100MB
		ChunkSize:         64 * 1024,         // 64KB
		AuthToken:         "default-secret-key-change-in-production-12345", // Clé par défaut
	}
}

// LoadFromEnv charge la configuration depuis les variables d'environnement
func (c *Config) LoadFromEnv() {
	if host := os.Getenv("REMOTESHELL_SERVER_HOST"); host != "" {
		c.ServerHost = host
	}
	if port := os.Getenv("REMOTESHELL_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.ServerPort = p
		}
	}
	if tls := os.Getenv("REMOTESHELL_SERVER_TLS"); tls == "true" {
		c.ServerTLS = true
	}
	if cert := os.Getenv("REMOTESHELL_CERT_FILE"); cert != "" {
		c.CertFile = cert
	}
	if key := os.Getenv("REMOTESHELL_KEY_FILE"); key != "" {
		c.KeyFile = key
	}
	if agentID := os.Getenv("REMOTESHELL_AGENT_ID"); agentID != "" {
		c.AgentID = agentID
	}
	if agentName := os.Getenv("REMOTESHELL_AGENT_NAME"); agentName != "" {
		c.AgentName = agentName
	}
	if token := os.Getenv("REMOTESHELL_AUTH_TOKEN"); token != "" {
		c.AuthToken = token
	}
	if tokenFile := os.Getenv("REMOTESHELL_TOKEN_FILE"); tokenFile != "" {
		c.TokenFile = tokenFile
	}
	if dbPath := os.Getenv("REMOTESHELL_DB_PATH"); dbPath != "" {
		c.DatabasePath = dbPath
	}
	if logLevel := os.Getenv("REMOTESHELL_LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}
	if logFile := os.Getenv("REMOTESHELL_LOG_FILE"); logFile != "" {
		c.LogFile = logFile
	}
}

// GetServerURL retourne l'URL du serveur
func (c *Config) GetServerURL() string {
	protocol := "ws"
	if c.ServerTLS {
		protocol = "wss"
	}
	return protocol + "://" + c.ServerHost + ":" + strconv.Itoa(c.ServerPort)
}

// GetAPIURL retourne l'URL de l'API REST
func (c *Config) GetAPIURL() string {
	protocol := "http"
	if c.ServerTLS {
		protocol = "https"
	}
	return protocol + "://" + c.ServerHost + ":" + strconv.Itoa(c.ServerPort)
}
