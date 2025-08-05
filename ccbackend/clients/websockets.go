package clients

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/google/uuid"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"github.com/gorilla/mux"
)

type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type Client struct {
	ID                 string
	SocketConn         socketio.Conn
	SlackIntegrationID string
	AgentID            uuid.UUID
}

type MessageHandlerFunc func(client *Client, msg any)
type ConnectionHookFunc func(client *Client) error
type APIKeyValidatorFunc func(apiKey string) (string, error)

type WebSocketClient struct {
	server             *socketio.Server
	clients            []*Client
	mutex              sync.RWMutex
	messageHandlers    []MessageHandlerFunc
	connectionHooks    []ConnectionHookFunc
	disconnectionHooks []ConnectionHookFunc
	apiKeyValidator    APIKeyValidatorFunc
}

func NewWebSocketClient(apiKeyValidator APIKeyValidatorFunc) *WebSocketClient {
	// Create Socket.IO server with custom transports
	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			},
			&websocket.Transport{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			},
		},
	})

	ws := &WebSocketClient{
		server:             server,
		clients:            make([]*Client, 0),
		messageHandlers:    make([]MessageHandlerFunc, 0),
		connectionHooks:    make([]ConnectionHookFunc, 0),
		disconnectionHooks: make([]ConnectionHookFunc, 0),
		apiKeyValidator:    apiKeyValidator,
	}

	// Set up connection event handler
	server.OnConnect("/", func(s socketio.Conn) error {
		return ws.handleSocketConnection(s)
	})

	// Set up disconnection event handler
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		ws.handleSocketDisconnection(s, reason)
	})

	// Set up message handlers for all message types
	ws.setupMessageHandlers(server)

	return ws
}

func (ws *WebSocketClient) handleSocketConnection(s socketio.Conn) error {
	log.Printf("🔗 New Socket.IO connection from %s", s.RemoteAddr())

	// Extract and validate authentication from query parameters (since go-socket.io doesn't support custom headers easily)
	queryValues, err := url.ParseQuery(s.URL().RawQuery)
	if err != nil {
		log.Printf("❌ Failed to parse query parameters: %v", err)
		return fmt.Errorf("failed to parse query parameters")
	}

	// Extract and validate API key
	apiKey := queryValues.Get("api_key")
	if apiKey == "" {
		log.Printf("❌ Rejecting Socket.IO connection: missing api_key parameter")
		return fmt.Errorf("missing api_key parameter")
	}

	// Extract agent ID and validate it's a UUID
	agentIDStr := queryValues.Get("agent_id")
	if agentIDStr == "" {
		log.Printf("❌ No agent_id provided, rejecting connection")
		return fmt.Errorf("missing agent_id parameter")
	}

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		log.Printf("❌ Rejecting Socket.IO connection: invalid agent ID format (must be UUID): %s", agentIDStr)
		return fmt.Errorf("invalid agent_id format (must be UUID)")
	}

	// Validate API key
	slackIntegrationID, err := ws.apiKeyValidator(apiKey)
	if err != nil {
		log.Printf("❌ Rejecting Socket.IO connection: invalid API key: %v", err)
		return fmt.Errorf("invalid API key")
	}

	client := &Client{
		ID:                 s.ID(),
		SocketConn:         s,
		SlackIntegrationID: slackIntegrationID,
		AgentID:            agentID,
	}

	ws.addClient(client)
	log.Printf("✅ Socket.IO client connected with ID: %s", client.ID)
	ws.invokeConnectionHooks(client)

	return nil
}

func (ws *WebSocketClient) handleSocketDisconnection(s socketio.Conn, reason string) {
	log.Printf("🔌 Socket.IO client %s disconnected: %s", s.ID(), reason)

	client := ws.getClientByID(s.ID())
	if client != nil {
		ws.invokeDisconnectionHooks(client)
		ws.removeClient(client.ID)
	}
}

func (ws *WebSocketClient) setupMessageHandlers(server *socketio.Server) {
	// Handle generic message events - maintain compatibility with existing protocol
	server.OnEvent("/", "message", func(s socketio.Conn, data any) {
		log.Printf("📥 Message received from client %s", s.ID())
		client := ws.getClientByID(s.ID())
		if client != nil {
			ws.invokeMessageHandlers(client, data)
		}
	})

	// Handle specific event types for better Socket.IO integration
	eventTypes := []string{
		"start_conversation_v1",
		"user_message_v1",
		"assistant_message_v1",
		"system_message_v1",
		"job_complete_v1",
		"processing_slack_message_v1",
		"job_unassigned_v1",
		"check_idle_jobs_v1",
	}

	for _, eventType := range eventTypes {
		eventType := eventType // capture for closure
		server.OnEvent("/", eventType, func(s socketio.Conn, data any) {
			log.Printf("📥 Event '%s' received from client %s", eventType, s.ID())
			client := ws.getClientByID(s.ID())
			if client != nil {
				// Convert to compatible message format
				msg := Message{
					Type:    eventType,
					Payload: data,
				}
				ws.invokeMessageHandlers(client, msg)
			}
		})
	}
}

func (ws *WebSocketClient) RegisterWithRouter(router *mux.Router) {
	log.Printf("🚀 Registering Socket.IO server on /socket.io/ endpoint")
	router.Handle("/socket.io/", ws.server)
	log.Printf("✅ Socket.IO server registered on /socket.io/")
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
			log.Printf("🔌 Socket.IO client %s disconnected. Remaining clients: %d", clientID, len(ws.clients))
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

	// Check if message has a type field for event-based sending
	if msgMap, ok := msg.(map[string]any); ok {
		if msgType, hasType := msgMap["type"]; hasType {
			if typeStr, isString := msgType.(string); isString {
				// Send as specific event type
				client.SocketConn.Emit(typeStr, msgMap["payload"])
				log.Printf("✅ Event '%s' sent successfully to client %s", typeStr, clientID)
				return nil
			}
		}
	}

	// Fallback to generic message event
	client.SocketConn.Emit("message", msg)
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

func (ws *WebSocketClient) invokeConnectionHooks(client *Client) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	log.Printf("🔗 Invoking %d connection hooks for client %s", len(ws.connectionHooks), client.ID)
	for i, hook := range ws.connectionHooks {
		log.Printf("🎯 Executing connection hook %d for client %s", i+1, client.ID)
		if err := hook(client); err != nil {
			log.Printf("❌ Connection hook %d failed for client %s: %v", i+1, client.ID, err)
		}
	}
	log.Printf("✅ All connection hooks completed for client %s", client.ID)
}

func (ws *WebSocketClient) invokeDisconnectionHooks(client *Client) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	log.Printf("🔌 Invoking %d disconnection hooks for client %s", len(ws.disconnectionHooks), client.ID)
	for i, hook := range ws.disconnectionHooks {
		log.Printf("🎯 Executing disconnection hook %d for client %s", i+1, client.ID)
		if err := hook(client); err != nil {
			log.Printf("❌ Disconnection hook %d failed for client %s: %v", i+1, client.ID, err)
		}
	}
	log.Printf("✅ All disconnection hooks completed for client %s", client.ID)
}
