package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"remoteshell/internal/auth"
	"remoteshell/internal/common"
	"remoteshell/internal/server"
)

func main() {
	// Configuration des flags
	var (
		host       = flag.String("host", "0.0.0.0", "Adresse d'écoute")
		port       = flag.Int("port", 8080, "Port d'écoute")
		tls        = flag.Bool("tls", false, "Utiliser TLS")
		certFile   = flag.String("cert", "", "Fichier de certificat")
		keyFile    = flag.String("key", "", "Fichier de clé privée")
		secretKey  = flag.String("secret", "", "Clé secrète JWT")
		dbPath     = flag.String("db", "remoteshell.db", "Chemin de la base de données")
		verbose    = flag.Bool("verbose", false, "Mode verbeux")
	)
	flag.Parse()

	// Configuration par défaut
	config := common.DefaultConfig()
	config.LoadFromEnv()

	// Appliquer les flags
	config.ServerHost = *host
	config.ServerPort = *port
	config.ServerTLS = *tls
	config.CertFile = *certFile
	config.KeyFile = *keyFile
	config.DatabasePath = *dbPath
	if *secretKey != "" {
		config.AuthToken = *secretKey
	}
	if *verbose {
		config.LogLevel = "debug"
	}

	// Créer le gestionnaire de tokens
	tokenManager := auth.NewTokenManager(config.AuthToken, "remoteshell-server")

	// Créer le hub
	hub := server.NewHub()

	// Créer le serveur API
	apiServer := server.NewAPIServer(hub, tokenManager)

	// Démarrer le hub
	go hub.Run()

	// Démarrer le serveur WebSocket
	go func() {
		log.Printf("Serveur WebSocket démarré sur %s:%d", config.ServerHost, config.ServerPort)
		if err := apiServer.Run(config.ServerHost + ":" + strconv.Itoa(config.ServerPort)); err != nil {
			log.Fatalf("Erreur de démarrage du serveur: %v", err)
		}
	}()

	// Gérer les signaux système
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Serveur RemoteShell démarré")
	log.Printf("Interface web: http://%s:%d", config.ServerHost, config.ServerPort)
	log.Printf("WebSocket: ws://%s:%d/ws", config.ServerHost, config.ServerPort)
	log.Printf("API: http://%s:%d/api", config.ServerHost, config.ServerPort)

	// Attendre le signal d'arrêt
	<-sigChan
	log.Println("Signal d'arrêt reçu, fermeture du serveur...")
}
