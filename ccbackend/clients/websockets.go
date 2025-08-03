package clients

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type Client struct {
	ID                 string
	ClientConn         *websocket.Conn
	SlackIntegrationID string
	AgentID            uuid.UUID
}

type MessageHandlerFunc func(client *Client, msg any)
type ConnectionHookFunc func(clientID string) error
type APIKeyValidatorFunc func(apiKey string) (string, error)

type WebSocketClient struct {
	clients            []*Client
	mutex              sync.RWMutex
	upgrader           websocket.Upgrader
	messageHandlers    []MessageHandlerFunc
	connectionHooks    []ConnectionHookFunc
	disconnectionHooks []ConnectionHookFunc
	apiKeyValidator    APIKeyValidatorFunc
}

func NewWebSocketClient(apiKeyValidator APIKeyValidatorFunc) *WebSocketClient {
	return &WebSocketClient{
		clients: make([]*Client, 0),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
		messageHandlers:    make([]MessageHandlerFunc, 0),
		connectionHooks:    make([]ConnectionHookFunc, 0),
		disconnectionHooks: make([]ConnectionHookFunc, 0),
		apiKeyValidator:    apiKeyValidator,
	}
}

func (ws *WebSocketClient) RegisterWithRouter(router *mux.Router) {
	log.Printf("🚀 Registering WebSocket server on /ws endpoint")
	router.HandleFunc("/ws", ws.handleWebSocketConnection)
	log.Printf("✅ WebSocket server registered on /ws")
}

func (ws *WebSocketClient) handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔗 New WebSocket connection attempt from %s", r.RemoteAddr)

	// Extract and validate API key before upgrading connection
	apiKey := r.Header.Get("X-CCAGENT-API-KEY")
	if apiKey == "" {
		log.Printf("❌ Rejecting WebSocket connection from %s: missing X-CCAGENT-API-KEY header", r.RemoteAddr)
		http.Error(w, "Missing X-CCAGENT-API-KEY header", http.StatusUnauthorized)
		return
	}

	// Extract agent ID and validate it's a UUID
	agentIDStr := r.Header.Get("X-CCAGENT-ID")
	if agentIDStr == "" {
		log.Printf("❌ No X-CCAGENT-ID provided, rejecting connection")
		http.Error(w, "Invalid X-CCAGENT-ID header", http.StatusBadRequest)
		return
	}

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		log.Printf("❌ Rejecting WebSocket connection from %s: invalid agent ID format (must be UUID): %s", r.RemoteAddr, agentIDStr)
		http.Error(w, "Invalid X-CCAGENT-ID format (must be UUID)", http.StatusBadRequest)
		return
	}

	// Validate API key
	slackIntegrationID, err := ws.apiKeyValidator(apiKey)
	if err != nil {
		log.Printf("❌ Rejecting WebSocket connection from %s: invalid API key: %v", r.RemoteAddr, err)
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed from %s: %v", r.RemoteAddr, err)
		return
	}
	defer func() {
		log.Printf("🔌 Closing WebSocket connection")
		conn.Close()
	}()

	client := &Client{
		ID:                 uuid.New().String(),
		ClientConn:         conn,
		SlackIntegrationID: slackIntegrationID,
		AgentID:            agentID,
	}
	ws.addClient(client)
	log.Printf("✅ WebSocket client connected with ID: %s from %s", client.ID, r.RemoteAddr)
	ws.invokeConnectionHooks(client.ID)
	defer func() {
		ws.invokeDisconnectionHooks(client.ID)
		ws.removeClient(client.ID)
	}()

	log.Printf("👂 Starting message listener for client %s", client.ID)
	for {
		var msg any
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket unexpected error from client %s: %v", client.ID, err)
			} else {
				log.Printf("🔌 WebSocket connection closed for client %s (normal closure)", client.ID)
			}
			break
		}

		log.Printf("📥 Raw message received from client %s", client.ID)
		ws.invokeMessageHandlers(client, msg)
	}
	log.Printf("🛑 Message listener stopped for client %s", client.ID)
}

func (ws *WebSocketClient) addClient(client *Client) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.clients = append(ws.clients, client)
	log.Printf("📊 Client %s added to active connections. Total clients: %d", client.ID, len(ws.clients))
}

func (ws *WebSocketClient) removeClient(clientID string) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	for i, client := range ws.clients {
		if client.ID == clientID {
			ws.clients = append(ws.clients[:i], ws.clients[i+1:]...)
			log.Printf("🔌 WebSocket client %s disconnected. Remaining clients: %d", clientID, len(ws.clients))
			return
		}
	}
	log.Printf("⚠️ Attempted to remove client %s but not found in active connections", clientID)
}

func (ws *WebSocketClient) GetClientIDs() []string {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	clientIDs := make([]string, len(ws.clients))
	for i, client := range ws.clients {
		clientIDs[i] = client.ID
	}
	log.Printf("📋 Retrieved %d active client IDs", len(clientIDs))
	return clientIDs
}

func (ws *WebSocketClient) getClientByID(clientID string) *Client {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	for _, client := range ws.clients {
		if client.ID == clientID {
			log.Printf("🔍 Found client %s in active connections", clientID)
			return client
		}
	}
	log.Printf("❌ Client %s not found in active connections", clientID)
	return nil
}

func (ws *WebSocketClient) GetClientByID(clientID string) *Client {
	return ws.getClientByID(clientID)
}

func (ws *WebSocketClient) GetSlackIntegrationIDByClientID(clientID string) string {
	client := ws.getClientByID(clientID)
	if client == nil {
		return ""
	}
	return client.SlackIntegrationID
}

func (ws *WebSocketClient) SendMessage(clientID string, msg any) error {
	log.Printf("📤 Attempting to send message to client %s", clientID)
	client := ws.getClientByID(clientID)
	if client == nil {
		log.Printf("❌ Cannot send message: client %s not found", clientID)
		return fmt.Errorf("client with ID %s not found", clientID)
	}

	if err := client.ClientConn.WriteJSON(msg); err != nil {
		log.Printf("❌ Failed to send message to client %s: %v", clientID, err)
		return err
	}

	log.Printf("✅ Message sent successfully to client %s", clientID)
	return nil
}

func (ws *WebSocketClient) RegisterMessageHandler(handler MessageHandlerFunc) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.messageHandlers = append(ws.messageHandlers, handler)
	log.Printf("📝 Message handler registered. Total handlers: %d", len(ws.messageHandlers))
}

func (ws *WebSocketClient) RegisterConnectionHook(hook ConnectionHookFunc) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.connectionHooks = append(ws.connectionHooks, hook)
	log.Printf("🔗 Connection hook registered. Total connection hooks: %d", len(ws.connectionHooks))
}

func (ws *WebSocketClient) RegisterDisconnectionHook(hook ConnectionHookFunc) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.disconnectionHooks = append(ws.disconnectionHooks, hook)
	log.Printf("🔌 Disconnection hook registered. Total disconnection hooks: %d", len(ws.disconnectionHooks))
}

func (ws *WebSocketClient) invokeMessageHandlers(client *Client, msg any) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	log.Printf("🔄 Invoking %d message handlers for client %s", len(ws.messageHandlers), client.ID)
	for i, handler := range ws.messageHandlers {
		log.Printf("🎯 Executing handler %d for client %s", i+1, client.ID)
		handler(client, msg)
	}
	log.Printf("✅ All message handlers completed for client %s", client.ID)
}

func (ws *WebSocketClient) invokeConnectionHooks(clientID string) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	log.Printf("🔗 Invoking %d connection hooks for client %s", len(ws.connectionHooks), clientID)
	for i, hook := range ws.connectionHooks {
		log.Printf("🎯 Executing connection hook %d for client %s", i+1, clientID)
		if err := hook(clientID); err != nil {
			log.Printf("❌ Connection hook %d failed for client %s: %v", i+1, clientID, err)
		}
	}
	log.Printf("✅ All connection hooks completed for client %s", clientID)
}

func (ws *WebSocketClient) invokeDisconnectionHooks(clientID string) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	log.Printf("🔌 Invoking %d disconnection hooks for client %s", len(ws.disconnectionHooks), clientID)
	for i, hook := range ws.disconnectionHooks {
		log.Printf("🎯 Executing disconnection hook %d for client %s", i+1, clientID)
		if err := hook(clientID); err != nil {
			log.Printf("❌ Disconnection hook %d failed for client %s: %v", i+1, clientID, err)
		}
	}
	log.Printf("✅ All disconnection hooks completed for client %s", clientID)
}

