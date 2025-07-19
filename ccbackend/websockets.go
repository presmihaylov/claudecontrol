package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type string `json:"type"`
}

type Client struct {
	ClientConn *websocket.Conn
}

type WebSocketManager struct {
	clients  []Client
	mutex    sync.RWMutex
	upgrader websocket.Upgrader
}

var wsManager = &WebSocketManager{
	clients: make([]Client, 0),
	upgrader: websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	},
}

func setupWebSocketEndpoint() {
	http.HandleFunc("/ws", wsManager.handleWebSocketConnection)
}

func (wsm *WebSocketManager) handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := wsm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	client := Client{ClientConn: conn}
	wsm.addClient(client)
	defer wsm.removeClient(conn)

	log.Printf("✅ WebSocket client connected")

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error: %v", err)
			}
			break
		}

		if msg.Type == "ping" {
			response := Message{Type: "pong"}
			if err := conn.WriteJSON(response); err != nil {
				log.Printf("❌ Failed to send pong: %v", err)
				break
			}
			log.Printf("🏓 Received ping, sent pong")
		}
	}

	log.Printf("🔌 WebSocket client disconnected")
}

func (wsm *WebSocketManager) addClient(client Client) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	wsm.clients = append(wsm.clients, client)
}

func (wsm *WebSocketManager) removeClient(conn *websocket.Conn) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	for i, client := range wsm.clients {
		if client.ClientConn == conn {
			wsm.clients = append(wsm.clients[:i], wsm.clients[i+1:]...)
			break
		}
	}
}

func sendPingToClient(conn *websocket.Conn) error {
	msg := Message{Type: "ping"}
	return conn.WriteJSON(msg)
}

func sendPingToAllClients() {
	wsManager.mutex.RLock()
	defer wsManager.mutex.RUnlock()

	msg := Message{Type: "ping"}
	for _, client := range wsManager.clients {
		if err := client.ClientConn.WriteJSON(msg); err != nil {
			log.Printf("❌ Failed to send ping to client: %v", err)
		}
	}
}

