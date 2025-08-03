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

type ReliableMessageProcessor struct {
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

func NewReliableMessageProcessor(conn *websocket.Conn) *ReliableMessageProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	processor := &ReliableMessageProcessor{
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

func (rmp *ReliableMessageProcessor) SendReliableMessage(messageType string, payload any) (string, error) {
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
	rmp.pendingMutex.Lock()
	rmp.pendingMessages[messageID] = pendingMsg
	rmp.pendingMutex.Unlock()
	
	// Submit to worker pool for processing
	rmp.workerPool.Submit(func() {
		if err := rmp.sendMessage(pendingMsg); err != nil {
			log.Info("❌ Failed to send reliable message %s: %v", messageID, err)
		}
	})
	
	log.Info("📋 Completed successfully - queued reliable message %s for sending", messageID)
	return messageID, nil
}

func (rmp *ReliableMessageProcessor) sendMessage(pendingMsg *PendingMessage) error {
	log.Info("📤 Sending message %s (attempt %d)", pendingMsg.ID, pendingMsg.Retries+1)
	
	if err := rmp.conn.WriteJSON(pendingMsg.Message); err != nil {
		log.Info("❌ Failed to write message %s to WebSocket: %v", pendingMsg.ID, err)
		return err
	}
	
	// Update retry count
	rmp.pendingMutex.Lock()
	if msg, exists := rmp.pendingMessages[pendingMsg.ID]; exists {
		msg.Retries++
		msg.Timestamp = time.Now()
	}
	rmp.pendingMutex.Unlock()
	
	log.Info("✅ Message %s sent successfully", pendingMsg.ID)
	return nil
}

func (rmp *ReliableMessageProcessor) HandleAcknowledgement(messageID string) {
	log.Info("📋 Starting to handle acknowledgement for message: %s", messageID)
	
	rmp.pendingMutex.Lock()
	defer rmp.pendingMutex.Unlock()
	
	if _, exists := rmp.pendingMessages[messageID]; exists {
		delete(rmp.pendingMessages, messageID)
		log.Info("✅ Message %s acknowledged and removed from pending", messageID)
		
		// Call the acknowledgement callback if set
		if rmp.onAckReceived != nil {
			rmp.onAckReceived(messageID)
		}
	} else {
		log.Info("⚠️ Received acknowledgement for unknown message: %s", messageID)
	}
	
	log.Info("📋 Completed successfully - handled acknowledgement for message %s", messageID)
}

func (rmp *ReliableMessageProcessor) retryLoop() {
	log.Info("🔄 Starting retry loop for reliable message processor")
	
	ticker := time.NewTicker(rmp.retryInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rmp.processRetries()
		case <-rmp.ctx.Done():
			log.Info("🛑 Retry loop stopping due to context cancellation")
			return
		}
	}
}

func (rmp *ReliableMessageProcessor) processRetries() {
	log.Info("🔍 Processing message retries")
	
	rmp.pendingMutex.RLock()
	messagesToRetry := make([]*PendingMessage, 0)
	messagesToRemove := make([]string, 0)
	
	now := time.Now()
	for messageID, pendingMsg := range rmp.pendingMessages {
		// Check if message has timed out
		if now.Sub(pendingMsg.Timestamp) > rmp.ackTimeout {
			if pendingMsg.Retries >= rmp.maxRetries {
				log.Info("❌ Message %s exceeded max retries (%d), removing", messageID, rmp.maxRetries)
				messagesToRemove = append(messagesToRemove, messageID)
			} else {
				log.Info("⏰ Message %s timed out, queueing for retry", messageID)
				messagesToRetry = append(messagesToRetry, pendingMsg)
			}
		}
	}
	rmp.pendingMutex.RUnlock()
	
	// Remove messages that exceeded max retries
	if len(messagesToRemove) > 0 {
		rmp.pendingMutex.Lock()
		for _, messageID := range messagesToRemove {
			delete(rmp.pendingMessages, messageID)
		}
		rmp.pendingMutex.Unlock()
	}
	
	// Retry messages that timed out
	for _, pendingMsg := range messagesToRetry {
		rmp.workerPool.Submit(func() {
			if err := rmp.sendMessage(pendingMsg); err != nil {
				log.Info("❌ Failed to retry message %s: %v", pendingMsg.ID, err)
			}
		})
	}
	
	if len(messagesToRetry) > 0 || len(messagesToRemove) > 0 {
		log.Info("🔄 Processed %d retries and removed %d failed messages", len(messagesToRetry), len(messagesToRemove))
	}
}

func (rmp *ReliableMessageProcessor) GetPendingMessageCount() int {
	rmp.pendingMutex.RLock()
	defer rmp.pendingMutex.RUnlock()
	return len(rmp.pendingMessages)
}

func (rmp *ReliableMessageProcessor) SetAcknowledgementCallback(callback func(messageID string)) {
	rmp.onAckReceived = callback
}

func (rmp *ReliableMessageProcessor) Stop() {
	log.Info("📋 Starting to stop reliable message processor")
	
	// Cancel the context to stop the retry loop
	rmp.cancel()
	
	// Stop the worker pool and wait for completion
	rmp.workerPool.StopWait()
	
	log.Info("📋 Completed successfully - stopped reliable message processor")
}