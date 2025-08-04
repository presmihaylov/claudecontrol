package services

import (
	"log"

	"ccbackend/models"
)

// MessageSender defines the interface for sending messages to WebSocket clients
type MessageSender interface {
	SendMessage(clientID string, msg any) error
}

type MessageProcessor struct {
	messageSender MessageSender
}

func NewMessageProcessor(messageSender MessageSender) *MessageProcessor {
	return &MessageProcessor{
		messageSender: messageSender,
	}
}

func (mp *MessageProcessor) SendMessage(clientID string, msg models.UnknownMessage) (string, error) {
	log.Printf("📋 Starting to send message to client %s", clientID)

	if err := mp.messageSender.SendMessage(clientID, msg); err != nil {
		log.Printf("❌ Failed to send message to client %s: %v", clientID, err)
		return "", err
	}

	log.Printf("✅ Message sent successfully to client %s", clientID)
	log.Printf("📋 Completed successfully - sent message to client %s", clientID)
	return msg.ID, nil
}

func (mp *MessageProcessor) CleanupClientMessages(clientID string) {
	log.Printf("📋 Starting to cleanup messages for disconnected client %s", clientID)
	// No-op since we don't track pending messages anymore
	log.Printf("📋 Completed successfully - cleaned up messages for client %s", clientID)
}

func (mp *MessageProcessor) Stop() {
	log.Printf("📋 Starting to stop message processor")
	// No-op since we don't have background processes anymore
	log.Printf("📋 Completed successfully - stopped message processor")
}
