package services

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/models"
)

// MessageProcessorConfig allows customizing MessageProcessor settings
type MessageProcessorConfig struct {
	RetryInterval time.Duration
	MaxRetries    int
	AckTimeout    time.Duration
}

// NewMessageProcessorWithConfig creates a MessageProcessor with custom configuration
func NewMessageProcessorWithConfig(messageSender MessageSender, config MessageProcessorConfig) *MessageProcessor {
	processor := NewMessageProcessor(messageSender)

	// Override default configuration
	processor.retryInterval = config.RetryInterval
	processor.maxRetries = config.MaxRetries
	processor.ackTimeout = config.AckTimeout

	return processor
}


// Test setup helper
func setupMessageProcessorTest(t *testing.T) (*MessageProcessor, *MockMessageSender, func()) {
	mockSender := NewMockMessageSender()
	processor := NewMessageProcessor(mockSender)

	cleanup := func() {
		processor.Stop()
	}

	return processor, mockSender, cleanup
}

func setupMessageProcessorTestWithConfig(t *testing.T, config MessageProcessorConfig) (*MessageProcessor, *MockMessageSender, func()) {
	mockSender := NewMockMessageSender()
	processor := NewMessageProcessorWithConfig(mockSender, config)

	cleanup := func() {
		processor.Stop()
	}

	return processor, mockSender, cleanup
}

// Test helper to create test messages
func createTestMessage(msgType string) models.UnknownMessage {
	return models.UnknownMessage{
		ID:   uuid.New().String(),
		Type: msgType,
		Payload: map[string]any{
			"test": "data",
		},
	}
}


// Reliable Message Delivery Tests
func TestMessageProcessor_ReliableDelivery(t *testing.T) {
	t.Run("successful message delivery", func(t *testing.T) {
		processor, mockSender, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		msg := createTestMessage("test")
		originalID := msg.ID

		messageID, err := processor.SendMessageReliably("client1", msg)

		require.NoError(t, err)
		assert.Equal(t, originalID, messageID)

		// Wait for message to be sent
		time.Sleep(50 * time.Millisecond)

		// Verify message was sent
		calls := mockSender.GetSendMessageCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "client1", calls[0].ClientID)

		// Verify message has the original ID
		sentMsg := calls[0].Message.(models.UnknownMessage)
		assert.Equal(t, originalID, sentMsg.ID)
		assert.Equal(t, "test", sentMsg.Type)
	})

	t.Run("message added to pending list", func(t *testing.T) {
		processor, mockSender, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		// Configure mock to simulate network delay
		mockSender.SendMessageDelay = 100 * time.Millisecond

		msg := createTestMessage("test")
		messageID, err := processor.SendMessageReliably("client1", msg)
		require.NoError(t, err)

		// Message should be in pending list initially
		processor.pendingMutex.RLock()
		_, exists := processor.pendingMessages[messageID]
		pendingCount := len(processor.pendingMessages)
		processor.pendingMutex.RUnlock()
		assert.True(t, exists)
		assert.Equal(t, 1, pendingCount)

		// Wait for send to complete
		time.Sleep(150 * time.Millisecond)

		// Verify message was sent
		calls := mockSender.GetSendMessageCalls()
		assert.Len(t, calls, 1)
	})

	t.Run("multiple messages to different clients", func(t *testing.T) {
		processor, mockSender, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		msg1 := createTestMessage("test1")
		msg2 := createTestMessage("test2")

		messageID1, err1 := processor.SendMessageReliably("client1", msg1)
		messageID2, err2 := processor.SendMessageReliably("client2", msg2)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, messageID1, messageID2)

		// Wait for messages to be sent
		time.Sleep(50 * time.Millisecond)

		calls := mockSender.GetSendMessageCalls()
		require.Len(t, calls, 2)

		// Verify both messages were sent to correct clients
		clientIDs := []string{calls[0].ClientID, calls[1].ClientID}
		assert.Contains(t, clientIDs, "client1")
		assert.Contains(t, clientIDs, "client2")
	})
}

// Acknowledgment Handling Tests
func TestMessageProcessor_Acknowledgments(t *testing.T) {
	t.Run("successful acknowledgment removes pending message", func(t *testing.T) {
		processor, _, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		msg := createTestMessage("test")
		messageID, err := processor.SendMessageReliably("client1", msg)
		require.NoError(t, err)

		// Wait for send to complete
		time.Sleep(50 * time.Millisecond)
		processor.pendingMutex.RLock()
		_, exists := processor.pendingMessages[messageID]
		processor.pendingMutex.RUnlock()
		assert.True(t, exists)

		// Send acknowledgment
		processor.HandleAcknowledgement(messageID)

		// Message should be removed from pending
		processor.pendingMutex.RLock()
		_, existsAfter := processor.pendingMessages[messageID]
		pendingCountAfter := len(processor.pendingMessages)
		processor.pendingMutex.RUnlock()
		assert.False(t, existsAfter)
		assert.Equal(t, 0, pendingCountAfter)
	})

	t.Run("acknowledgment for unknown message is handled gracefully", func(t *testing.T) {
		processor, _, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		// Should not panic or error
		processor.HandleAcknowledgement("unknown-message-id")

		// Verify no pending messages
		processor.pendingMutex.RLock()
		pendingCount := len(processor.pendingMessages)
		processor.pendingMutex.RUnlock()
		assert.Equal(t, 0, pendingCount)
	})

	t.Run("multiple acknowledgments for same message", func(t *testing.T) {
		processor, _, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		msg := createTestMessage("test")
		messageID, err := processor.SendMessageReliably("client1", msg)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		processor.pendingMutex.RLock()
		_, exists := processor.pendingMessages[messageID]
		processor.pendingMutex.RUnlock()
		assert.True(t, exists)

		// Send acknowledgment multiple times
		processor.HandleAcknowledgement(messageID)
		processor.HandleAcknowledgement(messageID)
		processor.HandleAcknowledgement(messageID)

		// Should not cause issues
		processor.pendingMutex.RLock()
		_, existsAfter := processor.pendingMessages[messageID]
		processor.pendingMutex.RUnlock()
		assert.False(t, existsAfter)
	})
}

// Retry Logic Tests
func TestMessageProcessor_RetryLogic(t *testing.T) {
	t.Run("message retried on timeout", func(t *testing.T) {
		config := MessageProcessorConfig{
			RetryInterval: 100 * time.Millisecond,
			MaxRetries:    2,
			AckTimeout:    50 * time.Millisecond,
		}
		processor, mockSender, cleanup := setupMessageProcessorTestWithConfig(t, config)
		defer cleanup()

		msg := createTestMessage("test")
		messageID, err := processor.SendMessageReliably("client1", msg)
		require.NoError(t, err)

		// Wait for initial send + retry cycle
		time.Sleep(250 * time.Millisecond)

		// Should have been attempted at least twice (initial + retry)
		calls := mockSender.GetSendMessageCalls()
		assert.GreaterOrEqual(t, len(calls), 2)

		// All calls should be for the same message
		for _, call := range calls {
			sentMsg := call.Message.(models.UnknownMessage)
			assert.Equal(t, messageID, sentMsg.ID)
			assert.Equal(t, "client1", call.ClientID)
		}

		// Send ACK to stop retries
		processor.HandleAcknowledgement(messageID)
		processor.pendingMutex.RLock()
		_, exists := processor.pendingMessages[messageID]
		processor.pendingMutex.RUnlock()
		assert.False(t, exists)
	})

	t.Run("message removed after max retries", func(t *testing.T) {
		config := MessageProcessorConfig{
			RetryInterval: 50 * time.Millisecond,
			MaxRetries:    2,
			AckTimeout:    25 * time.Millisecond,
		}
		processor, mockSender, cleanup := setupMessageProcessorTestWithConfig(t, config)
		defer cleanup()

		msg := createTestMessage("test")
		messageID, err := processor.SendMessageReliably("client1", msg)
		require.NoError(t, err)

		// Wait for max retries + cleanup
		time.Sleep(300 * time.Millisecond)

		// Message should be removed from pending after max retries
		processor.pendingMutex.RLock()
		_, exists := processor.pendingMessages[messageID]
		processor.pendingMutex.RUnlock()
		assert.False(t, exists)

		// Should have attempted 3 times (initial + 2 retries)
		calls := mockSender.GetSendMessageCalls()
		assert.GreaterOrEqual(t, len(calls), 3)
	})

	t.Run("retry stops on acknowledgment", func(t *testing.T) {
		config := MessageProcessorConfig{
			RetryInterval: 50 * time.Millisecond,
			MaxRetries:    5,
			AckTimeout:    25 * time.Millisecond,
		}
		processor, mockSender, cleanup := setupMessageProcessorTestWithConfig(t, config)
		defer cleanup()

		msg := createTestMessage("test")
		messageID, err := processor.SendMessageReliably("client1", msg)
		require.NoError(t, err)

		// Wait for first retry
		time.Sleep(100 * time.Millisecond)

		// Send acknowledgment
		processor.HandleAcknowledgement(messageID)

		// Record calls so far
		callsAfterAck := len(mockSender.GetSendMessageCalls())

		// Wait more time - no more retries should happen
		time.Sleep(150 * time.Millisecond)

		// Should not have additional calls after ACK
		finalCalls := len(mockSender.GetSendMessageCalls())
		assert.Equal(t, callsAfterAck, finalCalls)
	})
}

// Client Cleanup Tests
func TestMessageProcessor_ClientCleanup(t *testing.T) {
	t.Run("cleanup removes all client messages", func(t *testing.T) {
		processor, _, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		// Send messages for different clients
		msg1 := createTestMessage("test1")
		msg2 := createTestMessage("test2")
		msg3 := createTestMessage("test3")

		msg1ID, _ := processor.SendMessageReliably("client1", msg1)
		msg2ID, _ := processor.SendMessageReliably("client1", msg2)
		msg3ID, _ := processor.SendMessageReliably("client2", msg3)

		time.Sleep(50 * time.Millisecond)

		// All should be pending
		processor.pendingMutex.RLock()
		_, exists1 := processor.pendingMessages[msg1ID]
		_, exists2 := processor.pendingMessages[msg2ID]
		_, exists3 := processor.pendingMessages[msg3ID]
		pendingCount := len(processor.pendingMessages)
		processor.pendingMutex.RUnlock()
		assert.True(t, exists1)
		assert.True(t, exists2)
		assert.True(t, exists3)
		assert.Equal(t, 3, pendingCount)

		// Cleanup client1
		processor.CleanupClientMessages("client1")

		// Only client1 messages should be removed
		processor.pendingMutex.RLock()
		_, exists1After := processor.pendingMessages[msg1ID]
		_, exists2After := processor.pendingMessages[msg2ID]
		_, exists3After := processor.pendingMessages[msg3ID]
		pendingCountAfter := len(processor.pendingMessages)
		processor.pendingMutex.RUnlock()
		assert.False(t, exists1After)
		assert.False(t, exists2After)
		assert.True(t, exists3After)
		assert.Equal(t, 1, pendingCountAfter)
	})

	t.Run("cleanup with no messages for client", func(t *testing.T) {
		processor, _, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		msg := createTestMessage("test")
		msgID, _ := processor.SendMessageReliably("client1", msg)

		time.Sleep(50 * time.Millisecond)
		processor.pendingMutex.RLock()
		pendingCount := len(processor.pendingMessages)
		processor.pendingMutex.RUnlock()
		assert.Equal(t, 1, pendingCount)

		// Cleanup different client
		processor.CleanupClientMessages("client2")

		// Should not affect client1's message
		processor.pendingMutex.RLock()
		_, exists := processor.pendingMessages[msgID]
		pendingCountAfter := len(processor.pendingMessages)
		processor.pendingMutex.RUnlock()
		assert.True(t, exists)
		assert.Equal(t, 1, pendingCountAfter)
	})
}

// Concurrent Access Tests
func TestMessageProcessor_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent message sending", func(t *testing.T) {
		processor, mockSender, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		const numGoroutines = 10
		const messagesPerGoroutine = 5

		var wg sync.WaitGroup
		messageIDs := make(chan string, numGoroutines*messagesPerGoroutine)

		// Send messages concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(clientID string) {
				defer wg.Done()
				for j := 0; j < messagesPerGoroutine; j++ {
					msg := createTestMessage("concurrent")
					msgID, err := processor.SendMessageReliably(clientID, msg)
					require.NoError(t, err)
					messageIDs <- msgID
				}
			}(fmt.Sprintf("client%d", i))
		}

		wg.Wait()
		close(messageIDs)

		// Wait for all messages to be processed
		time.Sleep(200 * time.Millisecond)

		// Verify all messages were sent
		totalMessages := numGoroutines * messagesPerGoroutine
		calls := mockSender.GetSendMessageCalls()
		assert.Len(t, calls, totalMessages)

		// Verify all message IDs are unique
		seenIDs := make(map[string]bool)
		for msgID := range messageIDs {
			assert.False(t, seenIDs[msgID], "Duplicate message ID: %s", msgID)
			seenIDs[msgID] = true
		}
	})

	t.Run("concurrent acknowledgments", func(t *testing.T) {
		processor, _, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		// Send multiple messages
		messageIDs := make([]string, 20)
		for i := 0; i < 20; i++ {
			msg := createTestMessage("concurrent_ack")
			msgID, err := processor.SendMessageReliably(fmt.Sprintf("client%d", i), msg)
			require.NoError(t, err)
			messageIDs[i] = msgID
		}

		time.Sleep(50 * time.Millisecond)
		processor.pendingMutex.RLock()
		pendingCount := len(processor.pendingMessages)
		processor.pendingMutex.RUnlock()
		assert.Equal(t, 20, pendingCount)

		// Send acknowledgments concurrently
		var wg sync.WaitGroup
		for _, msgID := range messageIDs {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				processor.HandleAcknowledgement(id)
			}(msgID)
		}

		wg.Wait()

		// All messages should be acknowledged
		processor.pendingMutex.RLock()
		pendingCountAfter := len(processor.pendingMessages)
		processor.pendingMutex.RUnlock()
		assert.Equal(t, 0, pendingCountAfter)
	})
}

// Error Handling Tests
func TestMessageProcessor_ErrorHandling(t *testing.T) {
	t.Run("send message error handling", func(t *testing.T) {
		processor, mockSender, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		// Configure mock to return error
		mockSender.SendMessageError = errors.New("network error")

		msg := createTestMessage("error_test")
		messageID, err := processor.SendMessageReliably("client1", msg)

		// Should not return error immediately (queued for processing)
		require.NoError(t, err)
		processor.pendingMutex.RLock()
		_, exists := processor.pendingMessages[messageID]
		processor.pendingMutex.RUnlock()
		assert.True(t, exists)

		// Wait for send attempt
		time.Sleep(50 * time.Millisecond)

		// Should have attempted to send despite error
		calls := mockSender.GetSendMessageCalls()
		assert.Len(t, calls, 1)

		// Message should still be pending for retry
		processor.pendingMutex.RLock()
		_, existsAfter := processor.pendingMessages[messageID]
		processor.pendingMutex.RUnlock()
		assert.True(t, existsAfter)
	})

	t.Run("processor stop cleans up resources", func(t *testing.T) {
		mockSender := NewMockMessageSender()
		processor := NewMessageProcessor(mockSender)

		msg := createTestMessage("stop_test")
		messageID, err := processor.SendMessageReliably("client1", msg)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		processor.pendingMutex.RLock()
		_, exists := processor.pendingMessages[messageID]
		processor.pendingMutex.RUnlock()
		assert.True(t, exists)

		// Stop processor
		processor.Stop()

		// Should not panic or hang
		// Note: We can't easily test that goroutines are cleaned up,
		// but Stop() should complete without blocking
	})
}
