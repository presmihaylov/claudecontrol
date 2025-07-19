package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type string `json:"type"`
}

func main() {
	serverURL := "ws://localhost:8080/ws"
	if len(os.Args) > 1 {
		serverURL = os.Args[1]
	}

	log.Printf("🔌 Connecting to WebSocket server at %s", serverURL)

	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatal("❌ Failed to connect to WebSocket server:", err)
	}
	defer conn.Close()

	log.Printf("✅ Connected to WebSocket server")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("❌ Read error: %v", err)
				return
			}
			log.Printf("📨 Received: %s", msg.Type)

			if msg.Type == "ping" {
				response := Message{Type: "pong"}
				if err := conn.WriteJSON(response); err != nil {
					log.Printf("❌ Failed to send pong: %v", err)
					return
				}
				log.Printf("🏓 Sent pong response")
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			ping := Message{Type: "ping"}
			if err := conn.WriteJSON(ping); err != nil {
				log.Printf("❌ Failed to send ping: %v", err)
				return
			}
			log.Printf("🏓 Sent ping to server")
		case <-interrupt:
			log.Println("🔌 Interrupt received, closing connection...")

			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("❌ Failed to send close message: %v", err)
				return
			}

			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

