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

// Helper functions for safe access to private fields that need mutex protection
func hasPendingMessage(processor *MessageProcessor, messageID string) bool {
	processor.pendingMutex.RLock()
	defer processor.pendingMutex.RUnlock()
	_, exists := processor.pendingMessages[messageID]
	return exists
}

func getPendingMessageCount(processor *MessageProcessor) int {
	processor.pendingMutex.RLock()
	defer processor.pendingMutex.RUnlock()
	return len(processor.pendingMessages)
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
		assert.True(t, hasPendingMessage(processor, messageID))
		assert.Equal(t, 1, getPendingMessageCount(processor))

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
		assert.True(t, hasPendingMessage(processor, messageID))

		// Send acknowledgment
		processor.HandleAcknowledgement(messageID)

		// Message should be removed from pending
		assert.False(t, hasPendingMessage(processor, messageID))
		assert.Equal(t, 0, getPendingMessageCount(processor))
	})

	t.Run("acknowledgment for unknown message is handled gracefully", func(t *testing.T) {
		processor, _, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		// Should not panic or error
		processor.HandleAcknowledgement("unknown-message-id")

		// Verify no pending messages
		assert.Equal(t, 0, getPendingMessageCount(processor))
	})

	t.Run("multiple acknowledgments for same message", func(t *testing.T) {
		processor, _, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		msg := createTestMessage("test")
		messageID, err := processor.SendMessageReliably("client1", msg)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		assert.True(t, hasPendingMessage(processor, messageID))

		// Send acknowledgment multiple times
		processor.HandleAcknowledgement(messageID)
		processor.HandleAcknowledgement(messageID)
		processor.HandleAcknowledgement(messageID)

		// Should not cause issues
		assert.False(t, hasPendingMessage(processor, messageID))
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
		assert.False(t, hasPendingMessage(processor, messageID))
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
		assert.False(t, hasPendingMessage(processor, messageID))

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
		assert.True(t, hasPendingMessage(processor, msg1ID))
		assert.True(t, hasPendingMessage(processor, msg2ID))
		assert.True(t, hasPendingMessage(processor, msg3ID))
		assert.Equal(t, 3, getPendingMessageCount(processor))

		// Cleanup client1
		processor.CleanupClientMessages("client1")

		// Only client1 messages should be removed
		assert.False(t, hasPendingMessage(processor, msg1ID))
		assert.False(t, hasPendingMessage(processor, msg2ID))
		assert.True(t, hasPendingMessage(processor, msg3ID))
		assert.Equal(t, 1, getPendingMessageCount(processor))
	})

	t.Run("cleanup with no messages for client", func(t *testing.T) {
		processor, _, cleanup := setupMessageProcessorTest(t)
		defer cleanup()

		msg := createTestMessage("test")
		msgID, _ := processor.SendMessageReliably("client1", msg)

		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 1, getPendingMessageCount(processor))

		// Cleanup different client
		processor.CleanupClientMessages("client2")

		// Should not affect client1's message
		assert.True(t, hasPendingMessage(processor, msgID))
		assert.Equal(t, 1, getPendingMessageCount(processor))
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
		assert.Equal(t, 20, getPendingMessageCount(processor))

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
		assert.Equal(t, 0, getPendingMessageCount(processor))
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
		assert.True(t, hasPendingMessage(processor, messageID))

		// Wait for send attempt
		time.Sleep(50 * time.Millisecond)

		// Should have attempted to send despite error
		calls := mockSender.GetSendMessageCalls()
		assert.Len(t, calls, 1)

		// Message should still be pending for retry
		assert.True(t, hasPendingMessage(processor, messageID))
	})

	t.Run("processor stop cleans up resources", func(t *testing.T) {
		mockSender := NewMockMessageSender()
		processor := NewMessageProcessor(mockSender)

		msg := createTestMessage("stop_test")
		messageID, err := processor.SendMessageReliably("client1", msg)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		assert.True(t, hasPendingMessage(processor, messageID))

		// Stop processor
		processor.Stop()

		// Should not panic or hang
		// Note: We can't easily test that goroutines are cleaned up,
		// but Stop() should complete without blocking
	})
}
