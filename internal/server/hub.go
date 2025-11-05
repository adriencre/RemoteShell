package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"remoteshell/internal/common"
)

// Agent représente un agent connecté
type Agent struct {
	ID         string
	Name       string
	Conn       WebSocketConn
	LastSeen   time.Time
	Printers   []*common.PrinterInfo
	SystemInfo *common.SystemInfo
	FileCache  map[string][]*common.FileData // Cache des fichiers par chemin
	Services   []*common.ServiceInfo         // Cache des services
	LogSources []*common.LogSource           // Cache des sources de logs
	responses  map[string]chan *common.Message
	mu         sync.RWMutex
}

// WebSocketConn encapsule une connexion WebSocket
type WebSocketConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
	RemoteAddr() string
	SendMessage(msg *common.Message) error
}

// WebClient représente un client web connecté
type WebClient struct {
	ID   string
	Conn WebSocketConn
	mu   sync.RWMutex
}

// AgentMetadata contient les métadonnées d'un agent (franchise, category, etc.)
type AgentMetadata struct {
	Franchise string `json:"franchise"`
	Category  string `json:"category"`
}

// Hub gère les connexions des agents et des clients web
type Hub struct {
	agents        map[string]*Agent
	webClients    map[string]*WebClient
	metadata      map[string]*AgentMetadata // Métadonnées des agents (franchise, category)
	register      chan *Agent
	unregister    chan *Agent
	registerWeb   chan *WebClient
	unregisterWeb chan *WebClient
	broadcast     chan *common.Message
	db            *Database // Référence à la base de données pour sauvegarder les agents
	mu            sync.RWMutex
}

// NewHub crée un nouveau hub
func NewHub(db *Database) *Hub {
	return &Hub{
		agents:        make(map[string]*Agent),
		webClients:    make(map[string]*WebClient),
		metadata:      make(map[string]*AgentMetadata),
		register:      make(chan *Agent),
		unregister:    make(chan *Agent),
		registerWeb:   make(chan *WebClient),
		unregisterWeb: make(chan *WebClient),
		broadcast:     make(chan *common.Message, 256),
		db:            db,
	}
}

// Run démarre le hub
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second) // Nettoyage périodique
	defer ticker.Stop()
	
	// Ticker pour mettre à jour LastSeen des agents actifs
	updateTicker := time.NewTicker(60 * time.Second) // Mise à jour toutes les minutes
	defer updateTicker.Stop()

	for {
		select {
		case agent := <-h.register:
			h.registerAgent(agent)

		case agent := <-h.unregister:
			h.unregisterAgent(agent)

		case webClient := <-h.registerWeb:
			h.registerWebClient(webClient)

		case webClient := <-h.unregisterWeb:
			h.unregisterWebClient(webClient)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-ticker.C:
			h.cleanupInactiveAgents()
			
		case <-updateTicker.C:
			// Mettre à jour LastSeen des agents actifs dans la base de données
			h.updateAgentsLastSeen()
		}
	}
}

// registerAgent enregistre un nouvel agent
func (h *Hub) registerAgent(agent *Agent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Initialiser le map des réponses et le cache de fichiers
	agent.responses = make(map[string]chan *common.Message)
	agent.FileCache = make(map[string][]*common.FileData)

	h.agents[agent.ID] = agent
	log.Printf("Agent enregistré: %s (%s) depuis %s", agent.Name, agent.ID, agent.Conn.RemoteAddr())
	
	// Charger les métadonnées depuis la base de données si elles existent
	if h.db != nil {
		existingAgent, err := h.db.GetAgent(agent.ID)
		if err == nil && existingAgent != nil {
			// Charger les métadonnées depuis la base de données
			h.metadata[agent.ID] = &AgentMetadata{
				Franchise: existingAgent.Franchise,
				Category:  existingAgent.Category,
			}
		}
		
		// Sauvegarder ou mettre à jour l'agent dans la base de données
		agentRecord := &AgentRecord{
			ID:        agent.ID,
			Name:      agent.Name,
			LastSeen:  agent.LastSeen,
			Status:    "online",
			IPAddress: agent.Conn.RemoteAddr(),
			UpdatedAt: time.Now(),
		}
		// Préserver les métadonnées existantes si elles existent
		if existingAgent != nil {
			agentRecord.Franchise = existingAgent.Franchise
			agentRecord.Category = existingAgent.Category
		}
		if err := h.db.SaveAgent(agentRecord); err != nil {
			log.Printf("Erreur lors de la sauvegarde de l'agent %s en base: %v", agent.ID, err)
		} else {
			log.Printf("Agent %s sauvegardé dans la base de données", agent.ID)
		}
	}
}

// unregisterAgent désenregistre un agent
func (h *Hub) unregisterAgent(agent *Agent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.agents[agent.ID]; exists {
		delete(h.agents, agent.ID)
		log.Printf("Agent désenregistré: %s (%s)", agent.Name, agent.ID)
		
		// Mettre à jour le statut dans la base de données
		if h.db != nil {
			agentRecord := &AgentRecord{
				ID:        agent.ID,
				Status:    "offline",
				LastSeen:  time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := h.db.SaveAgent(agentRecord); err != nil {
				log.Printf("Erreur lors de la mise à jour du statut de l'agent %s: %v", agent.ID, err)
			}
		}
	}
}

// broadcastMessage diffuse un message à tous les agents
func (h *Hub) broadcastMessage(message *common.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, agent := range h.agents {
		select {
		case <-time.After(5 * time.Second):
			log.Printf("Timeout d'envoi de message à l'agent %s", agent.ID)
		default:
			if err := agent.SendMessage(message); err != nil {
				log.Printf("Erreur d'envoi de message à l'agent %s: %v", agent.ID, err)
			}
		}
	}
}

// SendToAgent envoie un message à un agent spécifique
func (h *Hub) SendToAgent(agentID string, message *common.Message) error {
	h.mu.RLock()
	agent, exists := h.agents[agentID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent %s non trouvé", agentID)
	}

	return agent.SendMessage(message)
}

// SendMessageWithResponse envoie un message à un agent et attend une réponse
func (h *Hub) SendMessageWithResponse(agentID string, message *common.Message, timeout time.Duration) (*common.Message, error) {
	h.mu.RLock()
	agent, exists := h.agents[agentID]
	h.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %s non trouvé", agentID)
	}

	return agent.SendMessageWithResponse(message, timeout)
}

// GetAgent retourne un agent par son ID
func (h *Hub) GetAgent(agentID string) (*Agent, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agent, exists := h.agents[agentID]
	return agent, exists
}

// GetAgents retourne la liste de tous les agents
func (h *Hub) GetAgents() []*Agent {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]*Agent, 0, len(h.agents))
	for _, agent := range h.agents {
		agents = append(agents, agent)
	}
	return agents
}

// GetAgentCount retourne le nombre d'agents connectés
func (h *Hub) GetAgentCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.agents)
}

// cleanupInactiveAgents nettoie les agents inactifs
func (h *Hub) cleanupInactiveAgents() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for id, agent := range h.agents {
		if now.Sub(agent.LastSeen) > 5*time.Minute {
			log.Printf("Nettoyage de l'agent inactif: %s (%s)", agent.Name, id)
			agent.Conn.Close()
			delete(h.agents, id)
		}
	}
}

// updateAgentsLastSeen met à jour LastSeen des agents actifs dans la base de données
func (h *Hub) updateAgentsLastSeen() {
	if h.db == nil {
		return
	}
	
	h.mu.RLock()
	agents := make([]*Agent, 0, len(h.agents))
	for _, agent := range h.agents {
		agents = append(agents, agent)
	}
	h.mu.RUnlock()
	
	for _, agent := range agents {
		agentRecord := &AgentRecord{
			ID:        agent.ID,
			Name:      agent.Name,
			LastSeen:  agent.LastSeen,
			Status:    "online",
			UpdatedAt: time.Now(),
		}
		if err := h.db.SaveAgent(agentRecord); err != nil {
			log.Printf("Erreur lors de la mise à jour LastSeen de l'agent %s: %v", agent.ID, err)
		}
	}
}

// SendMessage envoie un message à l'agent
func (a *Agent) SendMessage(message *common.Message) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := message.ToJSON()
	if err != nil {
		return err
	}

	return a.Conn.WriteMessage(1, data) // 1 = TextMessage
}

// SendMessageWithResponse envoie un message et attend une réponse
func (a *Agent) SendMessageWithResponse(message *common.Message, timeout time.Duration) (*common.Message, error) {
	// Générer un ID unique pour le message si pas déjà présent
	if message.ID == "" {
		message.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Créer un canal pour la réponse
	responseChan := make(chan *common.Message, 1)

	a.mu.Lock()
	a.responses[message.ID] = responseChan
	a.mu.Unlock()

	// Nettoyer le canal après utilisation
	defer func() {
		a.mu.Lock()
		delete(a.responses, message.ID)
		a.mu.Unlock()
	}()

	// Envoyer le message
	if err := a.SendMessage(message); err != nil {
		return nil, err
	}

	// Attendre la réponse avec timeout
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout en attendant la réponse")
	}
}

// HandleResponse traite une réponse reçue d'un agent
func (a *Agent) HandleResponse(response *common.Message) {
	a.mu.RLock()
	responseChan, exists := a.responses[response.ID]
	a.mu.RUnlock()

	if exists {
		select {
		case responseChan <- response:
		default:
			// Canal plein, ignorer la réponse
		}
	}
}

// UpdateLastSeen met à jour le timestamp de dernière activité
func (a *Agent) UpdateLastSeen() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.LastSeen = time.Now()
}

// UpdatePrinters met à jour les informations des imprimantes
func (a *Agent) UpdatePrinters(printers []*common.PrinterInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.Printers = printers
}

// UpdateSystemInfo met à jour les informations système
func (a *Agent) UpdateSystemInfo(systemInfo *common.SystemInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.SystemInfo = systemInfo
}

// GetPrinters retourne les informations des imprimantes
func (a *Agent) GetPrinters() []*common.PrinterInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.Printers
}

// GetSystemInfo retourne les informations système
func (a *Agent) GetSystemInfo() *common.SystemInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.SystemInfo
}

// IsActive vérifie si l'agent est actif
func (a *Agent) IsActive() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return time.Since(a.LastSeen) < 5*time.Minute
}

// UpdateFileCache met à jour le cache de fichiers pour un chemin
func (a *Agent) UpdateFileCache(path string, files []*common.FileData) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.FileCache[path] = files
}

// GetFileCache retourne le cache de fichiers pour un chemin
func (a *Agent) GetFileCache(path string) ([]*common.FileData, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	files, exists := a.FileCache[path]
	return files, exists
}

// UpdateServices met à jour les informations des services
func (a *Agent) UpdateServices(services []*common.ServiceInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.Services = services
}

// GetServices retourne les informations des services
func (a *Agent) GetServices() []*common.ServiceInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.Services
}

// UpdateLogSources met à jour les sources de logs
func (a *Agent) UpdateLogSources(sources []*common.LogSource) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.LogSources = sources
}

// GetLogSources retourne les sources de logs
func (a *Agent) GetLogSources() []*common.LogSource {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.LogSources
}

// registerWebClient enregistre un nouveau client web
func (h *Hub) registerWebClient(client *WebClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.webClients[client.ID] = client
	log.Printf("Client web enregistré: %s depuis %s", client.ID, client.Conn.RemoteAddr())
}

// unregisterWebClient désenregistre un client web
func (h *Hub) unregisterWebClient(client *WebClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.webClients[client.ID]; exists {
		delete(h.webClients, client.ID)
		log.Printf("Client web désenregistré: %s", client.ID)
	}
}

// BroadcastToWebClients diffuse un message à tous les clients web
func (h *Hub) BroadcastToWebClients(message *common.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.webClients {
		go func(c *WebClient) {
			if err := c.Conn.SendMessage(message); err != nil {
				log.Printf("Erreur lors de l'envoi au client web %s: %v", c.ID, err)
			}
		}(client)
	}
}

// GetAgentMetadata retourne les métadonnées d'un agent
func (h *Hub) GetAgentMetadata(agentID string) *AgentMetadata {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	meta, exists := h.metadata[agentID]
	if !exists {
		// Retourner des métadonnées par défaut si elles n'existent pas
		return &AgentMetadata{Franchise: "", Category: ""}
	}
	return meta
}

// SetAgentMetadata définit les métadonnées d'un agent
func (h *Hub) SetAgentMetadata(agentID string, metadata *AgentMetadata) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metadata[agentID] = metadata
}
