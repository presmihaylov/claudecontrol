package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"ccbackend/clients"
	"ccbackend/models"
	"github.com/google/uuid"
)

type ProcessedMessage struct {
	MessageID   string
	ProcessedAt time.Time
	ClientID    string
}

type ReliableMessageHandler struct {
	processedMessages map[string]*ProcessedMessage
	mutex             sync.RWMutex
	cleanupInterval   time.Duration
	messageRetention  time.Duration
	wsClient          *clients.WebSocketClient
}

func NewReliableMessageHandler(wsClient *clients.WebSocketClient) *ReliableMessageHandler {
	handler := &ReliableMessageHandler{
		processedMessages: make(map[string]*ProcessedMessage),
		cleanupInterval:   5 * time.Minute,
		messageRetention:  30 * time.Minute,
		wsClient:          wsClient,
	}

	// Start cleanup goroutine
	go handler.cleanupLoop()

	return handler
}

func (rmh *ReliableMessageHandler) ProcessReliableMessage(client *clients.Client, rawMsg any) (bool, error) {
	log.Printf("📋 Starting to process reliable message from client %s", client.ID)

	// Try to extract message ID from the raw message
	msgMap, ok := rawMsg.(map[string]any)
	if !ok {
		log.Printf("⚠️ Message from client %s is not a map, processing as regular message", client.ID)
		return false, nil // Not a reliable message, process normally
	}

	messageID, hasID := msgMap["id"].(string)
	if !hasID || messageID == "" {
		log.Printf("⚠️ Message from client %s has no ID, processing as regular message", client.ID)
		return false, nil // Not a reliable message, process normally
	}

	// Check if this message was already processed
	rmh.mutex.RLock()
	processedMsg, alreadyProcessed := rmh.processedMessages[messageID]
	rmh.mutex.RUnlock()

	if alreadyProcessed {
		log.Printf("🔄 Message %s from client %s already processed at %v, sending ack",
			messageID, client.ID, processedMsg.ProcessedAt)

		// Send acknowledgement for already processed message
		if err := rmh.sendAcknowledgement(client.ID, messageID); err != nil {
			log.Printf("❌ Failed to send acknowledgement for duplicate message %s: %v", messageID, err)
			return true, fmt.Errorf("failed to send acknowledgement for duplicate message: %w", err)
		}

		log.Printf("📋 Completed successfully - handled duplicate message %s", messageID)
		return true, nil // Message was handled (deduplicated)
	}

	log.Printf("📋 Completed successfully - message %s not yet processed, will handle normally", messageID)
	return false, nil // Message should be processed normally by other handlers
}

func (rmh *ReliableMessageHandler) MarkMessageProcessed(client *clients.Client, rawMsg any) error {
	log.Printf("📋 Starting to mark message as processed from client %s", client.ID)

	// Try to extract message ID from the raw message
	msgMap, ok := rawMsg.(map[string]any)
	if !ok {
		log.Printf("⚠️ Message from client %s is not a map, skipping processing marker", client.ID)
		return nil // Not a reliable message, nothing to mark
	}

	messageID, hasID := msgMap["id"].(string)
	if !hasID || messageID == "" {
		log.Printf("⚠️ Message from client %s has no ID, skipping processing marker", client.ID)
		return nil // Not a reliable message, nothing to mark
	}

	// Mark message as processed
	rmh.mutex.Lock()
	rmh.processedMessages[messageID] = &ProcessedMessage{
		MessageID:   messageID,
		ProcessedAt: time.Now(),
		ClientID:    client.ID,
	}
	rmh.mutex.Unlock()

	log.Printf("✅ Message %s from client %s marked as processed", messageID, client.ID)

	// Send acknowledgement
	if err := rmh.sendAcknowledgement(client.ID, messageID); err != nil {
		log.Printf("❌ Failed to send acknowledgement for message %s: %v", messageID, err)
		return fmt.Errorf("failed to send acknowledgement: %w", err)
	}

	log.Printf("📋 Completed successfully - marked message %s as processed and sent ack", messageID)
	return nil
}

func (rmh *ReliableMessageHandler) sendAcknowledgement(clientID, messageID string) error {
	log.Printf("📤 Sending acknowledgement for message %s to client %s", messageID, clientID)

	ackMsg := models.UnknownMessage{
		ID:   uuid.New().String(),
		Type: models.MessageTypeAcknowledgement,
		Payload: models.AcknowledgementPayload{
			MessageID: messageID,
		},
	}

	if err := rmh.wsClient.SendMessage(clientID, ackMsg); err != nil {
		log.Printf("❌ Failed to send acknowledgement to client %s: %v", clientID, err)
		return err
	}

	log.Printf("✅ Acknowledgement sent successfully for message %s", messageID)
	return nil
}

func (rmh *ReliableMessageHandler) cleanupLoop() {
	log.Printf("🧹 Starting cleanup loop for processed messages")

	ticker := time.NewTicker(rmh.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rmh.cleanupOldMessages()
	}
}

func (rmh *ReliableMessageHandler) cleanupOldMessages() {
	log.Printf("🧹 Starting cleanup of old processed messages")

	rmh.mutex.Lock()
	defer rmh.mutex.Unlock()

	now := time.Now()
	removedCount := 0

	for messageID, processedMsg := range rmh.processedMessages {
		if now.Sub(processedMsg.ProcessedAt) > rmh.messageRetention {
			delete(rmh.processedMessages, messageID)
			removedCount++
		}
	}

	if removedCount > 0 {
		log.Printf("🧹 Cleaned up %d old processed messages. Remaining: %d",
			removedCount, len(rmh.processedMessages))
	}
}
