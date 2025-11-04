package main

import (
	"flag"
	"fmt"
	"log"

	"remoteshell/internal/agent"
	"remoteshell/internal/auth"
	"remoteshell/internal/common"
)

func main() {
	// Configuration des flags (sans valeurs par défaut pour ne pas écraser les env vars)
	var (
		serverHost = flag.String("server", "", "Adresse du serveur (défaut depuis REMOTESHELL_SERVER_HOST:PORT)")
		authToken  = flag.String("token", "", "Token d'authentification (défaut depuis REMOTESHELL_AUTH_TOKEN)")
		agentName  = flag.String("name", "", "Nom de l'agent (défaut depuis REMOTESHELL_AGENT_NAME)")
		agentID    = flag.String("id", "", "ID de l'agent (défaut depuis REMOTESHELL_AGENT_ID)")
		tls        = flag.Bool("tls", false, "Utiliser TLS (défaut depuis REMOTESHELL_SERVER_TLS)")
		verbose    = flag.Bool("verbose", false, "Mode verbeux")
	)
	flag.Parse()

	// Configuration par défaut
	config := common.DefaultConfig()
	config.LoadFromEnv()

	// Appliquer les flags seulement s'ils sont fournis (pour ne pas écraser les env vars)
	if *serverHost != "" {
		// Parser l'adresse serveur en utilisant les valeurs de config comme fallback
		host, port := parseServerAddress(*serverHost, config.ServerHost, config.ServerPort)
		config.ServerHost = host
		config.ServerPort = port
	}
	if *authToken != "" {
		config.AuthToken = *authToken
	}
	if *agentName != "" {
		config.AgentName = *agentName
	}
	if *agentID != "" {
		config.AgentID = *agentID
	}
	// Pour les booléens, on vérifie si le flag a été fourni explicitement
	flagTLSProvided := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "tls" {
			flagTLSProvided = true
		}
	})
	if flagTLSProvided {
		config.ServerTLS = *tls
	}
	if *verbose {
		config.LogLevel = "debug"
	}

	// Vérifier la configuration
	if config.AuthToken == "" {
		log.Fatal("Token d'authentification requis (--token ou REMOTESHELL_AUTH_TOKEN)")
	}

	// Créer le gestionnaire de tokens avec la clé depuis la config
	tokenManager := auth.NewTokenManager(config.AuthToken, "rms-agent")

	// Créer et démarrer le client agent
	client := agent.NewClient(config, tokenManager)

	log.Printf("Agent RemoteShell démarré")
	log.Printf("Serveur: %s", config.GetServerURL())
	log.Printf("Agent: %s (%s)", config.AgentName, config.AgentID)

	if err := client.Start(); err != nil {
		log.Fatalf("Erreur de démarrage de l'agent: %v", err)
	}
}

// parseServerAddress parse une adresse serveur au format host:port
// Utilise defaultHost et defaultPort comme valeurs de secours si le parsing échoue
func parseServerAddress(addr string, defaultHost string, defaultPort int) (string, int) {
	host := defaultHost
	port := defaultPort

	// Chercher le dernier ':' pour séparer host et port
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			host = addr[:i]
			if i+1 < len(addr) {
				// Parser le port
				if p, err := fmt.Sscanf(addr[i+1:], "%d", &port); err != nil || p != 1 {
					log.Printf("Port invalide '%s', utilisation du port par défaut %d", addr[i+1:], defaultPort)
					port = defaultPort
				}
			}
			break
		}
	}

	// Si pas de ':', c'est juste un host (utiliser le host fourni)
	if host == defaultHost && addr != defaultHost {
		host = addr
	}

	return host, port
}
