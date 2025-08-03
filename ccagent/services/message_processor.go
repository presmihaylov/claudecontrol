package services

import (
	"context"
	"sync"
	"time"

	"ccagent/core/log"

	"github.com/gammazero/workerpool"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type PendingMessage struct {
	ID        string
	Message   any
	Timestamp time.Time
	Retries   int
}

type MessageProcessor struct {
	conn                *websocket.Conn
	pendingMessages     map[string]*PendingMessage
	pendingMutex        sync.RWMutex
	workerPool          *workerpool.WorkerPool
	ctx                 context.Context
	cancel              context.CancelFunc
	retryInterval       time.Duration
	maxRetries          int
	ackTimeout          time.Duration
	onAckReceived       func(messageID string)
}

func NewMessageProcessor(conn *websocket.Conn) *MessageProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	processor := &MessageProcessor{
		conn:            conn,
		pendingMessages: make(map[string]*PendingMessage),
		workerPool:      workerpool.New(1), // Sequential processing
		ctx:             ctx,
		cancel:          cancel,
		retryInterval:   5 * time.Second,
		maxRetries:      3,
		ackTimeout:      30 * time.Second,
	}
	
	// Start the retry goroutine
	go processor.retryLoop()
	
	return processor
}

func (mp *MessageProcessor) SendReliableMessage(messageType string, payload any) (string, error) {
	log.Info("📋 Starting to send reliable message of type: %s", messageType)
	
	messageID := uuid.New().String()
	
	reliableMsg := map[string]any{
		"id":      messageID,
		"type":    messageType,
		"payload": payload,
	}
	
	pendingMsg := &PendingMessage{
		ID:        messageID,
		Message:   reliableMsg,
		Timestamp: time.Now(),
		Retries:   0,
	}
	
	// Store in pending messages
	mp.pendingMutex.Lock()
	mp.pendingMessages[messageID] = pendingMsg
	mp.pendingMutex.Unlock()
	
	// Submit to worker pool for processing
	mp.workerPool.Submit(func() {
		if err := mp.sendMessage(pendingMsg); err != nil {
			log.Info("❌ Failed to send reliable message %s: %v", messageID, err)
		}
	})
	
	log.Info("📋 Completed successfully - queued reliable message %s for sending", messageID)
	return messageID, nil
}

func (mp *MessageProcessor) sendMessage(pendingMsg *PendingMessage) error {
	log.Info("📤 Sending message %s (attempt %d)", pendingMsg.ID, pendingMsg.Retries+1)
	
	if err := mp.conn.WriteJSON(pendingMsg.Message); err != nil {
		log.Info("❌ Failed to write message %s to WebSocket: %v", pendingMsg.ID, err)
		return err
	}
	
	// Update retry count
	mp.pendingMutex.Lock()
	if msg, exists := mp.pendingMessages[pendingMsg.ID]; exists {
		msg.Retries++
		msg.Timestamp = time.Now()
	}
	mp.pendingMutex.Unlock()
	
	log.Info("✅ Message %s sent successfully", pendingMsg.ID)
	return nil
}

func (mp *MessageProcessor) HandleAcknowledgement(messageID string) {
	log.Info("📋 Starting to handle acknowledgement for message: %s", messageID)
	
	mp.pendingMutex.Lock()
	defer mp.pendingMutex.Unlock()
	
	if _, exists := mp.pendingMessages[messageID]; exists {
		delete(mp.pendingMessages, messageID)
		log.Info("✅ Message %s acknowledged and removed from pending", messageID)
		
		// Call the acknowledgement callback if set
		if mp.onAckReceived != nil {
			mp.onAckReceived(messageID)
		}
	} else {
		log.Info("⚠️ Received acknowledgement for unknown message: %s", messageID)
	}
	
	log.Info("📋 Completed successfully - handled acknowledgement for message %s", messageID)
}

func (mp *MessageProcessor) retryLoop() {
	log.Info("🔄 Starting retry loop for message processor")
	
	ticker := time.NewTicker(mp.retryInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mp.processRetries()
		case <-mp.ctx.Done():
			log.Info("🛑 Retry loop stopping due to context cancellation")
			return
		}
	}
}

func (mp *MessageProcessor) processRetries() {
	log.Info("🔍 Processing message retries")
	
	mp.pendingMutex.RLock()
	messagesToRetry := make([]*PendingMessage, 0)
	messagesToRemove := make([]string, 0)
	
	now := time.Now()
	for messageID, pendingMsg := range mp.pendingMessages {
		// Check if message has timed out
		if now.Sub(pendingMsg.Timestamp) > mp.ackTimeout {
			if pendingMsg.Retries >= mp.maxRetries {
				log.Info("❌ Message %s exceeded max retries (%d), removing", messageID, mp.maxRetries)
				messagesToRemove = append(messagesToRemove, messageID)
			} else {
				log.Info("⏰ Message %s timed out, queueing for retry", messageID)
				messagesToRetry = append(messagesToRetry, pendingMsg)
			}
		}
	}
	mp.pendingMutex.RUnlock()
	
	// Remove messages that exceeded max retries
	if len(messagesToRemove) > 0 {
		mp.pendingMutex.Lock()
		for _, messageID := range messagesToRemove {
			delete(mp.pendingMessages, messageID)
		}
		mp.pendingMutex.Unlock()
	}
	
	// Retry messages that timed out
	for _, pendingMsg := range messagesToRetry {
		mp.workerPool.Submit(func() {
			if err := mp.sendMessage(pendingMsg); err != nil {
				log.Info("❌ Failed to retry message %s: %v", pendingMsg.ID, err)
			}
		})
	}
	
	if len(messagesToRetry) > 0 || len(messagesToRemove) > 0 {
		log.Info("🔄 Processed %d retries and removed %d failed messages", len(messagesToRetry), len(messagesToRemove))
	}
}

func (mp *MessageProcessor) GetPendingMessageCount() int {
	mp.pendingMutex.RLock()
	defer mp.pendingMutex.RUnlock()
	return len(mp.pendingMessages)
}

func (mp *MessageProcessor) SetAcknowledgementCallback(callback func(messageID string)) {
	mp.onAckReceived = callback
}

func (mp *MessageProcessor) Stop() {
	log.Info("📋 Starting to stop message processor")
	
	// Cancel the context to stop the retry loop
	mp.cancel()
	
	// Stop the worker pool and wait for completion
	mp.workerPool.StopWait()
	
	log.Info("📋 Completed successfully - stopped message processor")
}