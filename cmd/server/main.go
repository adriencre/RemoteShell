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
	// Configuration des flags (sans valeurs par défaut pour ne pas écraser les env vars)
	var (
		host       = flag.String("host", "", "Adresse d'écoute (défaut depuis REMOTESHELL_SERVER_HOST)")
		port       = flag.Int("port", 0, "Port d'écoute (défaut depuis REMOTESHELL_SERVER_PORT)")
		tls        = flag.Bool("tls", false, "Utiliser TLS (défaut depuis REMOTESHELL_SERVER_TLS)")
		certFile   = flag.String("cert", "", "Fichier de certificat (défaut depuis REMOTESHELL_CERT_FILE)")
		keyFile    = flag.String("key", "", "Fichier de clé privée (défaut depuis REMOTESHELL_KEY_FILE)")
		secretKey  = flag.String("secret", "", "Clé secrète JWT (défaut depuis REMOTESHELL_AUTH_TOKEN)")
		dbPath     = flag.String("db", "", "Chemin de la base de données (défaut depuis REMOTESHELL_DB_PATH)")
		verbose    = flag.Bool("verbose", false, "Mode verbeux")
	)
	flag.Parse()

	// Configuration par défaut
	config := common.DefaultConfig()
	config.LoadFromEnv()

	// Appliquer les flags seulement s'ils sont fournis (pour ne pas écraser les env vars)
	if *host != "" {
		config.ServerHost = *host
	}
	if *port != 0 {
		config.ServerPort = *port
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
	if *certFile != "" {
		config.CertFile = *certFile
	}
	if *keyFile != "" {
		config.KeyFile = *keyFile
	}
	if *dbPath != "" {
		config.DatabasePath = *dbPath
	}
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
	apiServer := server.NewAPIServer(hub, tokenManager, config.AuthToken, config)

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
