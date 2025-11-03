package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
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
}

// NewAPIServer crée un nouveau serveur API
func NewAPIServer(hub *Hub, tokenManager *auth.TokenManager) *APIServer {
	api := &APIServer{
		hub:          hub,
		tokenManager: tokenManager,
		router:       gin.Default(),
		wsServer:     NewWebSocketServer(hub, tokenManager),
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
	api.router.POST("/api/auth/login", api.login)

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

// getAgents retourne la liste des agents
func (api *APIServer) getAgents(c *gin.Context) {
	agents := api.hub.GetAgents()

	var agentList []gin.H
	for _, agent := range agents {
		metadata := api.hub.GetAgentMetadata(agent.ID)
		agentList = append(agentList, gin.H{
			"id":          agent.ID,
			"name":        agent.Name,
			"last_seen":   agent.LastSeen,
			"active":      agent.IsActive(),
			"printers":    len(agent.GetPrinters()),
			"system_info": agent.GetSystemInfo(),
			"franchise":   metadata.Franchise,
			"category":    metadata.Category,
		})
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

	// Mettre à jour les métadonnées
	api.hub.SetAgentMetadata(agent.ID, &AgentMetadata{
		Franchise: metadata.Franchise,
		Category:  metadata.Category,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "métadonnées mises à jour",
		"agent_id": agentID,
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

	// Envoyer les chunks à l'agent
	msg := common.NewMessage(common.MessageTypeFileUpload, chunks)
	msg.AgentID = agentID

	if err := agent.SendMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi du fichier"})
		return
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

	// Envoyer la demande à l'agent
	if err := agent.SendMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi de la demande"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "demande de suppression envoyée",
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

	// Envoyer la demande à l'agent
	if err := agent.SendMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur d'envoi de la demande"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "demande de création de répertoire envoyée",
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
