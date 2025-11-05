package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"remoteshell/internal/auth"
	"remoteshell/internal/common"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketServer gère les connexions WebSocket
type WebSocketServer struct {
	hub          *Hub
	tokenManager *auth.TokenManager
	authToken    string // Token simple pour les agents (REMOTESHELL_AUTH_TOKEN)
	upgrader     websocket.Upgrader
}

// NewWebSocketServer crée un nouveau serveur WebSocket
func NewWebSocketServer(hub *Hub, tokenManager *auth.TokenManager, authToken string) *WebSocketServer {
	return &WebSocketServer{
		hub:          hub,
		tokenManager: tokenManager,
		authToken:    authToken,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// En production, vérifier l'origine
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// HandleWebSocket gère les connexions WebSocket
func (ws *WebSocketServer) HandleWebSocket(c *gin.Context) {
	conn, err := ws.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Erreur d'upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Créer un wrapper pour la connexion
	wsConn := &WebSocketConnection{conn: conn}

	// Traiter la connexion
	ws.handleConnection(wsConn)
}

// handleConnection traite une connexion WebSocket
func (ws *WebSocketServer) handleConnection(conn WebSocketConn) {
	var agent *Agent
	var webClient *WebClient

	defer func() {
		if agent != nil {
			ws.hub.unregister <- agent
		}
		if webClient != nil {
			ws.hub.unregisterWeb <- webClient
		}
	}()

	// Boucle de lecture des messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Erreur WebSocket: %v", err)
			}
			break
		}

		// Parser le message
		msg, err := common.FromJSON(message)
		if err != nil {
			// Ignorer les erreurs de parsing pour les messages vides ou invalides (déconnexions, etc.)
			if len(message) == 0 {
				log.Printf("Message vide reçu (probable déconnexion), ignoré")
				break
			}
			// Afficher un aperçu du message (max 100 caractères) pour le débogage
			msgPreview := string(message)
			if len(msgPreview) > 100 {
				msgPreview = msgPreview[:100] + "..."
			}
			log.Printf("Erreur de parsing du message: %v (aperçu: %q)", err, msgPreview)
			continue
		}

		// Déterminer le type de connexion basé sur le premier message
		if agent == nil && webClient == nil {
			if msg.Type == common.MessageTypeAuth {
				// Vérifier le type de token pour distinguer agent et client web
				// Si le token n'est pas le token simple d'authentification, c'est probablement un client web
				if tokenData, ok := msg.Data.(map[string]interface{}); ok {
					if token, exists := tokenData["token"]; exists {
						if tokenStr, ok := token.(string); ok {
							// Si ce n'est pas le token simple d'auth, c'est un client web avec un token JWT
							if tokenStr != ws.authToken && tokenStr != "" {
								// C'est un client web avec un token JWT
								webClient = &WebClient{
									ID:   fmt.Sprintf("webclient_%d", time.Now().UnixNano()),
									Conn: conn,
								}
								ws.hub.registerWeb <- webClient
							}
						}
					}
				}
			} else {
				// C'est probablement un client web
				webClient = &WebClient{
					ID:   fmt.Sprintf("webclient_%d", time.Now().UnixNano()),
					Conn: conn,
				}
				ws.hub.registerWeb <- webClient
			}
		}

		// Traiter le message
		if err := ws.handleMessage(conn, msg, &agent); err != nil {
			log.Printf("Erreur de traitement du message: %v", err)
		}
	}
}

// handleMessage traite un message WebSocket
func (ws *WebSocketServer) handleMessage(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	switch msg.Type {
	case common.MessageTypeAuth:
		return ws.handleAuth(conn, msg, agent)

	case common.MessageTypeCommand:
		return ws.handleCommand(conn, msg, agent)

	case common.MessageTypeCommandExec:
		return ws.handleCommandExec(conn, msg, agent)

	case common.MessageTypeCommandDone:
		return ws.handleCommandDone(conn, msg, agent)

	case common.MessageTypeFileUpload:
		return ws.handleFileUpload(conn, msg, agent)

	case common.MessageTypeFileDownload:
		return ws.handleFileDownload(conn, msg, agent)

	case common.MessageTypeFileList:
		return ws.handleFileList(conn, msg, agent)

	case common.MessageTypeFileDelete:
		return ws.handleFileDelete(conn, msg, agent)

	case common.MessageTypeFileCreateDir:
		return ws.handleFileCreateDir(conn, msg, agent)

	case common.MessageTypeFileComplete:
		return ws.handleFileComplete(conn, msg, agent)

	case common.MessageTypeFileError:
		return ws.handleFileError(conn, msg, agent)

	case common.MessageTypePrinterStatus:
		return ws.handlePrinterStatus(conn, msg, agent)

	case common.MessageTypeSystemInfo:
		return ws.handleSystemInfo(conn, msg, agent)

	case common.MessageTypeHeartbeat:
		return ws.handleHeartbeat(conn, msg, agent)

	// Gestion des services
	case common.MessageTypeServiceList:
		return ws.handleServiceList(conn, msg, agent)

	case common.MessageTypeServiceStatus:
		return ws.handleServiceStatus(conn, msg, agent)

	case common.MessageTypeServiceResult:
		return ws.handleServiceResult(conn, msg, agent)

	// Gestion des logs
	case common.MessageTypeLogList:
		return ws.handleLogList(conn, msg, agent)

	case common.MessageTypeLogContent:
		return ws.handleLogContent(conn, msg, agent)

	// Messages d'erreur
	case common.MessageTypeError:
		return ws.handleError(conn, msg, agent)

	default:
		log.Printf("Type de message non géré: %s", msg.Type)
	}

	return nil
}

// handleAuth traite l'authentification
func (ws *WebSocketServer) handleAuth(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	var token string
	var agentID, agentName string

	// Extraire le token et les infos agent depuis les données
	switch data := msg.Data.(type) {
	case *common.AuthData:
		token = data.Token
	case map[string]interface{}:
		if tokenVal, exists := data["token"]; exists {
			if tokenStr, ok := tokenVal.(string); ok {
				token = tokenStr
			} else {
				return ws.sendAuthError(conn, "token invalide")
			}
		} else {
			return ws.sendAuthError(conn, "token manquant")
		}
		// Extraire agentID et agentName depuis map si disponibles
		if agentIDVal, exists := data["agent_id"]; exists {
			if agentIDStr, ok := agentIDVal.(string); ok {
				agentID = agentIDStr
			}
		}
		if agentNameVal, exists := data["agent_name"]; exists {
			if agentNameStr, ok := agentNameVal.(string); ok {
				agentName = agentNameStr
			}
		}
	default:
		return ws.sendAuthError(conn, "format de données invalide")
	}

	// Utiliser agentID depuis le message si disponible
	if agentID == "" && msg.AgentID != "" {
		agentID = msg.AgentID
	}

	// Pour les agents, on accepte un token simple ou un JWT
	// Essayer d'abord de valider comme JWT
	claims, err := ws.tokenManager.ValidateToken(token)
	if err != nil {
		// Si ce n'est pas un JWT valide, vérifier si c'est le token simple d'authentification
		if token == ws.authToken && ws.authToken != "" {
			// Token simple accepté - c'est un vrai agent
			if agentID == "" {
				return ws.sendAuthError(conn, "agentID manquant")
			}
			if agentName == "" {
				agentName = agentID // Utiliser l'ID comme nom par défaut
			}
		} else {
			return ws.sendAuthError(conn, "token invalide")
		}
	} else {
		// JWT valide - vérifier si c'est un utilisateur web ou un agent
		if claims.UserID != "" {
			// C'est un utilisateur web, ne pas créer d'agent
			// L'utilisateur web est déjà géré comme WebClient dans handleConnection
			return ws.sendAuthError(conn, "token utilisateur web - connexion agent non autorisée")
		}
		// C'est un agent avec JWT
		if claims.AgentID == "" {
			return ws.sendAuthError(conn, "token agent invalide - agentID manquant")
		}
		agentID = claims.AgentID
		agentName = claims.AgentName
	}

	// Créer l'agent uniquement si ce n'est pas un utilisateur web
	*agent = &Agent{
		ID:       agentID,
		Name:     agentName,
		Conn:     conn,
		LastSeen: time.Now(),
	}

	// Enregistrer l'agent
	ws.hub.register <- *agent

	// Envoyer la confirmation d'authentification
	successMsg := common.NewMessage(common.MessageTypeAuthSuccess, nil)
	return conn.SendMessage(successMsg)
}

// handleCommand traite une commande depuis un client web
func (ws *WebSocketServer) handleCommand(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	// Trouver l'agent cible
	agentID := msg.AgentID
	if agentID == "" {
		return ws.sendError(conn, "ID d'agent manquant")
	}

	targetAgent, exists := ws.hub.GetAgent(agentID)
	if !exists {
		return ws.sendError(conn, "agent non trouvé")
	}

	// Convertir le message en MessageTypeCommandExec pour l'agent
	execMsg := &common.Message{
		Type:      common.MessageTypeCommandExec,
		ID:        msg.ID,
		Data:      msg.Data,
		Timestamp: msg.Timestamp,
		AgentID:   agentID,
	}

	// Envoyer la commande à l'agent
	targetAgent.UpdateLastSeen()
	return targetAgent.SendMessage(execMsg)
}

// handleCommandExec traite l'exécution de commande
func (ws *WebSocketServer) handleCommandExec(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Transférer le message à l'agent
	(*agent).UpdateLastSeen()
	return (*agent).SendMessage(msg)
}

// handleCommandDone traite le résultat d'une commande
func (ws *WebSocketServer) handleCommandDone(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Créer un message de résultat pour le client
	resultMsg := &common.Message{
		Type:      "command_result",
		ID:        msg.ID,
		Data:      msg.Data,
		Timestamp: msg.Timestamp,
		AgentID:   (*agent).ID,
	}

	// Envoyer le résultat à tous les clients web connectés
	ws.hub.BroadcastToWebClients(resultMsg)

	return nil
}

// handleFileUpload traite l'upload de fichier
func (ws *WebSocketServer) handleFileUpload(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Transférer le message à l'agent
	(*agent).UpdateLastSeen()
	return (*agent).SendMessage(msg)
}

// handleFileDownload traite le téléchargement de fichier
func (ws *WebSocketServer) handleFileDownload(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Transférer le message à l'agent
	(*agent).UpdateLastSeen()
	return (*agent).SendMessage(msg)
}

// handleFileDelete traite la suppression de fichier
func (ws *WebSocketServer) handleFileDelete(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Transférer le message à l'agent
	(*agent).UpdateLastSeen()
	return (*agent).SendMessage(msg)
}

// handleFileCreateDir traite la création de répertoire
func (ws *WebSocketServer) handleFileCreateDir(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Transférer le message à l'agent
	(*agent).UpdateLastSeen()
	return (*agent).SendMessage(msg)
}

// handleFileComplete traite la confirmation d'upload/répertoire
func (ws *WebSocketServer) handleFileComplete(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	(*agent).UpdateLastSeen()

	// Si le message a un ID, c'est une réponse à une demande
	if msg.ID != "" {
		log.Printf("[WS] handleFileComplete - Routage de la réponse, ID: %s", msg.ID)
		(*agent).HandleResponse(msg)
	}

	return nil
}

// handleFileError traite les erreurs de fichiers
func (ws *WebSocketServer) handleFileError(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	(*agent).UpdateLastSeen()

	// Si le message a un ID, c'est une réponse à une demande
	if msg.ID != "" {
		log.Printf("[WS] handleFileError - Routage de l'erreur, ID: %s", msg.ID)
		(*agent).HandleResponse(msg)
	}

	return nil
}

// handleFileList traite la liste de fichiers
func (ws *WebSocketServer) handleFileList(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	log.Printf("[WS] handleFileList - Message reçu, ID: %s, Type de données: %T", msg.ID, msg.Data)

	// Si c'est une réponse d'un agent avec des données de fichiers
	// Gérer []*common.FileData et []interface{} (après sérialisation JSON)
	var files []*common.FileData
	var filesOk bool
	
	if filesData, ok := msg.Data.([]*common.FileData); ok {
		files = filesData
		filesOk = true
	} else if filesInterface, ok := msg.Data.([]interface{}); ok {
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
		filesOk = true
	}
	
	if filesOk && len(files) > 0 {
		// Extraire le chemin depuis les données
		// Pour les fichiers à la racine, les chemins seront comme "/bin", "/usr", etc.
		// Pour les sous-répertoires, ils seront comme "/home/user", "/var/log", etc.
		var path string
		if len(files) > 0 {
			firstPath := files[0].Path
			// Si le chemin commence par "/" et n'a qu'un seul "/" (ex: "/bin", "/usr")
			// ou s'il a exactement "/", c'est la racine
			if firstPath == "/" {
				path = "/"
			} else if strings.HasPrefix(firstPath, "/") {
				// Compter le nombre de "/" dans le chemin
				slashCount := strings.Count(firstPath, "/")
				if slashCount == 1 {
					// Un seul "/", c'est un fichier/répertoire à la racine
					path = "/"
				} else {
					// Plusieurs "/", prendre le chemin parent
					lastSlash := strings.LastIndex(firstPath, "/")
					if lastSlash > 0 {
						path = firstPath[:lastSlash]
						// Normaliser: si on obtient une chaîne vide, c'est la racine
						if path == "" {
							path = "/"
						}
					} else {
						path = "/"
					}
				}
			} else {
				// Chemin relatif, essayer d'extraire le répertoire parent
				lastSlash := strings.LastIndex(firstPath, "/")
				if lastSlash > 0 {
					path = firstPath[:lastSlash]
				} else {
					path = "."
				}
			}
		} else {
			path = "/"
		}

		log.Printf("[WS] handleFileList - Chemin détecté: %s pour %d fichiers", path, len(files))

		// Mettre à jour le cache de fichiers
		(*agent).UpdateFileCache(path, files)

		// Si le message a un ID, c'est une réponse à une demande
		if msg.ID != "" {
			log.Printf("[WS] handleFileList - Envoi de la réponse au canal, ID: %s", msg.ID)
			(*agent).HandleResponse(msg)
		} else {
			log.Printf("[WS] handleFileList - Message sans ID, pas de réponse attendue")
		}

		return nil
	} else if _, ok := msg.Data.(*common.FileData); ok {
		// C'est une demande de liste, transférer à l'agent
		(*agent).UpdateLastSeen()
		return (*agent).SendMessage(msg)
	} else if _, ok := msg.Data.(map[string]interface{}); ok {
		// C'est peut-être une demande avec le chemin dans une map
		(*agent).UpdateLastSeen()
		return (*agent).SendMessage(msg)
	}

	// Sinon, transférer le message à l'agent
	(*agent).UpdateLastSeen()
	return (*agent).SendMessage(msg)
}

// handlePrinterStatus traite le statut des imprimantes
func (ws *WebSocketServer) handlePrinterStatus(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Mettre à jour les informations des imprimantes
	if printers, ok := msg.Data.([]*common.PrinterInfo); ok {
		(*agent).UpdatePrinters(printers)
	}

	(*agent).UpdateLastSeen()
	return nil
}

// handleSystemInfo traite les informations système
func (ws *WebSocketServer) handleSystemInfo(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Mettre à jour les informations système
	if systemInfo, ok := msg.Data.(*common.SystemInfo); ok {
		(*agent).UpdateSystemInfo(systemInfo)
	}

	(*agent).UpdateLastSeen()
	return nil
}

// handleHeartbeat traite le heartbeat
func (ws *WebSocketServer) handleHeartbeat(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Mettre à jour le timestamp
	(*agent).UpdateLastSeen()

	// Répondre au heartbeat (sans log pour éviter la pollution)
	response := common.NewMessage(common.MessageTypeHeartbeat, nil)
	return conn.SendMessage(response)
}

// handleServiceList traite la liste des services
func (ws *WebSocketServer) handleServiceList(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		log.Printf("[WS] handleServiceList - Agent non authentifié")
		return ws.sendError(conn, "non authentifié")
	}

	log.Printf("[WS] handleServiceList - Message reçu de l'agent %s, ID: %s", (*agent).ID, msg.ID)
	log.Printf("[WS] handleServiceList - Type de données: %T", msg.Data)

	// Mettre à jour les informations des services
	if services, ok := msg.Data.([]interface{}); ok {
		log.Printf("[WS] handleServiceList - Données sont []interface{} avec %d services", len(services))
		// Convertir en slice de ServiceInfo
		serviceInfos := make([]*common.ServiceInfo, 0, len(services))
		for i, svc := range services {
			if svcMap, ok := svc.(map[string]interface{}); ok {
				serviceInfo := &common.ServiceInfo{}
				if name, ok := svcMap["name"].(string); ok {
					serviceInfo.Name = name
				}
				if typ, ok := svcMap["type"].(string); ok {
					serviceInfo.Type = typ
				}
				if status, ok := svcMap["status"].(string); ok {
					serviceInfo.Status = status
				}
				if state, ok := svcMap["state"].(string); ok {
					serviceInfo.State = state
				}
				if desc, ok := svcMap["description"].(string); ok {
					serviceInfo.Description = desc
				}
				if enabled, ok := svcMap["enabled"].(bool); ok {
					serviceInfo.Enabled = enabled
				}
				if containerID, ok := svcMap["container_id"].(string); ok {
					serviceInfo.ContainerID = containerID
				}
				if image, ok := svcMap["image"].(string); ok {
					serviceInfo.Image = image
				}
				serviceInfos = append(serviceInfos, serviceInfo)
				log.Printf("[WS] handleServiceList - Service %d: %s (%s) - %s", i, serviceInfo.Name, serviceInfo.Type, serviceInfo.State)
			}
		}
		log.Printf("[WS] handleServiceList - Mise à jour du cache avec %d services", len(serviceInfos))
		(*agent).UpdateServices(serviceInfos)
	} else {
		log.Printf("[WS] handleServiceList - Les données ne sont PAS []interface{}, type: %T", msg.Data)
	}

	// Si le message a un ID, c'est une réponse à une demande
	if msg.ID != "" {
		log.Printf("[WS] handleServiceList - Message avec ID, envoi de la réponse au canal")
		(*agent).HandleResponse(msg)
	} else {
		log.Printf("[WS] handleServiceList - Message sans ID, pas de réponse attendue")
	}

	(*agent).UpdateLastSeen()
	return nil
}

// handleServiceStatus traite le statut d'un service
func (ws *WebSocketServer) handleServiceStatus(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Si le message a un ID, c'est une réponse à une demande
	if msg.ID != "" {
		(*agent).HandleResponse(msg)
	}

	(*agent).UpdateLastSeen()
	return nil
}

// handleServiceResult traite le résultat d'une action sur un service
func (ws *WebSocketServer) handleServiceResult(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		return ws.sendError(conn, "non authentifié")
	}

	// Si le message a un ID, c'est une réponse à une demande
	if msg.ID != "" {
		(*agent).HandleResponse(msg)
	}

	(*agent).UpdateLastSeen()
	return nil
}

// handleLogList traite la liste des sources de logs
func (ws *WebSocketServer) handleLogList(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		log.Printf("[WS] handleLogList - Agent non authentifié")
		return ws.sendError(conn, "non authentifié")
	}

	log.Printf("[WS] handleLogList - Message reçu de l'agent %s, ID: %s", (*agent).ID, msg.ID)
	log.Printf("[WS] handleLogList - Type de données: %T", msg.Data)

	// Mettre à jour les informations des sources de logs
	if sources, ok := msg.Data.([]interface{}); ok {
		log.Printf("[WS] handleLogList - Données sont []interface{} avec %d sources", len(sources))
		// Convertir en slice de LogSource
		logSources := make([]*common.LogSource, 0, len(sources))
		for i, src := range sources {
			if srcMap, ok := src.(map[string]interface{}); ok {
				logSource := &common.LogSource{}
				if name, ok := srcMap["name"].(string); ok {
					logSource.Name = name
				}
				if typ, ok := srcMap["type"].(string); ok {
					logSource.Type = typ
				}
				if path, ok := srcMap["path"].(string); ok {
					logSource.Path = path
				}
				if desc, ok := srcMap["description"].(string); ok {
					logSource.Description = desc
				}
				logSources = append(logSources, logSource)
				log.Printf("[WS] handleLogList - Source %d: %s (%s)", i, logSource.Name, logSource.Type)
			}
		}
		log.Printf("[WS] handleLogList - Mise à jour du cache avec %d sources", len(logSources))
		(*agent).UpdateLogSources(logSources)
	} else {
		log.Printf("[WS] handleLogList - Les données ne sont PAS []interface{}, type: %T", msg.Data)
	}

	// Si le message a un ID, c'est une réponse à une demande
	if msg.ID != "" {
		log.Printf("[WS] handleLogList - Message avec ID, envoi de la réponse au canal")
		(*agent).HandleResponse(msg)
	} else {
		log.Printf("[WS] handleLogList - Message sans ID, pas de réponse attendue")
	}

	(*agent).UpdateLastSeen()
	return nil
}

// handleLogContent traite le contenu des logs
func (ws *WebSocketServer) handleLogContent(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		log.Printf("[WS] handleLogContent - Agent non authentifié")
		return ws.sendError(conn, "non authentifié")
	}

	log.Printf("[WS] handleLogContent - Message reçu de l'agent %s, ID: %s", (*agent).ID, msg.ID)

	// Si c'est une réponse avec des logs
	if msg.ID != "" {
		if logData, ok := msg.Data.(map[string]interface{}); ok {
			if entries, ok := logData["entries"]; ok {
				if entriesSlice, ok := entries.([]interface{}); ok {
					log.Printf("[WS] handleLogContent - Logs reçus: %d entrées", len(entriesSlice))
				}
			}
		}
		log.Printf("[WS] handleLogContent - Envoi de la réponse au canal")
		(*agent).HandleResponse(msg)
	}

	(*agent).UpdateLastSeen()
	return nil
}

// handleError traite les messages d'erreur de l'agent
func (ws *WebSocketServer) handleError(conn WebSocketConn, msg *common.Message, agent **Agent) error {
	if *agent == nil {
		// Si ce n'est pas un agent, ignorer
		return nil
	}

	// Si le message a un ID, c'est probablement une réponse à une requête
	if msg.ID != "" {
		(*agent).HandleResponse(msg)
	} else {
		// Sinon, transférer l'erreur aux clients web
		resultMsg := &common.Message{
			Type:      "error",
			ID:        msg.ID,
			Data:      msg.Data,
			Timestamp: msg.Timestamp,
			AgentID:   (*agent).ID,
		}
		ws.hub.BroadcastToWebClients(resultMsg)
	}

	(*agent).UpdateLastSeen()
	return nil
}

// sendAuthError envoie une erreur d'authentification
func (ws *WebSocketServer) sendAuthError(conn WebSocketConn, message string) error {
	errorMsg := common.NewMessage(common.MessageTypeAuthError, &common.ErrorData{
		Code:    "AUTH_ERROR",
		Message: message,
	})
	return conn.SendMessage(errorMsg)
}

// sendError envoie une erreur générique
func (ws *WebSocketServer) sendError(conn WebSocketConn, message string) error {
	errorMsg := common.NewMessage(common.MessageTypeError, &common.ErrorData{
		Code:    "ERROR",
		Message: message,
	})
	return conn.SendMessage(errorMsg)
}

// WebSocketConnection encapsule une connexion WebSocket
type WebSocketConnection struct {
	conn *websocket.Conn
}

// ReadMessage lit un message
func (wsc *WebSocketConnection) ReadMessage() (int, []byte, error) {
	return wsc.conn.ReadMessage()
}

// WriteMessage écrit un message
func (wsc *WebSocketConnection) WriteMessage(messageType int, data []byte) error {
	return wsc.conn.WriteMessage(messageType, data)
}

// Close ferme la connexion
func (wsc *WebSocketConnection) Close() error {
	return wsc.conn.Close()
}

// RemoteAddr retourne l'adresse distante
func (wsc *WebSocketConnection) RemoteAddr() string {
	return wsc.conn.RemoteAddr().String()
}

// SendMessage envoie un message
func (wsc *WebSocketConnection) SendMessage(msg *common.Message) error {
	data, err := msg.ToJSON()
	if err != nil {
		return err
	}
	return wsc.WriteMessage(websocket.TextMessage, data)
}
