package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"remoteshell/internal/auth"
	"remoteshell/internal/common"

	"github.com/gin-gonic/gin"
)

// APIServer gère l'API REST
type APIServer struct {
	hub          *Hub
	tokenManager *auth.TokenManager
	router       *gin.Engine
	wsServer     *WebSocketServer
	config       *common.Config
	oauth2Config *auth.OAuth2Config
	db           *Database
}

// NewAPIServer crée un nouveau serveur API
func NewAPIServer(hub *Hub, tokenManager *auth.TokenManager, authToken string, config *common.Config, db *Database) *APIServer {
	api := &APIServer{
		hub:          hub,
		tokenManager: tokenManager,
		router:       gin.Default(),
		wsServer:     NewWebSocketServer(hub, tokenManager, authToken),
		config:       config,
		db:           db,
	}

	// Initialiser OAuth2 si configuré
	if config.OAuth2Enabled && config.OAuth2BaseURL != "" && config.OAuth2ClientID != "" {
		scopes := []string{"openid", "profile", "email"}
		if config.OAuth2Scopes != "" {
			// Parser les scopes séparés par des virgules
			scopeList := strings.Split(config.OAuth2Scopes, ",")
			for i, s := range scopeList {
				scopeList[i] = strings.TrimSpace(s)
			}
			scopes = scopeList
		}
		api.oauth2Config = auth.NewOAuth2Config(
			config.OAuth2Provider,
			config.OAuth2ClientID,
			config.OAuth2ClientSecret,
			config.OAuth2BaseURL,
			config.OAuth2RedirectURL,
			scopes,
		)
	}

	api.setupRoutes()
	return api
}

// setupRoutes configure les routes de l'API
func (api *APIServer) setupRoutes() {
	// Middleware CORS
	api.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Routes publiques
	api.router.GET("/health", api.healthCheck)
	api.router.GET("/api/database/info", api.getDatabaseInfo)
	api.router.GET("/download/agent", api.downloadAgent)
	api.router.GET("/download/install-agent.sh", api.downloadInstallScript)

	// Routes OAuth2/Authentik
	if api.oauth2Config != nil {
		api.router.GET("/api/auth/oauth2/login", api.oauth2Login)
		api.router.GET("/api/auth/oauth2/callback", api.oauth2Callback)
		api.router.GET("/api/auth/oauth2/config", api.oauth2ConfigEndpoint)
	} else {
		// Route de login classique seulement si OAuth2 n'est pas activé
		api.router.POST("/api/auth/login", api.login)
	}

	// WebSocket pour les agents (sans authentification)
	api.router.GET("/ws", api.wsServer.HandleWebSocket)

	// Routes protégées
	protected := api.router.Group("/api")
	protected.Use(auth.AuthMiddleware(api.tokenManager))
	{
		// Agents
		protected.GET("/agents", api.getAgents)
		protected.GET("/agents/:id", api.getAgent)
		protected.PUT("/agents/:id/metadata", api.updateAgentMetadata)
		protected.POST("/agents/:id/exec", api.executeCommand)
		protected.GET("/agents/:id/printers", api.getAgentPrinters)
		protected.GET("/agents/:id/system", api.getAgentSystem)

		// Fichiers
		protected.GET("/agents/:id/files", api.listFiles)
		protected.POST("/agents/:id/files/upload", api.uploadFile)
		protected.GET("/agents/:id/files/download", api.downloadFile)
		protected.DELETE("/agents/:id/files", api.deleteFile)
		protected.POST("/agents/:id/files/dir", api.createDirectory)

		// Services
		protected.GET("/agents/:id/services", api.listServices)
		protected.GET("/agents/:id/services/:service/status", api.getServiceStatus)
		protected.POST("/agents/:id/services/:service/:action", api.executeServiceAction)

		// Logs
		protected.GET("/agents/:id/logs", api.listLogSources)
		protected.GET("/agents/:id/logs/:source", api.getLogContent)

	}

	// Servir les fichiers statiques (interface web)
	api.router.Static("/assets", "./build/web/assets")
	api.router.StaticFile("/", "./build/web/index.html")

	// Rediriger toutes les routes non-API vers index.html (pour React Router)
	api.router.NoRoute(func(c *gin.Context) {
		// Si ce n'est pas une route API, servir index.html
		if !strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.File("./build/web/index.html")
		} else {
			c.JSON(404, gin.H{"error": "Route non trouvée"})
		}
	})
}

// Run démarre le serveur API
func (api *APIServer) Run(addr string) error {
	return api.router.Run(addr)
}

// healthCheck vérifie l'état du serveur
func (api *APIServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
		"agents":    api.hub.GetAgentCount(),
	})
}

// getDatabaseInfo retourne les informations sur la base de données utilisée
func (api *APIServer) getDatabaseInfo(c *gin.Context) {
	info := gin.H{
		"type": "unknown",
	}

	if api.config.MySQLEnabled && api.config.MySQLHost != "" {
		info["type"] = "MySQL"
		info["host"] = api.config.MySQLHost
		info["port"] = api.config.MySQLPort
		info["database"] = api.config.MySQLDatabase
		info["user"] = api.config.MySQLUser
		// Vérifier les tables
		stats, err := api.db.GetStats()
		if err == nil {
			info["stats"] = stats
		}
	} else {
		info["type"] = "SQLite"
		info["path"] = api.config.DatabasePath
		if info["path"] == "" {
			info["path"] = "./remoteshell.db"
		}
		stats, err := api.db.GetStats()
		if err == nil {
			info["stats"] = stats
		}
	}

	c.JSON(http.StatusOK, info)
}

// downloadAgent sert le fichier binaire de l'agent pour téléchargement
func (api *APIServer) downloadAgent(c *gin.Context) {
	// Récupérer les paramètres OS et architecture depuis la requête
	osParam := c.Query("os")
	archParam := c.Query("arch")

	// Si les paramètres ne sont pas fournis, essayer de détecter depuis User-Agent
	if osParam == "" || archParam == "" {
		userAgent := c.GetHeader("User-Agent")
		log.Printf("[API] downloadAgent - Paramètres OS/arch manquants, User-Agent: %s", userAgent)
		// Fallback: utiliser linux/amd64 par défaut
		if osParam == "" {
			osParam = "linux"
		}
		if archParam == "" {
			archParam = "amd64"
		}
		log.Printf("[API] downloadAgent - Utilisation des valeurs par défaut: os=%s, arch=%s", osParam, archParam)
	}

	// Normaliser les valeurs
	osParam = strings.ToLower(osParam)
	archParam = strings.ToLower(archParam)

	// Déterminer l'extension du fichier
	ext := ""
	if osParam == "windows" {
		ext = ".exe"
	}

	// Construire le nom du fichier attendu
	expectedFileName := fmt.Sprintf("agent-%s-%s%s", osParam, archParam, ext)

	// Obtenir le répertoire de travail actuel
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	log.Printf("[API] downloadAgent - Répertoire de travail: %s", workDir)

	// Chercher le fichier dans plusieurs emplacements possibles
	// PRIORITÉ: Chercher d'abord le binaire spécifique à l'OS/arch demandé
	possiblePaths := []string{
		// Chemins relatifs au répertoire de travail
		fmt.Sprintf("./build/%s", expectedFileName),
		fmt.Sprintf("./build/web/%s", expectedFileName),
		fmt.Sprintf("./%s", expectedFileName),
		fmt.Sprintf("./web/public/%s", expectedFileName),
		// Chemins absolus basés sur le répertoire de travail
		fmt.Sprintf("%s/build/%s", workDir, expectedFileName),
		fmt.Sprintf("%s/build/web/%s", workDir, expectedFileName),
		// Fallback: chercher sans préfixe "agent-"
		fmt.Sprintf("./build/rms-agent-%s-%s%s", osParam, archParam, ext),
		fmt.Sprintf("./build/web/rms-agent-%s-%s%s", osParam, archParam, ext),
		fmt.Sprintf("%s/build/rms-agent-%s-%s%s", workDir, osParam, archParam, ext),
	}
	
	// NE PAS inclure les fallbacks génériques (rms-agent sans suffixe) car ils peuvent
	// être pour une autre architecture et causer des problèmes d'exécution

	var agentPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			agentPath = path
			log.Printf("[API] downloadAgent - Binaire spécifique trouvé: %s (demandé: os=%s, arch=%s)", agentPath, osParam, archParam)
			break
		}
	}

	if agentPath == "" {
		log.Printf("[API] downloadAgent - Binaire spécifique non trouvé pour os=%s, arch=%s", osParam, archParam)
		log.Printf("[API] downloadAgent - Chemins testés: %v", possiblePaths)
		
		// Vérifier quels binaires sont disponibles
		availableBinaries := []string{}
		// Chercher dans plusieurs emplacements possibles
		buildDirs := []string{"./build", fmt.Sprintf("%s/build", workDir)}
		for _, buildDir := range buildDirs {
			if entries, err := os.ReadDir(buildDir); err == nil {
				for _, entry := range entries {
					if !entry.IsDir() && (strings.HasPrefix(entry.Name(), "agent-") || entry.Name() == "rms-agent") {
						availableBinaries = append(availableBinaries, entry.Name())
					}
				}
				// Si on a trouvé des binaires, arrêter la recherche
				if len(availableBinaries) > 0 {
					break
				}
			}
		}
		
		errorMsg := fmt.Sprintf("Le binaire agent-%s-%s%s n'est pas disponible.", osParam, archParam, ext)
		if len(availableBinaries) > 0 {
			errorMsg += fmt.Sprintf(" Binaires disponibles: %v.", availableBinaries)
		}
		errorMsg += " Pour générer les binaires multi-plateformes, exécutez: ./scripts/build.sh ou make build-all"
		
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "fichier agent non trouvé",
			"os":        osParam,
			"arch":      archParam,
			"expected":  expectedFileName,
			"available": availableBinaries,
			"hint":      errorMsg,
		})
		return
	}

	// Déterminer le nom du fichier pour le téléchargement
	downloadFileName := "rms-agent"
	if osParam == "windows" {
		downloadFileName = "rms-agent.exe"
	}

	// Vérifier que le nom du fichier correspond bien à l'architecture demandée
	// (sécurité supplémentaire pour éviter de servir un mauvais binaire)
	if !strings.Contains(agentPath, fmt.Sprintf("%s-%s", osParam, archParam)) && 
	   !strings.Contains(agentPath, fmt.Sprintf("%s-%s%s", osParam, archParam, ext)) {
		log.Printf("[API] downloadAgent - ATTENTION: Le nom du fichier trouvé (%s) ne correspond pas à la demande (os=%s, arch=%s)", agentPath, osParam, archParam)
		// Ne pas servir le fichier si le nom ne correspond pas
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "fichier agent incompatible",
			"os":    osParam,
			"arch":  archParam,
			"found": agentPath,
			"hint":  fmt.Sprintf("Le fichier trouvé ne correspond pas à l'architecture demandée (%s/%s)", osParam, archParam),
		})
		return
	}

	// Définir les en-têtes pour forcer le téléchargement
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", downloadFileName))
	c.Header("Content-Type", "application/octet-stream")
	// Ajouter un header personnalisé pour indiquer quel binaire est servi
	c.Header("X-Agent-Binary", expectedFileName)
	c.Header("X-Agent-OS", osParam)
	c.Header("X-Agent-Arch", archParam)
	
	log.Printf("[API] downloadAgent - ✅ Servir le binaire spécifique: %s (os=%s, arch=%s, fichier=%s)", agentPath, osParam, archParam, expectedFileName)
	
	// Servir le fichier
	c.File(agentPath)
}

// downloadInstallScript sert le script d'installation de l'agent
func (api *APIServer) downloadInstallScript(c *gin.Context) {
	// Obtenir le répertoire de travail actuel
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	// Chercher le script dans plusieurs emplacements possibles
	possiblePaths := []string{
		"./scripts/install-agent.sh",
		"./install-agent.sh",
		"./install-agent-simple.sh",
		fmt.Sprintf("%s/scripts/install-agent.sh", workDir),
		fmt.Sprintf("%s/install-agent.sh", workDir),
		"/app/scripts/install-agent.sh", // Pour Docker
		"/app/install-agent.sh",           // Pour Docker
	}

	var scriptPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			scriptPath = path
			break
		}
	}

	if scriptPath == "" {
		log.Printf("[API] downloadInstallScript - Script non trouvé. Répertoire de travail: %s", workDir)
		log.Printf("[API] downloadInstallScript - Chemins testés: %v", possiblePaths)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "script d'installation non trouvé",
			"workdir": workDir,
			"hint":    "Le script doit être dans ./scripts/install-agent.sh ou dans le répertoire de travail",
		})
		return
	}

	log.Printf("[API] downloadInstallScript - Script trouvé: %s", scriptPath)

	// Définir les en-têtes pour servir le script
	c.Header("Content-Type", "text/x-shellscript")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", "install-agent.sh"))
	
	// Servir le fichier
	c.File(scriptPath)
}

// login authentifie un utilisateur
func (api *APIServer) login(c *gin.Context) {
	var loginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "données invalides"})
		return
	}

	// Authentification simple (à remplacer par une vraie authentification)
	if loginData.Username == "admin" && loginData.Password == "admin" {
		token, err := api.tokenManager.GenerateToken("admin", "Administrator", "admin", 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de génération du token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"id":   "admin",
				"name": "Administrator",
				"role": "admin",
			},
		})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "identifiants invalides"})
	}
}

// getAgents retourne la liste des agents (filtrée selon l'utilisateur)
// Inclut tous les agents (actifs et inactifs) depuis la base de données
func (api *APIServer) getAgents(c *gin.Context) {
	// Récupérer l'utilisateur depuis les claims
	claims, exists := auth.GetClaimsFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "utilisateur non authentifié"})
		return
	}

	// Si c'est un utilisateur web (UserID présent), filtrer les agents
	userID := claims.UserID
	if userID == "" {
		// Fallback pour compatibilité avec anciens tokens
		userID = claims.AgentID
	}

	// Récupérer les agents connectés (actifs)
	connectedAgents := api.hub.GetAgents()
	connectedMap := make(map[string]*Agent)
	for _, agent := range connectedAgents {
		connectedMap[agent.ID] = agent
	}

	// Récupérer tous les agents depuis la base de données (actifs et inactifs)
	var allAgentRecords []*AgentRecord
	if api.db != nil {
		allAgentRecordsFromDB, err := api.db.GetAgents()
		if err != nil {
			log.Printf("Erreur lors de la récupération des agents depuis la base de données: %v", err)
			// Continuer avec seulement les agents connectés en cas d'erreur
			allAgentRecords = []*AgentRecord{}
		} else {
			allAgentRecords = allAgentRecordsFromDB
		}
	}

	// Créer une map pour éviter les doublons (agents de la DB qui sont aussi connectés)
	agentMap := make(map[string]gin.H)

	// Traiter d'abord les agents connectés (actifs)
	for _, agent := range connectedAgents {
		metadata := api.hub.GetAgentMetadata(agent.ID)
		agentMap[agent.ID] = gin.H{
			"id":          agent.ID,
			"name":        agent.Name,
			"last_seen":   agent.LastSeen,
			"active":      true,
			"printers":    len(agent.GetPrinters()),
			"system_info": agent.GetSystemInfo(),
			"franchise":   metadata.Franchise,
			"category":    metadata.Category,
		}
	}

	// Ajouter les agents de la base de données qui ne sont pas connectés (inactifs)
	for _, agentRecord := range allAgentRecords {
		if _, alreadyAdded := agentMap[agentRecord.ID]; !alreadyAdded {
			// Agent uniquement en base de données, pas connecté
			// Pour les agents inactifs, utiliser les métadonnées depuis la DB ou le hub
			metadata := api.hub.GetAgentMetadata(agentRecord.ID)
			// Si pas de métadonnées dans le hub, utiliser celles de la DB
			if metadata.Franchise == "" {
				metadata = &AgentMetadata{
					Franchise: agentRecord.Franchise,
					Category:  agentRecord.Category,
				}
			}
			agentMap[agentRecord.ID] = gin.H{
				"id":          agentRecord.ID,
				"name":        agentRecord.Name,
				"last_seen":   agentRecord.LastSeen,
				"active":      false,
				"printers":    0, // Pas d'info sur les imprimantes pour les agents inactifs
				"system_info": nil,
				"franchise":   metadata.Franchise,
				"category":    metadata.Category,
			}
		}
	}

	// Convertir la map en liste
	var agentList []gin.H
	for _, agentData := range agentMap {
		agentList = append(agentList, agentData)
	}

	c.JSON(http.StatusOK, gin.H{
		"agents": agentList,
		"count":  len(agentList),
	})
}

// getAgent retourne les détails d'un agent
func (api *APIServer) getAgent(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	metadata := api.hub.GetAgentMetadata(agentID)
	c.JSON(http.StatusOK, gin.H{
		"id":          agent.ID,
		"name":        agent.Name,
		"last_seen":   agent.LastSeen,
		"active":      agent.IsActive(),
		"printers":    agent.GetPrinters(),
		"system_info": agent.GetSystemInfo(),
		"franchise":   metadata.Franchise,
		"category":    metadata.Category,
	})
}

// updateAgentMetadata met à jour les métadonnées d'un agent (franchise, category)
func (api *APIServer) updateAgentMetadata(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	var metadata struct {
		Franchise string `json:"franchise"`
		Category  string `json:"category"`
	}

	if err := c.ShouldBindJSON(&metadata); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "données invalides"})
		return
	}

	// Mettre à jour les métadonnées en mémoire
	api.hub.SetAgentMetadata(agent.ID, &AgentMetadata{
		Franchise: metadata.Franchise,
		Category:  metadata.Category,
	})

	// Sauvegarder dans la base de données
	if api.db != nil {
		// Charger l'agent existant depuis la base pour préserver les autres champs
		existingRecord, err := api.db.GetAgent(agent.ID)
		if err != nil {
			// Si l'agent n'existe pas en base, créer un nouvel enregistrement
			existingRecord = &AgentRecord{
				ID:        agent.ID,
				Name:      agent.Name,
				LastSeen:  agent.LastSeen,
				Status:    "online",
				CreatedAt: time.Now(),
			}
			// Récupérer l'IP depuis la connexion si disponible
			if agent.Conn != nil {
				existingRecord.IPAddress = agent.Conn.RemoteAddr()
			}
		}
		
		// Mettre à jour uniquement les métadonnées et les champs nécessaires
		existingRecord.Franchise = metadata.Franchise
		existingRecord.Category = metadata.Category
		existingRecord.Name = agent.Name
		existingRecord.LastSeen = agent.LastSeen
		if agent.Conn != nil {
			existingRecord.IPAddress = agent.Conn.RemoteAddr()
		}
		existingRecord.Status = "online"
		existingRecord.UpdatedAt = time.Now()
		
		if err := api.db.SaveAgent(existingRecord); err != nil {
			log.Printf("Erreur lors de la sauvegarde des métadonnées de l'agent %s en base: %v", agent.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors de la sauvegarde en base de données"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "métadonnées mises à jour",
		"agent_id":  agentID,
		"franchise": metadata.Franchise,
		"category":  metadata.Category,
	})
}

// executeCommand exécute une commande sur un agent
func (api *APIServer) executeCommand(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	var cmdData common.CommandData
	if err := c.ShouldBindJSON(&cmdData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "données de commande invalides"})
		return
	}

	// Créer un message de commande
	msg := common.NewMessage(common.MessageTypeCommand, &cmdData)
	msg.AgentID = agentID

	// Envoyer la commande à l'agent
	if err := agent.SendMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi de la commande"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "commande envoyée",
		"command": cmdData.Command,
	})
}

// getAgentPrinters retourne les imprimantes d'un agent
func (api *APIServer) getAgentPrinters(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	printers := agent.GetPrinters()
	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"printers": printers,
		"count":    len(printers),
	})
}

// getAgentSystem retourne les informations système d'un agent
func (api *APIServer) getAgentSystem(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	systemInfo := agent.GetSystemInfo()
	c.JSON(http.StatusOK, gin.H{
		"agent_id":    agentID,
		"system_info": systemInfo,
	})
}

// listFiles liste les fichiers d'un agent
func (api *APIServer) listFiles(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	path := c.Query("path")
	if path == "" {
		path = "."
	}

	// Normaliser le chemin : "/" au lieu de "." pour la racine
	if path == "." || path == "" {
		path = "/"
	}

	// Vérifier d'abord le cache
	if files, exists := agent.GetFileCache(path); exists {
		c.JSON(http.StatusOK, gin.H{
			"files": files,
			"path":  path,
		})
		return
	}

	// Si pas dans le cache, demander à l'agent de lister les fichiers
	fileData := &common.FileData{Path: path}
	msg := common.NewMessage(common.MessageTypeFileList, fileData)
	msg.AgentID = agentID

	// Envoyer la demande avec réponse (timeout: 5s)
	response, err := agent.SendMessageWithResponse(msg, 5*time.Second)
	if err != nil {
		log.Printf("[API] listFiles - Erreur lors de la demande: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"files":   []*common.FileData{},
			"path":    path,
			"message": "délai d'attente dépassé, l'agent n'a pas répondu",
		})
		return
	}

	// Parser la réponse
	var files []*common.FileData
	if filesData, ok := response.Data.([]*common.FileData); ok {
		files = filesData
		// Mettre à jour le cache avec le bon chemin
		agent.UpdateFileCache(path, files)
	} else if filesInterface, ok := response.Data.([]interface{}); ok {
		// Convertir []interface{} en []*common.FileData
		files = make([]*common.FileData, 0, len(filesInterface))
		for _, item := range filesInterface {
			if fileMap, ok := item.(map[string]interface{}); ok {
				fileData := &common.FileData{}
				if pathVal, exists := fileMap["path"]; exists {
					if pathStr, ok := pathVal.(string); ok {
						fileData.Path = pathStr
					}
				}
				if isDir, exists := fileMap["is_dir"]; exists {
					if isDirBool, ok := isDir.(bool); ok {
						fileData.IsDir = isDirBool
					}
				}
				if size, exists := fileMap["size"]; exists {
					if sizeFloat, ok := size.(float64); ok {
						fileData.Size = int64(sizeFloat)
					}
				}
				if mode, exists := fileMap["mode"]; exists {
					if modeFloat, ok := mode.(float64); ok {
						fileData.Mode = uint32(modeFloat)
					}
				}
				if modified, exists := fileMap["modified"]; exists {
					if modifiedStr, ok := modified.(string); ok {
						if modifiedTime, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
							fileData.Modified = modifiedTime
						}
					}
				}
				files = append(files, fileData)
			}
		}
		// Mettre à jour le cache avec le bon chemin
		agent.UpdateFileCache(path, files)
	} else {
		log.Printf("[API] listFiles - Format de données inattendu: %T", response.Data)
		files = []*common.FileData{}
	}

	c.JSON(http.StatusOK, gin.H{
		"files": files,
		"path":  path,
	})
}

// uploadFile upload un fichier vers un agent
func (api *APIServer) uploadFile(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	// Récupérer le fichier uploadé
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fichier manquant"})
		return
	}

	path := c.PostForm("path")
	if path == "" {
		path = file.Filename
	}

	// Ouvrir le fichier
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'ouverture du fichier"})
		return
	}
	defer src.Close()

	// Lire le fichier par chunks
	chunks := make([]*common.FileChunk, 0)
	buffer := make([]byte, 64*1024) // 64KB chunks
	offset := int64(0)

	for {
		n, err := src.Read(buffer)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de lecture du fichier"})
			return
		}

		if n == 0 {
			break
		}

		chunk := &common.FileChunk{
			Path:   path,
			Offset: offset,
			Data:   make([]byte, n),
			IsLast: false,
		}
		copy(chunk.Data, buffer[:n])

		chunks = append(chunks, chunk)
		offset += int64(n)

		if err == io.EOF {
			chunk.IsLast = true
			break
		}
	}

	// Envoyer les chunks à l'agent et attendre la confirmation
	msg := common.NewMessage(common.MessageTypeFileUpload, chunks)
	msg.AgentID = agentID

	log.Printf("[API] uploadFile - Envoi de %d chunks à l'agent %s, ID message: %s", len(chunks), agentID, msg.ID)

	// Attendre la réponse de l'agent (timeout de 30 secondes)
	response, err := agent.SendMessageWithResponse(msg, 30*time.Second)
	if err != nil {
		log.Printf("[API] uploadFile - Erreur lors de l'upload du fichier: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi du fichier ou timeout"})
		return
	}

	log.Printf("[API] uploadFile - Réponse reçue de l'agent, Type: %s, ID: %s", response.Type, response.ID)

	// Vérifier si l'agent a renvoyé une erreur
	if response.Type == common.MessageTypeFileError || response.Type == common.MessageTypeError {
		var errorData *common.ErrorData
		if errData, ok := response.Data.(*common.ErrorData); ok {
			errorData = errData
		} else if errMap, ok := response.Data.(map[string]interface{}); ok {
			errorData = &common.ErrorData{}
			if msg, ok := errMap["message"].(string); ok {
				errorData.Message = msg
			}
			if code, ok := errMap["code"].(string); ok {
				errorData.Code = code
			}
		}
		errorMsg := "erreur lors de l'upload du fichier"
		if errorData != nil && errorData.Message != "" {
			errorMsg = errorData.Message
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorMsg})
		return
	}

	// Vérifier que c'est bien une confirmation de complétion
	if response.Type != common.MessageTypeFileComplete {
		log.Printf("[API] uploadFile - Réponse inattendue: Type=%s (attendu: %s), ID=%s", response.Type, common.MessageTypeFileComplete, response.ID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "réponse inattendue de l'agent",
			"response_type": string(response.Type),
			"expected_type": string(common.MessageTypeFileComplete),
		})
		return
	}

	log.Printf("[API] uploadFile - Confirmation reçue, upload réussi pour %s", path)

	// Enregistrer l'opération dans les logs
	if api.db != nil {
		fileLog := &FileLog{
			AgentID:   agentID,
			Operation: "upload",
			Path:      path,
			Size:      file.Size,
			Success:   true,
			CreatedAt: time.Now(),
		}
		if err := api.db.LogFile(fileLog); err != nil {
			log.Printf("Erreur lors de l'enregistrement du log d'upload: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "fichier uploadé",
		"path":    path,
		"size":    file.Size,
	})
}

// downloadFile télécharge un fichier depuis un agent
func (api *APIServer) downloadFile(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chemin manquant"})
		return
	}

	// Créer un message de demande de téléchargement
	fileData := &common.FileData{Path: path}
	msg := common.NewMessage(common.MessageTypeFileDownload, fileData)
	msg.AgentID = agentID

	// Envoyer la demande à l'agent
	if err := agent.SendMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi de la demande"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "demande de téléchargement envoyée",
		"path":    path,
	})
}

// deleteFile supprime un fichier sur un agent
func (api *APIServer) deleteFile(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chemin manquant"})
		return
	}

	// Créer un message de suppression de fichier
	fileData := &common.FileData{Path: path}
	msg := common.NewMessage(common.MessageTypeFileDelete, fileData)
	msg.AgentID = agentID

	// Envoyer la demande à l'agent et attendre la confirmation
	response, err := agent.SendMessageWithResponse(msg, 10*time.Second)
	if err != nil {
		log.Printf("Erreur lors de la suppression du fichier: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi de la demande ou timeout"})
		return
	}

	// Vérifier si l'agent a renvoyé une erreur
	if response.Type == common.MessageTypeFileError || response.Type == common.MessageTypeError {
		var errorData *common.ErrorData
		if errData, ok := response.Data.(*common.ErrorData); ok {
			errorData = errData
		} else if errMap, ok := response.Data.(map[string]interface{}); ok {
			errorData = &common.ErrorData{}
			if msg, ok := errMap["message"].(string); ok {
				errorData.Message = msg
			}
			if code, ok := errMap["code"].(string); ok {
				errorData.Code = code
			}
		}
		errorMsg := "erreur lors de la suppression du fichier"
		if errorData != nil && errorData.Message != "" {
			errorMsg = errorData.Message
		}
		
		// Enregistrer l'erreur dans les logs
		if api.db != nil {
			fileLog := &FileLog{
				AgentID:   agentID,
				Operation: "delete",
				Path:      path,
				Success:   false,
				Error:     errorMsg,
				CreatedAt: time.Now(),
			}
			if err := api.db.LogFile(fileLog); err != nil {
				log.Printf("Erreur lors de l'enregistrement du log de suppression: %v", err)
			}
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorMsg})
		return
	}

	// Enregistrer l'opération réussie dans les logs
	if api.db != nil {
		fileLog := &FileLog{
			AgentID:   agentID,
			Operation: "delete",
			Path:      path,
			Success:   true,
			CreatedAt: time.Now(),
		}
		if err := api.db.LogFile(fileLog); err != nil {
			log.Printf("Erreur lors de l'enregistrement du log de suppression: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "fichier supprimé",
		"path":    path,
	})
}

// createDirectory crée un répertoire sur un agent
func (api *APIServer) createDirectory(c *gin.Context) {
	agentID := c.Param("id")
	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chemin manquant"})
		return
	}

	// Créer un message de création de répertoire
	fileData := &common.FileData{Path: req.Path, IsDir: true}
	msg := common.NewMessage(common.MessageTypeFileCreateDir, fileData)
	msg.AgentID = agentID

	// Envoyer la demande à l'agent et attendre la confirmation
	response, err := agent.SendMessageWithResponse(msg, 10*time.Second)
	if err != nil {
		log.Printf("Erreur lors de la création du répertoire: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi de la demande ou timeout"})
		return
	}

	// Vérifier si l'agent a renvoyé une erreur
	if response.Type == common.MessageTypeFileError || response.Type == common.MessageTypeError {
		var errorData *common.ErrorData
		if errData, ok := response.Data.(*common.ErrorData); ok {
			errorData = errData
		} else if errMap, ok := response.Data.(map[string]interface{}); ok {
			errorData = &common.ErrorData{}
			if msg, ok := errMap["message"].(string); ok {
				errorData.Message = msg
			}
			if code, ok := errMap["code"].(string); ok {
				errorData.Code = code
			}
		}
		errorMsg := "erreur lors de la création du répertoire"
		if errorData != nil && errorData.Message != "" {
			errorMsg = errorData.Message
		}
		
		// Enregistrer l'erreur dans les logs
		if api.db != nil {
			fileLog := &FileLog{
				AgentID:   agentID,
				Operation: "create_dir",
				Path:      req.Path,
				Success:   false,
				Error:     errorMsg,
				CreatedAt: time.Now(),
			}
			if err := api.db.LogFile(fileLog); err != nil {
				log.Printf("Erreur lors de l'enregistrement du log de création de répertoire: %v", err)
			}
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorMsg})
		return
	}

	// Enregistrer l'opération réussie dans les logs
	if api.db != nil {
		fileLog := &FileLog{
			AgentID:   agentID,
			Operation: "create_dir",
			Path:      req.Path,
			Success:   true,
			CreatedAt: time.Now(),
		}
		if err := api.db.LogFile(fileLog); err != nil {
			log.Printf("Erreur lors de l'enregistrement du log de création de répertoire: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "répertoire créé",
		"path":    req.Path,
	})
}

// listServices liste les services d'un agent
func (api *APIServer) listServices(c *gin.Context) {
	agentID := c.Param("id")
	log.Printf("[API] listServices - Demande de liste de services pour l'agent: %s", agentID)

	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		log.Printf("[API] listServices - Agent non trouvé: %s", agentID)
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	// Vérifier d'abord le cache
	services := agent.GetServices()
	log.Printf("[API] listServices - Services en cache: %d services", len(services))

	if len(services) > 0 {
		log.Printf("[API] listServices - Retour des services depuis le cache")
		c.JSON(http.StatusOK, gin.H{
			"services": services,
			"count":    len(services),
			"agent_id": agentID,
		})
		return
	}

	// Si pas dans le cache, demander à l'agent
	log.Printf("[API] listServices - Pas de cache, envoi de la demande à l'agent")
	msg := common.NewMessage(common.MessageTypeServiceList, nil)
	msg.AgentID = agentID
	log.Printf("[API] listServices - Message créé avec ID: %s", msg.ID)

	// Envoyer la demande avec réponse
	log.Printf("[API] listServices - Envoi du message et attente de la réponse (timeout: 5s)")
	response, err := agent.SendMessageWithResponse(msg, 5*time.Second)
	if err != nil {
		// Si erreur de timeout, retourner une liste vide
		log.Printf("[API] listServices - ERREUR: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"services": []interface{}{},
			"count":    0,
			"agent_id": agentID,
			"message":  "délai d'attente dépassé, l'agent n'a pas répondu",
		})
		return
	}

	log.Printf("[API] listServices - Réponse reçue, type: %s", response.Type)

	// Parser la réponse
	if services, ok := response.Data.([]interface{}); ok {
		log.Printf("[API] listServices - Données parsées avec succès: %d services", len(services))
		c.JSON(http.StatusOK, gin.H{
			"services": services,
			"count":    len(services),
			"agent_id": agentID,
		})
		return
	}

	// Vérifier à nouveau le cache après la réponse
	log.Printf("[API] listServices - Vérification du cache après réponse")
	services = agent.GetServices()
	log.Printf("[API] listServices - Services finaux: %d services", len(services))
	c.JSON(http.StatusOK, gin.H{
		"services": services,
		"count":    len(services),
		"agent_id": agentID,
	})
}

// getServiceStatus obtient le statut d'un service
func (api *APIServer) getServiceStatus(c *gin.Context) {
	agentID := c.Param("id")
	serviceName := c.Param("service")
	serviceType := c.Query("type")

	if serviceType == "" {
		serviceType = "systemd"
	}

	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	// Créer un message de demande de statut
	serviceInfo := &common.ServiceInfo{
		Name: serviceName,
		Type: serviceType,
	}
	msg := common.NewMessage(common.MessageTypeServiceStatus, serviceInfo)
	msg.AgentID = agentID

	// Envoyer la demande à l'agent
	if err := agent.SendMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi de la demande"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "demande de statut envoyée",
		"service": serviceName,
		"type":    serviceType,
	})
}

// executeServiceAction exécute une action sur un service
func (api *APIServer) executeServiceAction(c *gin.Context) {
	agentID := c.Param("id")
	serviceName := c.Param("service")
	action := c.Param("action")
	serviceType := c.Query("type")

	if serviceType == "" {
		serviceType = "systemd"
	}

	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	// Valider l'action
	validActions := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
		"enable":  true,
		"disable": true,
	}

	if !validActions[action] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action invalide"})
		return
	}

	// Créer un message d'action
	serviceAction := &common.ServiceAction{
		Name:   serviceName,
		Type:   serviceType,
		Action: action,
	}
	msg := common.NewMessage(common.MessageTypeServiceAction, serviceAction)
	msg.AgentID = agentID

	// Envoyer la demande à l'agent
	if err := agent.SendMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi de la demande"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "action envoyée",
		"service": serviceName,
		"type":    serviceType,
		"action":  action,
	})
}

// listLogSources liste les sources de logs disponibles
func (api *APIServer) listLogSources(c *gin.Context) {
	agentID := c.Param("id")
	log.Printf("[API] listLogSources - Demande de liste de sources de logs pour l'agent: %s", agentID)

	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		log.Printf("[API] listLogSources - Agent non trouvé: %s", agentID)
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	// Vérifier d'abord le cache
	sources := agent.GetLogSources()
	log.Printf("[API] listLogSources - Sources en cache: %d sources", len(sources))

	if len(sources) > 0 {
		log.Printf("[API] listLogSources - Retour des sources depuis le cache")
		c.JSON(http.StatusOK, gin.H{
			"sources":  sources,
			"count":    len(sources),
			"agent_id": agentID,
		})
		return
	}

	// Si pas dans le cache, demander à l'agent
	log.Printf("[API] listLogSources - Pas de cache, envoi de la demande à l'agent")
	msg := common.NewMessage(common.MessageTypeLogList, nil)
	msg.AgentID = agentID
	log.Printf("[API] listLogSources - Message créé avec ID: %s", msg.ID)

	// Envoyer la demande avec réponse
	log.Printf("[API] listLogSources - Envoi du message et attente de la réponse (timeout: 5s)")
	response, err := agent.SendMessageWithResponse(msg, 5*time.Second)
	if err != nil {
		log.Printf("[API] listLogSources - ERREUR: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"sources":  []interface{}{},
			"count":    0,
			"agent_id": agentID,
			"message":  "délai d'attente dépassé, l'agent n'a pas répondu",
		})
		return
	}

	log.Printf("[API] listLogSources - Réponse reçue, type: %s", response.Type)

	// Parser la réponse
	if sources, ok := response.Data.([]interface{}); ok {
		log.Printf("[API] listLogSources - Données parsées avec succès: %d sources", len(sources))
		c.JSON(http.StatusOK, gin.H{
			"sources":  sources,
			"count":    len(sources),
			"agent_id": agentID,
		})
		return
	}

	// Vérifier à nouveau le cache après la réponse
	log.Printf("[API] listLogSources - Vérification du cache après réponse")
	sources = agent.GetLogSources()
	log.Printf("[API] listLogSources - Sources finales: %d sources", len(sources))
	c.JSON(http.StatusOK, gin.H{
		"sources":  sources,
		"count":    len(sources),
		"agent_id": agentID,
	})
}

// getLogContent récupère le contenu des logs
func (api *APIServer) getLogContent(c *gin.Context) {
	agentID := c.Param("id")
	source := c.Param("source")
	log.Printf("[API] getLogContent - Demande de logs pour l'agent %s, source: %s", agentID, source)

	agent, exists := api.hub.GetAgent(agentID)
	if !exists {
		log.Printf("[API] getLogContent - Agent non trouvé: %s", agentID)
		c.JSON(http.StatusNotFound, gin.H{"error": "agent non trouvé"})
		return
	}

	// Récupérer les paramètres de requête
	logType := c.Query("type")
	if logType == "" {
		logType = "agent"
	}

	linesStr := c.Query("lines")
	lines := 100
	if linesStr != "" {
		fmt.Sscanf(linesStr, "%d", &lines)
	}

	log.Printf("[API] getLogContent - Paramètres: type=%s, lines=%d", logType, lines)

	// Créer la requête de logs
	logReq := &common.LogRequest{
		Source:   source,
		Type:     logType,
		Lines:    lines,
		Path:     c.Query("path"),
		Unit:     c.Query("unit"),
		Priority: c.Query("priority"),
		Since:    c.Query("since"),
		Until:    c.Query("until"),
	}

	msg := common.NewMessage(common.MessageTypeLogContent, logReq)
	msg.AgentID = agentID
	log.Printf("[API] getLogContent - Message créé avec ID: %s", msg.ID)

	// Envoyer la demande avec réponse
	log.Printf("[API] getLogContent - Envoi du message et attente de la réponse (timeout: 10s)")
	response, err := agent.SendMessageWithResponse(msg, 10*time.Second)
	if err != nil {
		log.Printf("[API] getLogContent - ERREUR: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"logs":     []interface{}{},
			"source":   source,
			"count":    0,
			"agent_id": agentID,
			"message":  "délai d'attente dépassé, l'agent n'a pas répondu",
		})
		return
	}

	log.Printf("[API] getLogContent - Réponse reçue, type: %s", response.Type)

	// Parser la réponse
	if logData, ok := response.Data.(map[string]interface{}); ok {
		if entries, ok := logData["entries"].([]interface{}); ok {
			log.Printf("[API] getLogContent - Données parsées avec succès: %d entrées", len(entries))
			c.JSON(http.StatusOK, gin.H{
				"logs":     entries,
				"source":   source,
				"count":    len(entries),
				"agent_id": agentID,
			})
			return
		}
	}

	// Sinon, essayer de parser comme un tableau direct
	if entries, ok := response.Data.([]interface{}); ok {
		log.Printf("[API] getLogContent - Données parsées comme tableau: %d entrées", len(entries))
		c.JSON(http.StatusOK, gin.H{
			"logs":     entries,
			"source":   source,
			"count":    len(entries),
			"agent_id": agentID,
		})
		return
	}

	log.Printf("[API] getLogContent - Format de données inattendu: %T", response.Data)
	c.JSON(http.StatusOK, gin.H{
		"logs":     []interface{}{},
		"source":   source,
		"count":    0,
		"agent_id": agentID,
	})
}

// oauth2Login redirige vers Authentik pour l'authentification
func (api *APIServer) oauth2Login(c *gin.Context) {
	if api.oauth2Config == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OAuth2 non configuré"})
		return
	}

	// Générer un state pour la sécurité
	state, err := auth.GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de génération du state"})
		return
	}

	// Stocker le state dans un cookie sécurisé
	c.SetCookie("oauth2_state", state, 600, "/", "", api.config.ServerTLS, true)

	// Rediriger vers Authentik
	authURL := api.oauth2Config.GetAuthURL(state)
	if authURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de génération de l'URL d'authentification"})
		return
	}

	log.Printf("[OAuth2] Redirection vers Authentik - Redirect URI: %s", api.config.OAuth2RedirectURL)
	log.Printf("[OAuth2] URL d'authentification complète: %s", authURL)
	c.Redirect(http.StatusFound, authURL)
}

// oauth2Callback gère le callback OAuth2 depuis Authentik
func (api *APIServer) oauth2Callback(c *gin.Context) {
	if api.oauth2Config == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OAuth2 non configuré"})
		return
	}

	// Récupérer le code et le state
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code ou state manquant"})
		return
	}

	// Vérifier le state depuis le cookie
	cookieState, err := c.Cookie("oauth2_state")
	if err != nil || cookieState == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state manquant ou invalide"})
		return
	}

	// Valider le state
	if err := auth.ValidateState(state, cookieState); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state invalide"})
		return
	}

	// Supprimer le cookie
	c.SetCookie("oauth2_state", "", -1, "/", "", api.config.ServerTLS, true)

	// Échanger le code contre un token
	ctx := context.Background()
	token, err := api.oauth2Config.ExchangeCode(ctx, code)
	if err != nil {
		log.Printf("Erreur d'échange du code OAuth2: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'authentification"})
		return
	}

	// Récupérer les informations utilisateur
	userInfo, err := api.oauth2Config.GetUserInfo(ctx, token)
	if err != nil {
		log.Printf("Erreur de récupération des infos utilisateur: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de récupération des informations utilisateur"})
		return
	}

	// Utiliser le sub (subject) comme ID utilisateur
	userID := userInfo.Sub
	if userID == "" {
		userID = userInfo.Email
	}
	if userID == "" {
		userID = userInfo.PreferredUsername
	}

	userName := userInfo.Name
	if userName == "" {
		userName = userInfo.Email
	}
	if userName == "" {
		userName = userInfo.PreferredUsername
	}

	// Déterminer le rôle (peut être basé sur les groupes)
	role := "user"
	if len(userInfo.Groups) > 0 {
		// Chercher un groupe admin
		for _, group := range userInfo.Groups {
			if strings.Contains(strings.ToLower(group), "admin") {
				role = "admin"
				break
			}
		}
	}

	// Sérialiser les groupes en JSON
	groupsJSON, _ := json.Marshal(userInfo.Groups)

	// Créer ou récupérer l'utilisateur en base de données
	user := &User{
		ID:       userID,
		Email:    userInfo.Email,
		Name:     userName,
		Username: userInfo.PreferredUsername,
		Role:     role,
		Groups:   string(groupsJSON),
	}

	log.Printf("[OAuth2] Tentative de sauvegarde utilisateur: ID=%s, Email=%s, Name=%s", userID, userInfo.Email, userName)

	// Vérifier si l'utilisateur existe déjà
	existingUser, err := api.db.GetUser(userID)
	if err != nil {
		// L'utilisateur n'existe pas, créer un nouveau
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		log.Printf("[OAuth2] Création d'un nouvel utilisateur dans la base de données...")
		if err := api.db.SaveUser(user); err != nil {
			log.Printf("[OAuth2] ❌ ERREUR de sauvegarde de l'utilisateur: %v", err)
			log.Printf("[OAuth2] Détails utilisateur: ID=%s, Email=%s, Name=%s", user.ID, user.Email, user.Name)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de sauvegarde de l'utilisateur", "details": err.Error()})
			return
		}
		log.Printf("[OAuth2] ✅ Nouvel utilisateur créé avec succès dans la base de données: %s (%s)", userName, userID)
	} else {
		// Mettre à jour l'utilisateur existant
		log.Printf("[OAuth2] Utilisateur existant trouvé, mise à jour...")
		existingUser.Email = userInfo.Email
		existingUser.Name = userName
		existingUser.Username = userInfo.PreferredUsername
		existingUser.Role = role
		existingUser.Groups = string(groupsJSON)
		existingUser.UpdatedAt = time.Now()
		if err := api.db.SaveUser(existingUser); err != nil {
			log.Printf("[OAuth2] ❌ ERREUR de mise à jour de l'utilisateur: %v", err)
		} else {
			log.Printf("[OAuth2] ✅ Utilisateur mis à jour avec succès: %s (%s)", userName, userID)
		}
		user = existingUser
	}

	// Générer le token JWT pour l'utilisateur web
	jwtToken, err := api.tokenManager.GenerateUserToken(userID, userName, role, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de génération du token"})
		return
	}

	// Rediriger vers le frontend avec le token dans l'URL
	// Le frontend récupérera le token depuis l'URL
	frontendURL := "/auth/callback?token=" + jwtToken
	c.Redirect(http.StatusFound, frontendURL)
}

// oauth2ConfigEndpoint retourne la configuration OAuth2 pour le frontend
func (api *APIServer) oauth2ConfigEndpoint(c *gin.Context) {
	if api.oauth2Config == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OAuth2 non configuré"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":  true,
		"loginUrl": "/api/auth/oauth2/login",
	})
}
