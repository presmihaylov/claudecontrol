package socketio

import (
	"ccbackend/clients"
	"ccbackend/core"
	"ccbackend/utils"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/zishang520/socket.io/v2/socket"
)

type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type MessageHandlerFunc func(client *clients.Client, msg any)
type ConnectionHookFunc func(client *clients.Client) error
type PingHandlerFunc func(client *clients.Client) error
type APIKeyValidatorFunc func(apiKey string) (string, error)

type SocketIOClientImpl struct {
	server             *socket.Server
	clients            []*clients.Client
	clientsBySocketID  map[string]*clients.Client
	mutex              sync.RWMutex
	messageHandlers    []MessageHandlerFunc
	connectionHooks    []ConnectionHookFunc
	disconnectionHooks []ConnectionHookFunc
	pingHooks          []PingHandlerFunc
	apiKeyValidator    APIKeyValidatorFunc
}

func NewSocketIOClient(apiKeyValidator APIKeyValidatorFunc) *SocketIOClientImpl {
	server := socket.NewServer(nil, nil)
	wsClient := &SocketIOClientImpl{
		server:             server,
		clients:            make([]*clients.Client, 0),
		clientsBySocketID:  make(map[string]*clients.Client),
		messageHandlers:    make([]MessageHandlerFunc, 0),
		connectionHooks:    make([]ConnectionHookFunc, 0),
		disconnectionHooks: make([]ConnectionHookFunc, 0),
		pingHooks:          make([]PingHandlerFunc, 0),
		apiKeyValidator:    apiKeyValidator,
	}

	// Set up Socket.IO connection handler
	err := server.On("connection", func(sockets ...any) {
		sock := sockets[0].(*socket.Socket)
		wsClient.handleSocketIOConnection(sock)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to register connection handler: %v", err))

	return wsClient
}

func (ws *SocketIOClientImpl) RegisterWithRouter(router *mux.Router) {
	log.Printf("🚀 Registering Socket.IO server on /socket.io/ endpoint")
	router.PathPrefix("/socket.io/").Handler(ws.server.ServeHandler(nil))
	log.Printf("✅ Socket.IO server registered on /socket.io/")
}

// getSocketIOHeader performs a case-insensitive lookup for a header in the headers map
func getSocketIOHeader(headers map[string][]string, headerName string) (string, bool) {
	for key, value := range headers {
		if strings.EqualFold(key, headerName) {
			if len(value) > 0 && value[0] != "" {
				return value[0], true
			}
		}
	}
	return "", false
}

func (ws *SocketIOClientImpl) handleSocketIOConnection(sock *socket.Socket) {
	log.Printf("🔗 New Socket.IO connection attempt, socket ID: %s", sock.Id())

	headers := sock.Handshake().Headers
	apiKey, exists := getSocketIOHeader(headers, "X-CCAGENT-API-KEY")
	if !exists {
		log.Printf("❌ Rejecting Socket.IO connection: missing X-CCAGENT-API-KEY header")
		sock.Disconnect(true)
		return
	}

	agentID, exists := getSocketIOHeader(headers, "X-CCAGENT-ID")
	if !exists {
		log.Printf("❌ No X-CCAGENT-ID provided, rejecting connection")
		sock.Disconnect(true)
		return
	}

	if !core.IsValidULID(agentID) {
		log.Printf("❌ Rejecting Socket.IO connection: agent ID must be a valid ULID")
		sock.Disconnect(true)
		return
	}

	// Validate API key
	slackIntegrationID, err := ws.apiKeyValidator(apiKey)
	if err != nil {
		log.Printf("❌ Rejecting Socket.IO connection: invalid API key: %v", err)
		sock.Disconnect(true)
		return
	}

	client := &clients.Client{
		ID:                 core.NewID("cl"),
		Socket:             sock,
		SlackIntegrationID: slackIntegrationID,
		AgentID:            agentID,
	}
	ws.addClient(client)
	log.Printf("✅ Socket.IO client connected with ID: %s, socket ID: %s", client.ID, sock.Id())
	ws.invokeConnectionHooks(client)

	// Set up message handler for cc_message event
	err = sock.On("cc_message", func(data ...any) {
		if len(data) == 0 {
			log.Printf("❌ No message data received for client %s", client.ID)
			return
		}

		log.Printf("📥 Raw message received from client %s", client.ID)
		ws.invokeMessageHandlers(client, data[0])
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up message handler for client %s: %v", client.ID, err))

	// Handle ping events
	err = sock.On("ping", func(data ...any) {
		log.Printf("💓 Received ping from client %s (socket ID: %s)", client.ID, sock.Id())
		ws.invokePingHooks(client)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up ping handler for client %s: %v", client.ID, err))

	// Handle disconnection
	err = sock.On("disconnect", func(data ...any) {
		log.Printf("🔌 Socket.IO connection closed for client %s (socket ID: %s)", client.ID, sock.Id())
		ws.invokeDisconnectionHooks(client)
		ws.removeClient(client.ID)
	})
	utils.AssertInvariant(
		err == nil,
		fmt.Sprintf("Failed to set up disconnection handler for client %s: %v", client.ID, err),
	)

	log.Printf("👂 Message listener setup complete for client %s", client.ID)
}

func (ws *SocketIOClientImpl) addClient(client *clients.Client) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.clients = append(ws.clients, client)
	ws.clientsBySocketID[string(client.Socket.Id())] = client
	log.Printf("📊 Client %s added to active connections. Total clients: %d", client.ID, len(ws.clients))
}

func (ws *SocketIOClientImpl) removeClient(clientID string) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	for i, client := range ws.clients {
		if client.ID == clientID {
			// Remove from both slices and map
			delete(ws.clientsBySocketID, string(client.Socket.Id()))
			ws.clients = append(ws.clients[:i], ws.clients[i+1:]...)
			log.Printf("🔌 Socket.IO client %s disconnected. Remaining clients: %d", clientID, len(ws.clients))
			return
		}
	}
	log.Printf("⚠️ Attempted to remove client %s but not found in active connections", clientID)
}

func (ws *SocketIOClientImpl) GetClientIDs() []string {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	clientIDs := make([]string, len(ws.clients))
	for i, client := range ws.clients {
		clientIDs[i] = client.ID
	}
	log.Printf("📋 Retrieved %d active client IDs", len(clientIDs))
	return clientIDs
}

func (ws *SocketIOClientImpl) getClientByID(clientID string) *clients.Client {
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

func (ws *SocketIOClientImpl) GetClientByID(clientID string) any {
	return ws.getClientByID(clientID)
}

func (ws *SocketIOClientImpl) SendMessage(clientID string, msg any) error {
	log.Printf("📤 Attempting to send message to client %s", clientID)
	client := ws.getClientByID(clientID)
	if client == nil {
		log.Printf("❌ Cannot send message: client %s not found", clientID)
		return fmt.Errorf("client with ID %s not found", clientID)
	}

	// Send message via Socket.IO emit to specific client
	err := client.Socket.Emit("cc_message", msg)
	if err != nil {
		log.Printf("❌ Failed to send message to client %s: %v", clientID, err)
		return fmt.Errorf("failed to send message to client %s: %w", clientID, err)
	}

	log.Printf("✅ Message sent successfully to client %s", clientID)
	return nil
}

// Interface methods with generic signatures
func (ws *SocketIOClientImpl) RegisterMessageHandler(handler func(client any, msg any)) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	// Convert the generic handler to our internal type
	adaptedHandler := func(client *clients.Client, msg any) {
		handler(client, msg)
	}
	ws.messageHandlers = append(ws.messageHandlers, adaptedHandler)
	log.Printf("📝 Message handler registered. Total handlers: %d", len(ws.messageHandlers))
}

func (ws *SocketIOClientImpl) RegisterConnectionHook(hook func(client any) error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	// Convert the generic hook to our internal type
	adaptedHook := func(client *clients.Client) error {
		return hook(client)
	}
	ws.connectionHooks = append(ws.connectionHooks, adaptedHook)
	log.Printf("🔗 Connection hook registered. Total connection hooks: %d", len(ws.connectionHooks))
}

func (ws *SocketIOClientImpl) RegisterDisconnectionHook(hook func(client any) error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	// Convert the generic hook to our internal type
	adaptedHook := func(client *clients.Client) error {
		return hook(client)
	}
	ws.disconnectionHooks = append(ws.disconnectionHooks, adaptedHook)
	log.Printf("🔌 Disconnection hook registered. Total disconnection hooks: %d", len(ws.disconnectionHooks))
}

func (ws *SocketIOClientImpl) RegisterPingHook(hook func(client any) error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	// Convert the generic hook to our internal type
	adaptedHook := func(client *clients.Client) error {
		return hook(client)
	}
	ws.pingHooks = append(ws.pingHooks, adaptedHook)
	log.Printf("💓 Ping hook registered. Total ping hooks: %d", len(ws.pingHooks))
}

func (ws *SocketIOClientImpl) invokeMessageHandlers(client *clients.Client, msg any) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	log.Printf("🔄 Invoking %d message handlers for client %s", len(ws.messageHandlers), client.ID)
	for i, handler := range ws.messageHandlers {
		log.Printf("🎯 Executing handler %d for client %s", i+1, client.ID)
		handler(client, msg)
	}
	log.Printf("✅ All message handlers completed for client %s", client.ID)
}

func (ws *SocketIOClientImpl) invokeConnectionHooks(client *clients.Client) {
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

func (ws *SocketIOClientImpl) invokeDisconnectionHooks(client *clients.Client) {
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

func (ws *SocketIOClientImpl) invokePingHooks(client *clients.Client) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	log.Printf("💓 Invoking %d ping hooks for client %s", len(ws.pingHooks), client.ID)
	for i, hook := range ws.pingHooks {
		log.Printf("🎯 Executing ping hook %d for client %s", i+1, client.ID)
		if err := hook(client); err != nil {
			log.Printf("❌ Ping hook %d failed for client %s: %v", i+1, client.ID, err)
		}
	}
	log.Printf("✅ All ping hooks completed for client %s", client.ID)
}
