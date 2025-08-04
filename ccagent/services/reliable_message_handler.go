package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"ccagent/core/log"
	"ccagent/models"
)

type ProcessedMessage struct {
	MessageID   string
	ProcessedAt time.Time
}

type ReliableMessageHandler struct {
	processedMessages map[string]*ProcessedMessage
	mutex             sync.RWMutex
	cleanupInterval   time.Duration
	messageRetention  time.Duration
	messageProcessor  *MessageProcessor
}

func NewReliableMessageHandler(messageProcessor *MessageProcessor) *ReliableMessageHandler {
	handler := &ReliableMessageHandler{
		processedMessages: make(map[string]*ProcessedMessage),
		cleanupInterval:   5 * time.Minute,
		messageRetention:  30 * time.Minute,
		messageProcessor:  messageProcessor,
	}

	// Start cleanup goroutine
	go handler.cleanupLoop()

	return handler
}

func (rmh *ReliableMessageHandler) ProcessReliableMessage(rawMsg any) (bool, error) {
	log.Info("📋 Starting to process reliable message")

	// Try to extract message ID from the raw message
	msgMap, ok := rawMsg.(map[string]any)
	if !ok {
		log.Info("⚠️ Message is not a map, processing as regular message")
		return false, nil // Not a reliable message, process normally
	}

	messageID, hasID := msgMap["id"].(string)
	if !hasID || messageID == "" {
		log.Info("⚠️ Message has no ID, processing as regular message")
		return false, nil // Not a reliable message, process normally
	}

	// Check if this message was already processed
	rmh.mutex.RLock()
	processedMsg, alreadyProcessed := rmh.processedMessages[messageID]
	rmh.mutex.RUnlock()

	if alreadyProcessed {
		log.Info("🔄 Message %s already processed at %v, sending ack",
			messageID, processedMsg.ProcessedAt)

		// Send acknowledgement for already processed message
		if err := rmh.sendAcknowledgement(messageID); err != nil {
			log.Info("❌ Failed to send acknowledgement for duplicate message %s: %v", messageID, err)
			return true, fmt.Errorf("failed to send acknowledgement for duplicate message: %w", err)
		}

		log.Info("📋 Completed successfully - handled duplicate message %s", messageID)
		return true, nil // Message was handled (deduplicated)
	}

	log.Info("📋 Completed successfully - message %s not yet processed, will handle normally", messageID)
	return false, nil // Message should be processed normally by other handlers
}

func (rmh *ReliableMessageHandler) MarkMessageProcessed(rawMsg any) error {
	log.Info("📋 Starting to mark message as processed")

	// Try to extract message ID from the raw message
	msgMap, ok := rawMsg.(map[string]any)
	if !ok {
		log.Info("⚠️ Message is not a map, skipping processing marker")
		return nil // Not a reliable message, nothing to mark
	}

	messageID, hasID := msgMap["id"].(string)
	if !hasID || messageID == "" {
		log.Info("⚠️ Message has no ID, skipping processing marker")
		return nil // Not a reliable message, nothing to mark
	}

	// Mark message as processed
	rmh.mutex.Lock()
	rmh.processedMessages[messageID] = &ProcessedMessage{
		MessageID:   messageID,
		ProcessedAt: time.Now(),
	}
	rmh.mutex.Unlock()

	log.Info("✅ Message %s marked as processed", messageID)

	// Send acknowledgement
	if err := rmh.sendAcknowledgement(messageID); err != nil {
		log.Info("❌ Failed to send acknowledgement for message %s: %v", messageID, err)
		return fmt.Errorf("failed to send acknowledgement: %w", err)
	}

	log.Info("📋 Completed successfully - marked message %s as processed and sent ack", messageID)
	return nil
}

func (rmh *ReliableMessageHandler) sendAcknowledgement(messageID string) error {
	log.Info("📤 Sending acknowledgement for message %s", messageID)

	ackMsg := models.UnknownMessage{
		ID:   uuid.New().String(),
		Type: models.MessageTypeAcknowledgement,
		Payload: models.AcknowledgementPayload{
			MessageID: messageID,
		},
	}

	// Use reliable delivery for ACK messages to prevent infinite retry loops
	if _, err := rmh.messageProcessor.SendMessageReliably(ackMsg); err != nil {
		log.Info("❌ Failed to send reliable acknowledgement: %v", err)
		return err
	}

	log.Info("✅ Acknowledgement sent reliably for message %s", messageID)
	return nil
}

func (rmh *ReliableMessageHandler) cleanupLoop() {
	log.Info("🧹 Starting cleanup loop for processed messages")

	ticker := time.NewTicker(rmh.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rmh.cleanupOldMessages()
	}
}

func (rmh *ReliableMessageHandler) cleanupOldMessages() {
	log.Info("🧹 Starting cleanup of old processed messages")

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
		log.Info("🧹 Cleaned up %d old processed messages. Remaining: %d",
			removedCount, len(rmh.processedMessages))
	}
}
