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
	// Configuration des flags
	var (
		serverHost = flag.String("server", "localhost:8080", "Adresse du serveur")
		authToken  = flag.String("token", "", "Token d'authentification")
		agentName  = flag.String("name", "", "Nom de l'agent")
		agentID    = flag.String("id", "", "ID de l'agent")
		tls        = flag.Bool("tls", false, "Utiliser TLS")
		verbose    = flag.Bool("verbose", false, "Mode verbeux")
	)
	flag.Parse()

	// Configuration par défaut
	config := common.DefaultConfig()
	config.LoadFromEnv()

	// Appliquer les flags
	if *serverHost != "" {
		// Parser l'adresse serveur
		host, port := parseServerAddress(*serverHost)
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
	if *tls {
		config.ServerTLS = true
	}
	if *verbose {
		config.LogLevel = "debug"
	}

	// Vérifier la configuration
	if config.AuthToken == "" {
		log.Fatal("Token d'authentification requis (--token ou REMOTESHELL_AUTH_TOKEN)")
	}

	// Créer le gestionnaire de tokens
	tokenManager := auth.NewTokenManager("", "remoteshell-agent")

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
func parseServerAddress(addr string) (string, int) {
	// Format par défaut
	host := "localhost"
	port := 8080

	// Chercher le dernier ':' pour séparer host et port
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			host = addr[:i]
			if i+1 < len(addr) {
				// Parser le port
				if p, err := fmt.Sscanf(addr[i+1:], "%d", &port); err != nil || p != 1 {
					log.Printf("Port invalide '%s', utilisation du port par défaut %d", addr[i+1:], port)
				}
			}
			break
		}
	}

	// Si pas de ':', c'est juste un host
	if host == addr {
		host = addr
	}

	return host, port
}
