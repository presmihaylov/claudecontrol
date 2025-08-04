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

// CheckAndAcknowledgeMessage checks if message was already processed and sends acknowledgement
// Returns true if message was already processed (should skip processing)
func (rmh *ReliableMessageHandler) CheckAndAcknowledgeMessage(msg models.UnknownMessage) bool {
	if msg.ID == "" {
		// No ID means not a reliable message, process normally
		return false
	}

	// Check if already processed
	rmh.mutex.RLock()
	processedMsg, alreadyProcessed := rmh.processedMessages[msg.ID]
	rmh.mutex.RUnlock()

	if alreadyProcessed {
		log.Info("🔄 Message %s already processed at %v, sending duplicate ack", msg.ID, processedMsg.ProcessedAt)
		rmh.sendAcknowledgement(msg.ID)
		return true // Skip processing
	}

	// New message - send acknowledgement immediately
	log.Info("📨 New message %s, sending ack", msg.ID)
	rmh.sendAcknowledgement(msg.ID)

	// Mark as processed
	rmh.mutex.Lock()
	rmh.processedMessages[msg.ID] = &ProcessedMessage{
		MessageID:   msg.ID,
		ProcessedAt: time.Now(),
	}
	rmh.mutex.Unlock()

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

func (rmh *ReliableMessageHandler) cleanupLoop() {
	log.Info("🧹 Starting cleanup loop for processed messages")

	ticker := time.NewTicker(rmh.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rmh.cleanupOldMessages()
	}
}

func (rmh *ReliableMessageHandler) cleanupOldMessages() {
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
