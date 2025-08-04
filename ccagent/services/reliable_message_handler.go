package services

import (
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
	messageRetention  time.Duration
	messageProcessor  *MessageProcessor
}

func NewReliableMessageHandler(messageProcessor *MessageProcessor) *ReliableMessageHandler {
	return &ReliableMessageHandler{
		processedMessages: make(map[string]*ProcessedMessage),
		messageRetention:  30 * time.Minute,
		messageProcessor:  messageProcessor,
	}
}

// CheckAndAcknowledgeMessage checks if message was already processed and sends acknowledgement
// Returns true if message was already processed (should skip processing)
func (rmh *ReliableMessageHandler) CheckAndAcknowledgeMessage(msg models.UnknownMessage) bool {
	if msg.ID == "" {
		// No ID means not a reliable message, process normally
		return false
	}

	rmh.mutex.Lock()
	defer rmh.mutex.Unlock()

	// Clean up old messages while we have the lock
	now := time.Now()
	removedCount := 0
	for messageID, processedMsg := range rmh.processedMessages {
		if now.Sub(processedMsg.ProcessedAt) > rmh.messageRetention {
			delete(rmh.processedMessages, messageID)
			removedCount++
		}
	}
	if removedCount > 0 {
		log.Info("🧹 Cleaned up %d old processed messages. Remaining: %d", removedCount, len(rmh.processedMessages))
	}

	// Check if already processed
	processedMsg, alreadyProcessed := rmh.processedMessages[msg.ID]
	if alreadyProcessed {
		log.Info("🔄 Message %s already processed at %v, sending duplicate ack", msg.ID, processedMsg.ProcessedAt)
		rmh.sendAcknowledgement(msg.ID)
		return true // Skip processing
	}

	// New message - send acknowledgement immediately
	log.Info("📨 New message %s, sending ack", msg.ID)
	rmh.sendAcknowledgement(msg.ID)

	// Mark as processed
	rmh.processedMessages[msg.ID] = &ProcessedMessage{
		MessageID:   msg.ID,
		ProcessedAt: now,
	}

	return false // Continue with processing
}

func (rmh *ReliableMessageHandler) sendAcknowledgement(messageID string) {
	ackMsg := models.UnknownMessage{
		ID:   uuid.New().String(),
		Type: models.MessageTypeAcknowledgement,
		Payload: models.AcknowledgementPayload{
			MessageID: messageID,
		},
	}

	// Send acknowledgement (don't use reliable delivery to avoid loops)
	if _, err := rmh.messageProcessor.SendMessage(ackMsg); err != nil {
		log.Info("❌ Failed to send acknowledgement for message %s: %v", messageID, err)
	} else {
		log.Info("✅ Sent ack for message %s", messageID)
	}
}
