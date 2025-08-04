package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"ccbackend/clients"
	"ccbackend/models"

	"github.com/google/uuid"
)

type PendingMessage struct {
	ID        string
	ClientID  string
	Message   any
	Timestamp time.Time
	Retries   int
}

type MessageProcessor struct {
	wsClient        *clients.WebSocketClient
	pendingMessages map[string]*PendingMessage
	pendingMutex    sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	retryInterval   time.Duration
	maxRetries      int
	ackTimeout      time.Duration
}

func NewMessageProcessor(wsClient *clients.WebSocketClient) *MessageProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	processor := &MessageProcessor{
		wsClient:        wsClient,
		pendingMessages: make(map[string]*PendingMessage),
		ctx:             ctx,
		cancel:          cancel,
		retryInterval:   30 * time.Second,
		maxRetries:      5,
		ackTimeout:      30 * time.Second,
	}

	go processor.retryLoop()

	return processor
}

func (mp *MessageProcessor) SendMessage(clientID string, msg any) (string, error) {
	log.Printf("📋 Starting to send message to client %s", clientID)

	messageID := uuid.New().String()

	// Add ID to the message - handle both struct and map cases
	var finalMsg any
	if unknownMsg, ok := msg.(models.UnknownMessage); ok {
		unknownMsg.ID = messageID
		finalMsg = unknownMsg
	} else {
		msgMap, ok := msg.(map[string]any)
		if !ok {
			log.Printf("❌ Message is not a supported type (UnknownMessage or map), cannot add ID")
			return "", fmt.Errorf("message must be UnknownMessage or map to add ID")
		}
		msgMap["id"] = messageID
		finalMsg = msgMap
	}

	pendingMsg := &PendingMessage{
		ID:        messageID,
		ClientID:  clientID,
		Message:   finalMsg,
		Timestamp: time.Now(),
		Retries:   0,
	}

	mp.pendingMutex.Lock()
	mp.pendingMessages[messageID] = pendingMsg
	mp.pendingMutex.Unlock()

	if err := mp.sendMessage(pendingMsg); err != nil {
		log.Printf("❌ Failed to send message %s: %v", messageID, err)
	}

	log.Printf("📋 Completed successfully - queued message %s for sending", messageID)
	return messageID, nil
}

func (mp *MessageProcessor) SendMessageReliably(clientID string, msg any) (string, error) {
	log.Printf("📋 Starting to send reliable message to client %s", clientID)

	messageID := uuid.New().String()

	// Add ID to the message - handle both struct and map cases
	var finalMsg any
	if unknownMsg, ok := msg.(models.UnknownMessage); ok {
		unknownMsg.ID = messageID
		finalMsg = unknownMsg
	} else {
		msgMap, ok := msg.(map[string]any)
		if !ok {
			log.Printf("❌ Message is not a supported type (UnknownMessage or map), cannot add ID")
			return "", fmt.Errorf("message must be UnknownMessage or map to add ID")
		}
		msgMap["id"] = messageID
		finalMsg = msgMap
	}

	pendingMsg := &PendingMessage{
		ID:        messageID,
		ClientID:  clientID,
		Message:   finalMsg,
		Timestamp: time.Now(),
		Retries:   0,
	}

	mp.pendingMutex.Lock()
	mp.pendingMessages[messageID] = pendingMsg
	mp.pendingMutex.Unlock()

	if err := mp.sendMessage(pendingMsg); err != nil {
		log.Printf("❌ Failed to send reliable message %s: %v", messageID, err)
	}

	log.Printf("📋 Completed successfully - queued reliable message %s for sending", messageID)
	return messageID, nil
}

func (mp *MessageProcessor) sendMessage(pendingMsg *PendingMessage) error {
	log.Printf("📤 Sending message %s to client %s (attempt %d)", pendingMsg.ID, pendingMsg.ClientID, pendingMsg.Retries+1)

	if err := mp.wsClient.SendMessage(pendingMsg.ClientID, pendingMsg.Message); err != nil {
		log.Printf("❌ Failed to send message %s to client %s: %v", pendingMsg.ID, pendingMsg.ClientID, err)
		return err
	}

	mp.pendingMutex.Lock()
	if msg, exists := mp.pendingMessages[pendingMsg.ID]; exists {
		msg.Retries++
		msg.Timestamp = time.Now()
	}
	mp.pendingMutex.Unlock()

	log.Printf("✅ Message %s sent successfully to client %s", pendingMsg.ID, pendingMsg.ClientID)
	return nil
}

func (mp *MessageProcessor) HandleAcknowledgement(messageID string) {
	log.Printf("📋 Starting to handle acknowledgement for message: %s", messageID)

	mp.pendingMutex.Lock()
	defer mp.pendingMutex.Unlock()

	if _, exists := mp.pendingMessages[messageID]; exists {
		delete(mp.pendingMessages, messageID)
		log.Printf("✅ Message %s acknowledged and removed from pending", messageID)
	} else {
		log.Printf("⚠️ Received acknowledgement for unknown message: %s", messageID)
	}

	log.Printf("📋 Completed successfully - handled acknowledgement for message %s", messageID)
}

func (mp *MessageProcessor) retryLoop() {
	log.Printf("🔄 Starting retry loop for message processor")

	ticker := time.NewTicker(mp.retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mp.processRetries()
		case <-mp.ctx.Done():
			log.Printf("🛑 Retry loop stopping due to context cancellation")
			return
		}
	}
}

func (mp *MessageProcessor) processRetries() {
	log.Printf("🔍 Processing message retries")

	mp.pendingMutex.RLock()
	messagesToRetry := make([]*PendingMessage, 0)
	messagesToRemove := make([]string, 0)

	now := time.Now()
	for messageID, pendingMsg := range mp.pendingMessages {
		if now.Sub(pendingMsg.Timestamp) > mp.ackTimeout {
			if pendingMsg.Retries >= mp.maxRetries {
				log.Printf("❌ Message %s exceeded max retries (%d), removing", messageID, mp.maxRetries)
				messagesToRemove = append(messagesToRemove, messageID)
			} else {
				log.Printf("⏰ Message %s timed out, queueing for retry", messageID)
				messagesToRetry = append(messagesToRetry, pendingMsg)
			}
		}
	}
	mp.pendingMutex.RUnlock()

	if len(messagesToRemove) > 0 {
		mp.pendingMutex.Lock()
		for _, messageID := range messagesToRemove {
			delete(mp.pendingMessages, messageID)
		}
		mp.pendingMutex.Unlock()
	}

	for _, pendingMsg := range messagesToRetry {
		if err := mp.sendMessage(pendingMsg); err != nil {
			log.Printf("❌ Failed to retry message %s: %v", pendingMsg.ID, err)
		}
	}

	if len(messagesToRetry) > 0 || len(messagesToRemove) > 0 {
		log.Printf("🔄 Processed %d retries and removed %d failed messages", len(messagesToRetry), len(messagesToRemove))
	}
}

func (mp *MessageProcessor) CleanupClientMessages(clientID string) {
	log.Printf("📋 Starting to cleanup pending messages for disconnected client %s", clientID)

	mp.pendingMutex.Lock()
	defer mp.pendingMutex.Unlock()

	removedCount := 0
	for messageID, pendingMsg := range mp.pendingMessages {
		if pendingMsg.ClientID == clientID {
			delete(mp.pendingMessages, messageID)
			removedCount++
		}
	}

	if removedCount > 0 {
		log.Printf("🧹 Cleaned up %d pending messages for disconnected client %s", removedCount, clientID)
	}

	log.Printf("📋 Completed successfully - cleaned up messages for client %s", clientID)
}

func (mp *MessageProcessor) Stop() {
	if mp == nil {
		return
	}

	log.Printf("📋 Starting to stop message processor")

	mp.cancel()

	log.Printf("📋 Completed successfully - stopped message processor")
}
