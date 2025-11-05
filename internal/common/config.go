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

	// Configuration OAuth2/Authentik
	OAuth2Enabled      bool
	OAuth2Provider     string // "authentik"
	OAuth2ClientID     string
	OAuth2ClientSecret string
	OAuth2BaseURL      string // URL d'Authentik (ex: https://auth.example.com)
	OAuth2RedirectURL  string // URL de callback
	OAuth2Scopes       string // Scopes séparés par des virgules

	// Configuration base de données
	DatabasePath string // Pour SQLite (legacy)
	
	// Configuration MySQL
	MySQLHost     string
	MySQLPort     int
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string
	MySQLEnabled  bool

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
	// Configuration MySQL
	if enabled := os.Getenv("REMOTESHELL_MYSQL_ENABLED"); enabled == "true" {
		c.MySQLEnabled = true
	}
	if host := os.Getenv("REMOTESHELL_MYSQL_HOST"); host != "" {
		c.MySQLHost = host
	}
	if port := os.Getenv("REMOTESHELL_MYSQL_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.MySQLPort = p
		}
	}
	if user := os.Getenv("REMOTESHELL_MYSQL_USER"); user != "" {
		c.MySQLUser = user
	}
	if password := os.Getenv("REMOTESHELL_MYSQL_PASSWORD"); password != "" {
		c.MySQLPassword = password
	}
	if database := os.Getenv("REMOTESHELL_MYSQL_DATABASE"); database != "" {
		c.MySQLDatabase = database
	}
	if logLevel := os.Getenv("REMOTESHELL_LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}
	if logFile := os.Getenv("REMOTESHELL_LOG_FILE"); logFile != "" {
		c.LogFile = logFile
	}
	if reconnectDelay := os.Getenv("REMOTESHELL_RECONNECT_DELAY"); reconnectDelay != "" {
		if d, err := time.ParseDuration(reconnectDelay); err == nil {
			c.ReconnectDelay = d
		}
	}
	if heartbeatInterval := os.Getenv("REMOTESHELL_HEARTBEAT_INTERVAL"); heartbeatInterval != "" {
		if d, err := time.ParseDuration(heartbeatInterval); err == nil {
			c.HeartbeatInterval = d
		}
	}
	if maxFileSize := os.Getenv("REMOTESHELL_MAX_FILE_SIZE"); maxFileSize != "" {
		if size, err := strconv.ParseInt(maxFileSize, 10, 64); err == nil {
			c.MaxFileSize = size
		}
	}
	if chunkSize := os.Getenv("REMOTESHELL_CHUNK_SIZE"); chunkSize != "" {
		if size, err := strconv.Atoi(chunkSize); err == nil {
			c.ChunkSize = size
		}
	}
	// Configuration OAuth2/Authentik
	if enabled := os.Getenv("REMOTESHELL_OAUTH2_ENABLED"); enabled == "true" {
		c.OAuth2Enabled = true
	}
	if provider := os.Getenv("REMOTESHELL_OAUTH2_PROVIDER"); provider != "" {
		c.OAuth2Provider = provider
	}
	if clientID := os.Getenv("REMOTESHELL_OAUTH2_CLIENT_ID"); clientID != "" {
		c.OAuth2ClientID = clientID
	}
	if clientSecret := os.Getenv("REMOTESHELL_OAUTH2_CLIENT_SECRET"); clientSecret != "" {
		c.OAuth2ClientSecret = clientSecret
	}
	if baseURL := os.Getenv("REMOTESHELL_OAUTH2_BASE_URL"); baseURL != "" {
		c.OAuth2BaseURL = baseURL
	}
	if redirectURL := os.Getenv("REMOTESHELL_OAUTH2_REDIRECT_URL"); redirectURL != "" {
		c.OAuth2RedirectURL = redirectURL
	}
	if scopes := os.Getenv("REMOTESHELL_OAUTH2_SCOPES"); scopes != "" {
		c.OAuth2Scopes = scopes
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
