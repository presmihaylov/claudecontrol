package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type Client struct {
	ID         string
	ClientConn *websocket.Conn
}

type MessageHandlerFunc func(client *Client, msg any)

type WebSocketServer struct {
	clients         []*Client
	mutex           sync.RWMutex
	upgrader        websocket.Upgrader
	messageHandlers []MessageHandlerFunc
}

func NewWebsocketServer() *WebSocketServer {
	return &WebSocketServer{
		clients: make([]*Client, 0),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
		messageHandlers: make([]MessageHandlerFunc, 0),
	}
}

func (ws *WebSocketServer) StartWebsocketServer() {
	log.Printf("🚀 Starting WebSocket server on /ws endpoint")
	http.HandleFunc("/ws", ws.handleWebSocketConnection)
	log.Printf("✅ WebSocket server registered on /ws")
}

func (ws *WebSocketServer) handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔗 New WebSocket connection attempt from %s", r.RemoteAddr)
	
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
		ID:         uuid.New().String(),
		ClientConn: conn,
	}
	ws.addClient(client)
	log.Printf("✅ WebSocket client connected with ID: %s from %s", client.ID, r.RemoteAddr)
	defer ws.removeClient(client.ID)

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

func (ws *WebSocketServer) addClient(client *Client) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.clients = append(ws.clients, client)
	log.Printf("📊 Client %s added to active connections. Total clients: %d", client.ID, len(ws.clients))
}

func (ws *WebSocketServer) removeClient(clientID string) {
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

func (ws *WebSocketServer) GetClientIDs() []string {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	clientIDs := make([]string, len(ws.clients))
	for i, client := range ws.clients {
		clientIDs[i] = client.ID
	}
	log.Printf("📋 Retrieved %d active client IDs", len(clientIDs))
	return clientIDs
}

func (ws *WebSocketServer) getClientByID(clientID string) *Client {
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

func (ws *WebSocketServer) SendMessage(clientID string, msg any) error {
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

func (ws *WebSocketServer) registerMessageHandler(handler MessageHandlerFunc) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.messageHandlers = append(ws.messageHandlers, handler)
	log.Printf("📝 Message handler registered. Total handlers: %d", len(ws.messageHandlers))
}

func (ws *WebSocketServer) invokeMessageHandlers(client *Client, msg any) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	log.Printf("🔄 Invoking %d message handlers for client %s", len(ws.messageHandlers), client.ID)
	for i, handler := range ws.messageHandlers {
		log.Printf("🎯 Executing handler %d for client %s", i+1, client.ID)
		handler(client, msg)
	}
	log.Printf("✅ All message handlers completed for client %s", client.ID)
}
