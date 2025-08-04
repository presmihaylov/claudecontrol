package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gammazero/workerpool"

	"ccbackend/models"
)

// MessageSender defines the interface for sending messages to WebSocket clients
type MessageSender interface {
	SendMessage(clientID string, msg any) error
}

type PendingMessage struct {
	ID        string
	Message   any
	ClientID  string
	Timestamp time.Time
	Retries   int
}

type MessageProcessor struct {
	messageSender   MessageSender
	pendingMessages map[string]*PendingMessage
	pendingMutex    sync.RWMutex
	workerPool      *workerpool.WorkerPool
	ctx             context.Context
	cancel          context.CancelFunc
	retryInterval   time.Duration
	maxRetries      int
	ackTimeout      time.Duration
}

func NewMessageProcessor(messageSender MessageSender) *MessageProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	processor := &MessageProcessor{
		messageSender:   messageSender,
		pendingMessages: make(map[string]*PendingMessage),
		workerPool:      workerpool.New(1), // Sequential processing
		ctx:             ctx,
		cancel:          cancel,
		retryInterval:   30 * time.Second,
		maxRetries:      5,
		ackTimeout:      30 * time.Second,
	}

	go processor.retryLoop()

	return processor
}

func (mp *MessageProcessor) SendMessageReliably(clientID string, msg models.UnknownMessage) (string, error) {
	log.Printf("📋 Starting to send reliable message to client %s", clientID)

	pendingMsg := &PendingMessage{
		ID:        msg.ID,
		Message:   msg,
		ClientID:  clientID,
		Timestamp: time.Now(),
		Retries:   0,
	}

	mp.pendingMutex.Lock()
	mp.pendingMessages[msg.ID] = pendingMsg
	mp.pendingMutex.Unlock()

	// Submit to worker pool for sequential processing
	mp.workerPool.Submit(func() {
		if err := mp.sendMessage(pendingMsg); err != nil {
			log.Printf("❌ Failed to send reliable message %s: %v", msg.ID, err)
		}
	})

	log.Printf("📋 Completed successfully - queued reliable message %s for sending", msg.ID)
	return msg.ID, nil
}

func (mp *MessageProcessor) sendMessage(pendingMsg *PendingMessage) error {
	log.Printf("📤 Sending message %s to client %s (attempt %d)", pendingMsg.ID, pendingMsg.ClientID, pendingMsg.Retries+1)

	if err := mp.messageSender.SendMessage(pendingMsg.ClientID, pendingMsg.Message); err != nil {
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

func (mp *MessageProcessor) CleanupClientMessages(clientID string) {
	log.Printf("📋 Starting to cleanup messages for disconnected client %s", clientID)

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
		mp.workerPool.Submit(func() {
			if err := mp.sendMessage(pendingMsg); err != nil {
				log.Printf("❌ Failed to retry message %s: %v", pendingMsg.ID, err)
			}
		})
	}

	if len(messagesToRetry) > 0 || len(messagesToRemove) > 0 {
		log.Printf("🔄 Processed %d retries and removed %d failed messages", len(messagesToRetry), len(messagesToRemove))
	}
}

func (mp *MessageProcessor) Stop() {
	if mp == nil {
		return
	}

	log.Printf("📋 Starting to stop message processor")

	mp.cancel()
	mp.workerPool.StopWait()

	log.Printf("📋 Completed successfully - stopped message processor")
}
