package services

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"

	"ccagent/core/log"
	"ccagent/models"
)

type MessageProcessor struct {
	conn      *websocket.Conn
	connMutex sync.RWMutex
}

func NewMessageProcessor(conn *websocket.Conn) *MessageProcessor {
	return &MessageProcessor{
		conn: conn,
	}
}

func (mp *MessageProcessor) SendMessage(msg models.UnknownMessage) (string, error) {
	log.Info("📋 Starting to send message")

	if msg.ID == "" {
		return "", fmt.Errorf("message ID cannot be empty")
	}

	mp.connMutex.RLock()
	conn := mp.conn
	mp.connMutex.RUnlock()

	if conn == nil {
		log.Info("⚠️ No WebSocket connection available for message %s", msg.ID)
		return "", fmt.Errorf("no WebSocket connection available")
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Info("❌ Failed to write message %s to WebSocket: %v", msg.ID, err)
		return "", err
	}

	log.Info("✅ Message %s sent successfully", msg.ID)
	log.Info("📋 Completed successfully - sent message %s", msg.ID)
	return msg.ID, nil
}

func (mp *MessageProcessor) ResetConnection(conn *websocket.Conn) {
	log.Info("📋 Starting to reset WebSocket connection")

	mp.connMutex.Lock()
	mp.conn = conn
	mp.connMutex.Unlock()

	log.Info("📋 Completed successfully - reset WebSocket connection")
}

func (mp *MessageProcessor) Stop() {
	log.Info("📋 Starting to stop message processor")
	// No-op since we don't have background processes anymore
	log.Info("📋 Completed successfully - stopped message processor")
}
