package agent

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"remoteshell/internal/auth"
	"remoteshell/internal/common"

	"github.com/gorilla/websocket"
)

// Client représente un client agent WebSocket
type Client struct {
	config         *common.Config
	tokenManager   *auth.TokenManager
	executor       *Executor
	printerMonitor *PrinterMonitor
	fileManager    *FileManager
	serviceManager *ServiceManager
	logManager     *LogManager
	conn           *websocket.Conn
	connected      bool
	reconnect      bool
	stopChan       chan struct{}
	disconnectChan chan struct{}
	messageChan    chan *common.Message
	mu             sync.RWMutex
	agentID        string
	agentName      string
	lastLogTime    time.Time // Pour limiter les logs de heartbeat
	logMutex       sync.Mutex
}

// NewClient crée un nouveau client agent
func NewClient(config *common.Config, tokenManager *auth.TokenManager) *Client {
	// Générer un ID d'agent unique si non fourni
	agentID := config.AgentID
	if agentID == "" {
		hostname, _ := os.Hostname()
		agentID = fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
	}

	// Utiliser le nom d'agent ou l'hostname
	agentName := config.AgentName
	if agentName == "" {
		hostname, _ := os.Hostname()
		agentName = hostname
	}

	executor := NewExecutor("")
	printerMonitor := NewPrinterMonitor()
	fileManager := NewFileManager("", config.ChunkSize)
	serviceManager := NewServiceManager()
	logManager := NewLogManager(1000)

	return &Client{
		config:         config,
		tokenManager:   tokenManager,
		executor:       executor,
		printerMonitor: printerMonitor,
		fileManager:    fileManager,
		serviceManager: serviceManager,
		logManager:     logManager,
		connected:      false,
		reconnect:      true,
		stopChan:       make(chan struct{}),
		disconnectChan:  make(chan struct{}, 1),
		messageChan:    make(chan *common.Message, 100),
		agentID:        agentID,
		agentName:      agentName,
	}
}

// Start démarre le client agent
func (c *Client) Start() error {
	log.Printf("Démarrage de l'agent %s (%s)", c.agentName, c.agentID)

	// Gérer les signaux système
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Signal d'arrêt reçu, fermeture de l'agent...")
		c.Stop()
	}()

	// Boucle de connexion avec reconnexion automatique
	for c.reconnect {
		if err := c.connect(); err != nil {
			log.Printf("Erreur de connexion: %v", err)
			if c.reconnect {
				log.Printf("Reconnexion dans %v...", c.config.ReconnectDelay)
				time.Sleep(c.config.ReconnectDelay)
				continue
			}
			return err
		}

		// Démarrer les goroutines de traitement
		go c.handleMessages()
		go c.sendHeartbeat()
		go c.sendPrinterStatus()

		// Attendre la déconnexion ou l'arrêt
		select {
		case <-c.stopChan:
			// Arrêt demandé
			return nil
		case <-c.disconnectChan:
			// Déconnexion inattendue, relancer la reconnexion
			log.Println("Déconnexion détectée, reconnexion automatique...")
			// Recréer le canal pour la prochaine déconnexion
			c.disconnectChan = make(chan struct{}, 1)
		}
	}

	return nil
}

// Stop arrête le client agent
func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reconnect = false
	c.connected = false

	if c.conn != nil {
		c.conn.Close()
	}

	close(c.stopChan)
	log.Println("Agent arrêté")
}

// connect établit la connexion WebSocket
func (c *Client) connect() error {
	// Construire l'URL WebSocket
	serverURL := c.config.GetServerURL() + "/ws"

	// Configuration TLS si nécessaire
	var dialer *websocket.Dialer
	if c.config.ServerTLS {
		dialer = &websocket.Dialer{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // À changer en production
			},
			HandshakeTimeout: 30 * time.Second, // Timeout plus long pour le handshake
		}
		log.Printf("Tentative de connexion WSS (WebSocket Secure) à %s", serverURL)
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout: 30 * time.Second,
		}
		log.Printf("Tentative de connexion WS (WebSocket) à %s", serverURL)
	}

	// Connexion WebSocket
	conn, resp, err := dialer.Dial(serverURL, nil)
	if err != nil {
		if resp != nil {
			statusCode := resp.StatusCode
			var bodyStr string
			if resp.Body != nil {
				bodyBytes, _ := io.ReadAll(resp.Body)
				bodyStr = string(bodyBytes)
			}
			return fmt.Errorf("échec de la connexion WebSocket: %v (Status: %d, Body: %s)", err, statusCode, bodyStr)
		}
		return fmt.Errorf("échec de la connexion WebSocket: %v", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	log.Printf("Connecté au serveur %s", serverURL)

	// Authentification
	if err := c.authenticate(); err != nil {
		conn.Close()
		return fmt.Errorf("échec de l'authentification: %v", err)
	}

	return nil
}

// authenticate s'authentifie auprès du serveur
func (c *Client) authenticate() error {
	// Utiliser le token fourni dans la configuration
	token := c.config.AuthToken
	if token == "" {
		return fmt.Errorf("aucun token d'authentification fourni")
	}

	// Envoyer le message d'authentification
	authMsg := common.NewMessage(common.MessageTypeAuth, &common.AuthData{
		Token: token,
	})
	authMsg.AgentID = c.agentID

	if err := c.sendMessage(authMsg); err != nil {
		return fmt.Errorf("envoi du message d'authentification échoué: %v", err)
	}

	// Attendre la réponse d'authentification
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout d'authentification")
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				return fmt.Errorf("lecture du message d'authentification échouée: %v", err)
			}

			msg, err := common.FromJSON(message)
			if err != nil {
				continue
			}

			switch msg.Type {
			case common.MessageTypeAuthSuccess:
				log.Println("Authentification réussie")
				return nil
			case common.MessageTypeAuthError:
				return fmt.Errorf("authentification échouée: %v", msg.Data)
			}
		}
	}
}

// handleMessages traite les messages entrants
func (c *Client) handleMessages() {
	for {
		c.mu.RLock()
		conn := c.conn
		connected := c.connected
		c.mu.RUnlock()

		if !connected || conn == nil {
			break
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Erreur de lecture WebSocket: %v", err)
			c.disconnect()
			// Signaler la déconnexion pour relancer la reconnexion
			select {
			case c.disconnectChan <- struct{}{}:
			default:
			}
			break
		}

		msg, err := common.FromJSON(message)
		if err != nil {
			log.Printf("Erreur de parsing du message: %v", err)
			continue
		}

		// Traiter le message
		if err := c.processMessage(msg); err != nil {
			log.Printf("Erreur de traitement du message: %v", err)
		}
	}
}

// processMessage traite un message spécifique
func (c *Client) processMessage(msg *common.Message) error {
	// Log tous les messages sauf heartbeat (pour éviter la pollution)
	// Les heartbeats sont loggés séparément avec limitation de fréquence
	if msg.Type != common.MessageTypeHeartbeat {
		log.Printf("[AGENT] processMessage - Message reçu: Type=%s, ID=%s, AgentID=%s", msg.Type, msg.ID, msg.AgentID)
	} else {
		// Pour les heartbeats, log seulement si plus d'1 seconde s'est écoulée depuis le dernier log
		c.logMutex.Lock()
		now := time.Now()
		shouldLog := now.Sub(c.lastLogTime) >= time.Second
		if shouldLog {
			c.lastLogTime = now
			log.Printf("DEBUG: Traitement du message de type: heartbeat, ID: %s", msg.ID)
			log.Printf("DEBUG: Données du message: %v", msg.Data)
		}
		c.logMutex.Unlock()
	}

	switch msg.Type {
	case common.MessageTypeCommand:
		return c.handleCommand(msg)
	case common.MessageTypeCommandExec:
		return c.handleCommand(msg)
	case common.MessageTypeFileUpload:
		return c.handleFileUpload(msg)
	case common.MessageTypeFileDownload:
		return c.handleFileDownload(msg)
	case common.MessageTypeFileList:
		return c.handleFileList(msg)
	case common.MessageTypeFileDelete:
		return c.handleFileDelete(msg)
	case common.MessageTypeFileCreateDir:
		return c.handleFileCreateDir(msg)
	case common.MessageTypeServiceList:
		return c.handleServiceList(msg)
	case common.MessageTypeServiceStatus:
		return c.handleServiceStatus(msg)
	case common.MessageTypeServiceAction:
		return c.handleServiceAction(msg)
	case common.MessageTypeLogList:
		return c.handleLogList(msg)
	case common.MessageTypeLogContent:
		return c.handleLogContent(msg)
	case common.MessageTypeHeartbeat:
		// Répondre au heartbeat (les logs sont déjà gérés dans processMessage avec limitation)
		response := common.NewMessage(common.MessageTypeHeartbeat, nil)
		response.AgentID = c.agentID
		return c.sendMessage(response)
	default:
		log.Printf("[AGENT] processMessage - Type de message non géré: %s", msg.Type)
	}

	return nil
}

// handleCommand traite une commande
func (c *Client) handleCommand(msg *common.Message) error {
	log.Printf("[Client] === handleCommand appelé === Message ID: %s", msg.ID)
	var cmdData *common.CommandData

	// Gérer les différents types de données
	switch data := msg.Data.(type) {
	case *common.CommandData:
		cmdData = data
		log.Printf("[Client] CommandData reçu directement: command=%q, workingDir=%q", cmdData.Command, cmdData.WorkingDir)
	case map[string]interface{}:
		// Convertir map en CommandData
		cmdData = &common.CommandData{}
		if cmd, exists := data["command"]; exists {
			if cmdStr, ok := cmd.(string); ok {
				cmdData.Command = cmdStr
				log.Printf("[Client] Commande extraite du map: %q", cmdStr)
			}
		}
		if workingDir, exists := data["working_dir"]; exists {
			if dirStr, ok := workingDir.(string); ok {
				cmdData.WorkingDir = dirStr
				log.Printf("[Client] WorkingDir extrait du map: %q", dirStr)
			}
		}
		if timeout, exists := data["timeout"]; exists {
			if timeoutNum, ok := timeout.(float64); ok {
				cmdData.Timeout = int(timeoutNum)
			}
		}
		log.Printf("[Client] CommandData construit depuis map: command=%q, workingDir=%q, timeout=%d", 
			cmdData.Command, cmdData.WorkingDir, cmdData.Timeout)
	default:
		log.Printf("[Client] ERREUR: Format de données invalide: %T", msg.Data)
		return fmt.Errorf("format de données de commande invalide: %T", msg.Data)
	}

	if cmdData.Command == "" {
		log.Printf("[Client] ERREUR: Commande manquante")
		return fmt.Errorf("commande manquante")
	}

	log.Printf("[Client] Commande validée: %q, appel de ExecuteWithTimeout...", cmdData.Command)

	// Vérifier la sécurité de la commande
	if !c.executor.IsCommandSafe(cmdData.Command) {
		log.Printf("[Client] ERREUR: Commande non sécurisée: %q", cmdData.Command)
		errorMsg := common.NewMessageWithID(common.MessageTypeError, msg.ID, &common.ErrorData{
			Code:    "UNSAFE_COMMAND",
			Message: "Commande non autorisée pour des raisons de sécurité",
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	log.Printf("[Client] Commande sécurisée, exécution...")
	// Exécuter la commande
	output, err := c.executor.ExecuteWithTimeout(cmdData)
	log.Printf("[Client] ExecuteWithTimeout terminé, erreur: %v", err)
	if err != nil {
		errorMsg := common.NewMessageWithID(common.MessageTypeError, msg.ID, &common.ErrorData{
			Code:    "EXECUTION_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	// Envoyer le résultat
	resultMsg := common.NewMessageWithID(common.MessageTypeCommandDone, msg.ID, output)
	resultMsg.AgentID = c.agentID
	return c.sendMessage(resultMsg)
}

// handleFileUpload traite l'upload de fichier
func (c *Client) handleFileUpload(msg *common.Message) error {
	log.Printf("[AGENT] handleFileUpload - Début, ID: %s, Type de données: %T", msg.ID, msg.Data)
	
	var chunks []*common.FileChunk
	var chunksOk bool

	// Gérer les différents types de données (après sérialisation JSON)
	if chunksData, ok := msg.Data.([]*common.FileChunk); ok {
		log.Printf("[AGENT] handleFileUpload - Type []*common.FileChunk détecté, %d chunks", len(chunksData))
		chunks = chunksData
		chunksOk = true
	} else if chunksInterface, ok := msg.Data.([]interface{}); ok {
		log.Printf("[AGENT] handleFileUpload - Type []interface{} détecté, %d éléments", len(chunksInterface))
		// Convertir []interface{} en []*common.FileChunk
		chunks = make([]*common.FileChunk, 0, len(chunksInterface))
		for i, item := range chunksInterface {
			if chunkMap, ok := item.(map[string]interface{}); ok {
				chunk := &common.FileChunk{}
				if pathVal, exists := chunkMap["path"]; exists {
					if pathStr, ok := pathVal.(string); ok {
						chunk.Path = pathStr
					}
				}
				if offsetVal, exists := chunkMap["offset"]; exists {
					if offsetFloat, ok := offsetVal.(float64); ok {
						chunk.Offset = int64(offsetFloat)
					}
				}
				if dataVal, exists := chunkMap["data"]; exists {
					// Les données binaires peuvent être encodées en base64 ou être un tableau de bytes
					if dataStr, ok := dataVal.(string); ok {
						// Essayer de décoder en base64
						decoded, err := base64.StdEncoding.DecodeString(dataStr)
						if err != nil {
							log.Printf("[AGENT] handleFileUpload - ERREUR décodage base64 chunk %d: %v", i, err)
							errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
								Code:    "DECODE_ERROR",
								Message: fmt.Sprintf("erreur de décodage base64 du chunk %d: %v", i, err),
							})
							errorMsg.AgentID = c.agentID
							if sendErr := c.sendMessage(errorMsg); sendErr != nil {
								log.Printf("[AGENT] handleFileUpload - ERREUR envoi message d'erreur: %v", sendErr)
							}
							return fmt.Errorf("erreur de décodage base64: %v", err)
						}
						chunk.Data = decoded
					} else if dataArray, ok := dataVal.([]interface{}); ok {
						// Tableau de nombres (bytes encodés en JSON)
						chunk.Data = make([]byte, len(dataArray))
						for j, v := range dataArray {
							if byteVal, ok := v.(float64); ok {
								chunk.Data[j] = byte(byteVal)
							}
						}
					} else if dataBytes, ok := dataVal.([]byte); ok {
						chunk.Data = dataBytes
					} else {
						log.Printf("[AGENT] handleFileUpload - Type de données inattendu pour chunk %d: %T", i, dataVal)
					}
				}
				if isLastVal, exists := chunkMap["is_last"]; exists {
					if isLastBool, ok := isLastVal.(bool); ok {
						chunk.IsLast = isLastBool
					}
				}
				if checksumVal, exists := chunkMap["checksum"]; exists {
					if checksumStr, ok := checksumVal.(string); ok {
						chunk.Checksum = checksumStr
					}
				}
				chunks = append(chunks, chunk)
			} else {
				log.Printf("[AGENT] handleFileUpload - Élément %d n'est pas un map: %T", i, item)
			}
		}
		chunksOk = true
		log.Printf("[AGENT] handleFileUpload - Conversion terminée, %d chunks valides", len(chunks))
	} else {
		log.Printf("[AGENT] handleFileUpload - Type de données non géré: %T", msg.Data)
	}

	if !chunksOk || len(chunks) == 0 {
		log.Printf("[AGENT] handleFileUpload - ERREUR: chunks invalides (chunksOk=%v, len=%d)", chunksOk, len(chunks))
		errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
			Code:    "INVALID_DATA",
			Message: fmt.Sprintf("données de chunks invalides (type: %T, chunksOk: %v, len: %d)", msg.Data, chunksOk, len(chunks)),
		})
		errorMsg.AgentID = c.agentID
		if err := c.sendMessage(errorMsg); err != nil {
			log.Printf("[AGENT] handleFileUpload - ERREUR envoi message d'erreur: %v", err)
			return err
		}
		return fmt.Errorf("données de chunks invalides")
	}

	// Utiliser le chemin du premier chunk
	path := chunks[0].Path
	log.Printf("[AGENT] handleFileUpload - Upload de %d chunks vers %s", len(chunks), path)

	// Uploader le fichier
	if err := c.fileManager.UploadFile(path, chunks); err != nil {
		log.Printf("[AGENT] handleFileUpload - ERREUR lors de l'upload: %v", err)
		errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
			Code:    "UPLOAD_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		if sendErr := c.sendMessage(errorMsg); sendErr != nil {
			log.Printf("[AGENT] handleFileUpload - ERREUR envoi message d'erreur: %v", sendErr)
			return sendErr
		}
		return err
	}

	log.Printf("[AGENT] handleFileUpload - Upload réussi, envoi de confirmation")
	// Confirmer l'upload
	completeMsg := common.NewMessageWithID(common.MessageTypeFileComplete, msg.ID, &common.FileData{
		Path: path,
	})
	completeMsg.AgentID = c.agentID
	if err := c.sendMessage(completeMsg); err != nil {
		log.Printf("[AGENT] handleFileUpload - ERREUR envoi message de confirmation: %v", err)
		return err
	}
	log.Printf("[AGENT] handleFileUpload - Confirmation envoyée avec succès")
	return nil
}

// handleFileDownload traite le téléchargement de fichier
func (c *Client) handleFileDownload(msg *common.Message) error {
	var path string

	// Gérer les différents types de données
	switch data := msg.Data.(type) {
	case *common.FileData:
		path = data.Path
	case map[string]interface{}:
		if pathValue, exists := data["path"]; exists {
			if pathStr, ok := pathValue.(string); ok {
				path = pathStr
			}
		}
	case string:
		path = data
	default:
		return fmt.Errorf("format de données de fichier invalide: %T", msg.Data)
	}

	if path == "" {
		return fmt.Errorf("chemin de fichier manquant")
	}

	// Télécharger le fichier
	chunks, err := c.fileManager.DownloadFile(path)
	if err != nil {
		errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
			Code:    "DOWNLOAD_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	// Envoyer les chunks
	for _, chunk := range chunks {
		chunkMsg := common.NewMessageWithID(common.MessageTypeFileChunk, msg.ID, chunk)
		chunkMsg.AgentID = c.agentID
		if err := c.sendMessage(chunkMsg); err != nil {
			return err
		}
	}

	// Confirmer la fin du téléchargement
	completeMsg := common.NewMessageWithID(common.MessageTypeFileComplete, msg.ID, &common.FileData{
		Path: path,
	})
	completeMsg.AgentID = c.agentID
	return c.sendMessage(completeMsg)
}

// handleFileList traite la demande de liste de fichiers
func (c *Client) handleFileList(msg *common.Message) error {
	log.Printf("[AGENT] handleFileList - Début du traitement, ID: %s", msg.ID)
	log.Printf("[AGENT] handleFileList - Type de données: %T", msg.Data)
	
	var path string

	// Gérer les différents types de données
	switch data := msg.Data.(type) {
	case *common.FileData:
		path = data.Path
	case map[string]interface{}:
		if pathValue, exists := data["path"]; exists {
			if pathStr, ok := pathValue.(string); ok {
				path = pathStr
			}
		}
	case string:
		path = data
	case []interface{}:
		// Le serveur envoie parfois une liste de fichiers (réponse) mais pas de demande
		// Si on reçoit []interface{} comme demande, c'est une erreur de format
		log.Printf("[AGENT] handleFileList - Reçu []interface{} avec %d éléments (cas non attendu pour une demande)", len(data))
		
		// Si c'est une liste vide, traiter comme une demande pour la racine
		if len(data) == 0 {
			log.Printf("[AGENT] handleFileList - Liste vide, traitement comme demande de liste pour chemin racine")
			path = "/"
			goto listFiles
		}
		
		// Sinon, c'est probablement une erreur de sérialisation
		// Essayer d'extraire un chemin si possible depuis le premier élément
		if len(data) > 0 {
			if firstItem, ok := data[0].(map[string]interface{}); ok {
				if pathValue, exists := firstItem["path"]; exists {
					if pathStr, ok := pathValue.(string); ok {
						// C'est peut-être une réponse mal formatée, mais on peut essayer
						log.Printf("[AGENT] handleFileList - Élément trouvé avec path: %s, utilisation comme chemin parent", pathStr)
						// Extraire le répertoire parent
						if strings.HasPrefix(pathStr, "/") && pathStr != "/" {
							lastSlash := strings.LastIndex(pathStr, "/")
							if lastSlash > 0 {
								path = pathStr[:lastSlash]
							} else {
								path = "/"
							}
						} else {
							path = "/"
						}
						goto listFiles
					}
				}
			}
		}
		
		// Si on arrive ici, on ne peut pas traiter cette liste comme une demande
		log.Printf("[AGENT] handleFileList - Format []interface{} non traitable comme demande, retour d'erreur")
		return fmt.Errorf("format de données invalide pour une demande: []interface{} avec %d éléments", len(data))
	default:
		return fmt.Errorf("format de données de fichier invalide: %T", msg.Data)
	}

	if path == "" {
		// Utiliser le répertoire racine par défaut
		path = "/"
	}

listFiles:
	log.Printf("[AGENT] handleFileList - Chemin final à lister: %s", path)
	
	// Lister les fichiers
	files, err := c.fileManager.ListFiles(path)
	if err != nil {
		log.Printf("[AGENT] handleFileList - ERREUR lors du listing: %v", err)
		errorMsg := common.NewMessageWithID(common.MessageTypeError, msg.ID, &common.ErrorData{
			Code:    "LIST_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		if sendErr := c.sendMessage(errorMsg); sendErr != nil {
			log.Printf("[AGENT] handleFileList - ERREUR lors de l'envoi du message d'erreur: %v", sendErr)
		}
		return err
	}

	log.Printf("[AGENT] handleFileList - %d fichiers récupérés, envoi de la réponse", len(files))
	
	// Envoyer la liste des fichiers
	responseMsg := common.NewMessageWithID(common.MessageTypeFileList, msg.ID, files)
	responseMsg.AgentID = c.agentID
	if err := c.sendMessage(responseMsg); err != nil {
		log.Printf("[AGENT] handleFileList - ERREUR lors de l'envoi de la réponse: %v", err)
		return err
	}
	
	log.Printf("[AGENT] handleFileList - Réponse envoyée avec succès")
	return nil
}

// handleFileDelete traite la suppression de fichier
func (c *Client) handleFileDelete(msg *common.Message) error {
	var path string

	// Gérer les différents types de données
	switch data := msg.Data.(type) {
	case *common.FileData:
		path = data.Path
	case map[string]interface{}:
		if pathValue, exists := data["path"]; exists {
			if pathStr, ok := pathValue.(string); ok {
				path = pathStr
			}
		}
	case string:
		path = data
	default:
		errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
			Code:    "INVALID_DATA",
			Message: fmt.Sprintf("format de données invalide: %T", msg.Data),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	if path == "" {
		errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
			Code:    "MISSING_PATH",
			Message: "chemin de fichier manquant",
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	// Supprimer le fichier
	if err := c.fileManager.DeleteFile(path); err != nil {
		errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
			Code:    "DELETE_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	// Confirmer la suppression
	completeMsg := common.NewMessageWithID(common.MessageTypeFileComplete, msg.ID, &common.FileData{
		Path: path,
	})
	completeMsg.AgentID = c.agentID
	return c.sendMessage(completeMsg)
}

// handleFileCreateDir traite la création de répertoire
func (c *Client) handleFileCreateDir(msg *common.Message) error {
	var path string

	// Gérer les différents types de données
	switch data := msg.Data.(type) {
	case *common.FileData:
		path = data.Path
	case map[string]interface{}:
		if pathValue, exists := data["path"]; exists {
			if pathStr, ok := pathValue.(string); ok {
				path = pathStr
			}
		}
	case string:
		path = data
	default:
		errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
			Code:    "INVALID_DATA",
			Message: fmt.Sprintf("format de données invalide: %T", msg.Data),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	if path == "" {
		errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
			Code:    "MISSING_PATH",
			Message: "chemin de répertoire manquant",
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	// Créer le répertoire
	if err := c.fileManager.CreateDirectory(path); err != nil {
		errorMsg := common.NewMessageWithID(common.MessageTypeFileError, msg.ID, &common.ErrorData{
			Code:    "CREATE_DIR_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	// Confirmer la création
	completeMsg := common.NewMessageWithID(common.MessageTypeFileComplete, msg.ID, &common.FileData{
		Path: path,
	})
	completeMsg.AgentID = c.agentID
	return c.sendMessage(completeMsg)
}

// sendMessage envoie un message via WebSocket
func (c *Client) sendMessage(msg *common.Message) error {
	c.mu.RLock()
	conn := c.conn
	connected := c.connected
	c.mu.RUnlock()

	if !connected || conn == nil {
		return fmt.Errorf("pas de connexion WebSocket")
	}

	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("sérialisation du message échouée: %v", err)
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// sendHeartbeat envoie des heartbeats périodiques
func (c *Client) sendHeartbeat() {
	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			heartbeat := common.NewMessage(common.MessageTypeHeartbeat, nil)
			heartbeat.AgentID = c.agentID
			if err := c.sendMessage(heartbeat); err != nil {
				log.Printf("Erreur d'envoi du heartbeat: %v", err)
				c.disconnect()
				// Signaler la déconnexion pour relancer la reconnexion
				select {
				case c.disconnectChan <- struct{}{}:
				default:
				}
				return
			}
		case <-c.stopChan:
			return
		}
	}
}

// sendPrinterStatus envoie périodiquement le statut des imprimantes
func (c *Client) sendPrinterStatus() {
	ticker := time.NewTicker(60 * time.Second) // Toutes les minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			printers, err := c.printerMonitor.GetPrinters()
			if err != nil {
				log.Printf("Erreur de récupération des imprimantes: %v", err)
				continue
			}

			statusMsg := common.NewMessage(common.MessageTypePrinterStatus, printers)
			statusMsg.AgentID = c.agentID
			if err := c.sendMessage(statusMsg); err != nil {
				log.Printf("Erreur d'envoi du statut des imprimantes: %v", err)
			}
		case <-c.stopChan:
			return
		}
	}
}

// disconnect ferme la connexion
func (c *Client) disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false

	log.Println("Déconnecté du serveur")
}

// handleServiceList traite une demande de liste de services
func (c *Client) handleServiceList(msg *common.Message) error {
	log.Printf("[AGENT] handleServiceList - Demande reçue avec ID: %s", msg.ID)

	services, err := c.serviceManager.ListServices()
	if err != nil {
		log.Printf("[AGENT] handleServiceList - ERREUR lors de la récupération des services: %v", err)
		errorMsg := common.NewMessageWithID(common.MessageTypeError, msg.ID, &common.ErrorData{
			Code:    "SERVICE_LIST_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	log.Printf("[AGENT] handleServiceList - %d services récupérés", len(services))
	for i, svc := range services {
		log.Printf("[AGENT] handleServiceList - Service %d: %s (%s) - %s", i, svc.Name, svc.Type, svc.State)
	}

	// Envoyer la liste des services
	responseMsg := common.NewMessageWithID(common.MessageTypeServiceList, msg.ID, services)
	responseMsg.AgentID = c.agentID
	log.Printf("[AGENT] handleServiceList - Envoi de la réponse avec ID: %s", msg.ID)

	err = c.sendMessage(responseMsg)
	if err != nil {
		log.Printf("[AGENT] handleServiceList - ERREUR lors de l'envoi de la réponse: %v", err)
	} else {
		log.Printf("[AGENT] handleServiceList - Réponse envoyée avec succès")
	}

	return err
}

// handleServiceStatus traite une demande de statut de service
func (c *Client) handleServiceStatus(msg *common.Message) error {
	var serviceInfo *common.ServiceInfo

	// Gérer les différents types de données
	switch data := msg.Data.(type) {
	case *common.ServiceInfo:
		serviceInfo = data
	case map[string]interface{}:
		serviceInfo = &common.ServiceInfo{}
		if name, exists := data["name"]; exists {
			if nameStr, ok := name.(string); ok {
				serviceInfo.Name = nameStr
			}
		}
		if svcType, exists := data["type"]; exists {
			if typeStr, ok := svcType.(string); ok {
				serviceInfo.Type = typeStr
			}
		}
	default:
		return fmt.Errorf("format de données de service invalide: %T", msg.Data)
	}

	status, err := c.serviceManager.GetServiceStatus(serviceInfo.Name, serviceInfo.Type)
	if err != nil {
		errorMsg := common.NewMessageWithID(common.MessageTypeError, msg.ID, &common.ErrorData{
			Code:    "SERVICE_STATUS_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	responseMsg := common.NewMessageWithID(common.MessageTypeServiceStatus, msg.ID, status)
	responseMsg.AgentID = c.agentID
	return c.sendMessage(responseMsg)
}

// handleServiceAction traite une action sur un service
func (c *Client) handleServiceAction(msg *common.Message) error {
	var action *common.ServiceAction

	// Gérer les différents types de données
	switch data := msg.Data.(type) {
	case *common.ServiceAction:
		action = data
	case map[string]interface{}:
		action = &common.ServiceAction{}
		if name, exists := data["name"]; exists {
			if nameStr, ok := name.(string); ok {
				action.Name = nameStr
			}
		}
		if svcType, exists := data["type"]; exists {
			if typeStr, ok := svcType.(string); ok {
				action.Type = typeStr
			}
		}
		if act, exists := data["action"]; exists {
			if actStr, ok := act.(string); ok {
				action.Action = actStr
			}
		}
	default:
		return fmt.Errorf("format de données d'action invalide: %T", msg.Data)
	}

	result, err := c.serviceManager.ExecuteAction(action)
	if err != nil {
		errorMsg := common.NewMessageWithID(common.MessageTypeError, msg.ID, &common.ErrorData{
			Code:    "SERVICE_ACTION_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	responseMsg := common.NewMessageWithID(common.MessageTypeServiceResult, msg.ID, result)
	responseMsg.AgentID = c.agentID
	return c.sendMessage(responseMsg)
}

// handleLogList traite une demande de liste des sources de logs
func (c *Client) handleLogList(msg *common.Message) error {
	sources, err := c.logManager.ListLogSources()
	if err != nil {
		errorMsg := common.NewMessageWithID(common.MessageTypeError, msg.ID, &common.ErrorData{
			Code:    "LOG_LIST_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	responseMsg := common.NewMessageWithID(common.MessageTypeLogList, msg.ID, sources)
	responseMsg.AgentID = c.agentID
	return c.sendMessage(responseMsg)
}

// handleLogContent traite une demande de contenu de logs
func (c *Client) handleLogContent(msg *common.Message) error {
	log.Printf("[AGENT] handleLogContent - Demande reçue avec ID: %s", msg.ID)
	
	var logReq *common.LogRequest

	// Gérer les différents types de données
	switch data := msg.Data.(type) {
	case *common.LogRequest:
		logReq = data
		log.Printf("[AGENT] handleLogContent - Données LogRequest: Source=%s, Type=%s, Lines=%d", logReq.Source, logReq.Type, logReq.Lines)
	case map[string]interface{}:
		logReq = &common.LogRequest{}
		if source, exists := data["source"]; exists {
			if sourceStr, ok := source.(string); ok {
				logReq.Source = sourceStr
			}
		}
		if logType, exists := data["type"]; exists {
			if typeStr, ok := logType.(string); ok {
				logReq.Type = typeStr
			}
		}
		if lines, exists := data["lines"]; exists {
			if linesFloat, ok := lines.(float64); ok {
				logReq.Lines = int(linesFloat)
			}
		}
		if path, exists := data["path"]; exists {
			if pathStr, ok := path.(string); ok {
				logReq.Path = pathStr
			}
		}
		if unit, exists := data["unit"]; exists {
			if unitStr, ok := unit.(string); ok {
				logReq.Unit = unitStr
			}
		}
		if priority, exists := data["priority"]; exists {
			if priorityStr, ok := priority.(string); ok {
				logReq.Priority = priorityStr
			}
		}
		if since, exists := data["since"]; exists {
			if sinceStr, ok := since.(string); ok {
				logReq.Since = sinceStr
			}
		}
		if until, exists := data["until"]; exists {
			if untilStr, ok := until.(string); ok {
				logReq.Until = untilStr
			}
		}
		log.Printf("[AGENT] handleLogContent - Données map: Source=%s, Type=%s, Lines=%d", logReq.Source, logReq.Type, logReq.Lines)
	default:
		log.Printf("[AGENT] handleLogContent - Format de données invalide: %T", msg.Data)
		return fmt.Errorf("format de données de requête log invalide: %T", msg.Data)
	}

	log.Printf("[AGENT] handleLogContent - Récupération des logs...")
	logs, err := c.logManager.GetLogs(logReq)
	if err != nil {
		log.Printf("[AGENT] handleLogContent - ERREUR lors de la récupération des logs: %v", err)
		errorMsg := common.NewMessageWithID(common.MessageTypeError, msg.ID, &common.ErrorData{
			Code:    "LOG_CONTENT_ERROR",
			Message: err.Error(),
		})
		errorMsg.AgentID = c.agentID
		return c.sendMessage(errorMsg)
	}

	log.Printf("[AGENT] handleLogContent - %d logs récupérés", len(logs))
	
	// Encapsuler les logs dans un objet avec la clé "entries"
	logData := map[string]interface{}{
		"entries": logs,
		"source":  logReq.Source,
		"count":   len(logs),
	}

	responseMsg := common.NewMessageWithID(common.MessageTypeLogContent, msg.ID, logData)
	responseMsg.AgentID = c.agentID
	log.Printf("[AGENT] handleLogContent - Envoi de la réponse avec ID: %s", msg.ID)
	
	err = c.sendMessage(responseMsg)
	if err != nil {
		log.Printf("[AGENT] handleLogContent - ERREUR lors de l'envoi de la réponse: %v", err)
	} else {
		log.Printf("[AGENT] handleLogContent - Réponse envoyée avec succès")
	}
	
	return err
}
