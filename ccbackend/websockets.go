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

type WebSocketManager struct {
	connections map[*websocket.Conn]bool
	mutex       sync.RWMutex
	upgrader    websocket.Upgrader
}

var wsManager = &WebSocketManager{
	connections: make(map[*websocket.Conn]bool),
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

	wsm.addConnection(conn)
	defer wsm.removeConnection(conn)

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

func (wsm *WebSocketManager) addConnection(conn *websocket.Conn) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	wsm.connections[conn] = true
}

func (wsm *WebSocketManager) removeConnection(conn *websocket.Conn) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	delete(wsm.connections, conn)
}

func sendPingToClient(conn *websocket.Conn) error {
	msg := Message{Type: "ping"}
	return conn.WriteJSON(msg)
}

func sendPingToAllClients() {
	wsManager.mutex.RLock()
	defer wsManager.mutex.RUnlock()

	msg := Message{Type: "ping"}
	for conn := range wsManager.connections {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("❌ Failed to send ping to client: %v", err)
		}
	}
}

