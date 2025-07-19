package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type string `json:"type"`
}

type WebSocketServer struct {
	connections map[*websocket.Conn]bool
	mutex       sync.RWMutex
	upgrader    websocket.Upgrader
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		connections: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
	}
}

func (ws *WebSocketServer) handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	ws.addConnection(conn)
	defer ws.removeConnection(conn)

	log.Printf("✅ Client connected from %s", conn.RemoteAddr())

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error: %v", err)
			}
			break
		}

		log.Printf("📨 Received %s from %s", msg.Type, conn.RemoteAddr())

		if msg.Type == "ping" {
			response := Message{Type: "pong"}
			if err := conn.WriteJSON(response); err != nil {
				log.Printf("❌ Failed to send pong to %s: %v", conn.RemoteAddr(), err)
				break
			}
			log.Printf("🏓 Sent pong to %s", conn.RemoteAddr())
		}
	}

	log.Printf("🔌 Client %s disconnected", conn.RemoteAddr())
}

func (ws *WebSocketServer) addConnection(conn *websocket.Conn) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.connections[conn] = true
}

func (ws *WebSocketServer) removeConnection(conn *websocket.Conn) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	delete(ws.connections, conn)
}

func (ws *WebSocketServer) sendPingToAllClients() {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	msg := Message{Type: "ping"}
	activeConnections := len(ws.connections)
	
	if activeConnections == 0 {
		return
	}

	log.Printf("🏓 Sending ping to %d client(s)", activeConnections)
	
	for conn := range ws.connections {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("❌ Failed to send ping to %s: %v", conn.RemoteAddr(), err)
		}
	}
}

func (ws *WebSocketServer) getConnectionCount() int {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	return len(ws.connections)
}

func main() {
	server := NewWebSocketServer()

	http.HandleFunc("/ws", server.handleConnection)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := `<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Server</title>
</head>
<body>
    <h1>WebSocket Server</h1>
    <p>WebSocket endpoint: <code>ws://localhost:8080/ws</code></p>
    <p>Active connections: <span id="count">0</span></p>
    <script>
        setInterval(() => {
            fetch('/status').then(r => r.json()).then(data => {
                document.getElementById('count').textContent = data.connections;
            });
        }, 1000);
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{
			"connections": server.getConnectionCount(),
		})
	})

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if server.getConnectionCount() > 0 {
				server.sendPingToAllClients()
			}
		}
	}()

	port := "8080"
	log.Printf("🚀 WebSocket server starting on port %s", port)
	log.Printf("📡 WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Printf("🌐 Web interface: http://localhost:%s", port)
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("❌ Server failed to start:", err)
	}
}