package services

import (
	"errors"
	"sync"
	"time"
)

// MockMessageSender implements the MessageSender interface for testing
type MockMessageSender struct {
	// Track calls for verification
	SendMessageCalls []SendMessageCall
	callsMutex       sync.RWMutex

	// Configure responses
	SendMessageError error
	SendMessageDelay time.Duration

	// Advanced scenarios
	FailAfterCalls int
	currentCalls   int
}

type SendMessageCall struct {
	ClientID  string
	Message   any
	Timestamp time.Time
}

// NewMockMessageSender creates a new mock message sender
func NewMockMessageSender() *MockMessageSender {
	return &MockMessageSender{
		SendMessageCalls: make([]SendMessageCall, 0),
	}
}

// SendMessage implements the MessageSender interface
func (m *MockMessageSender) SendMessage(clientID string, msg any) error {
	m.callsMutex.Lock()
	defer m.callsMutex.Unlock()

	// Add delay if configured
	if m.SendMessageDelay > 0 {
		time.Sleep(m.SendMessageDelay)
	}

	// Check if we should fail after certain number of calls
	if m.FailAfterCalls > 0 && m.currentCalls >= m.FailAfterCalls {
		m.currentCalls++
		return errors.New("mock: configured to fail after calls")
	}

	// Record the call
	m.SendMessageCalls = append(m.SendMessageCalls, SendMessageCall{
		ClientID:  clientID,
		Message:   msg,
		Timestamp: time.Now(),
	})

	m.currentCalls++

	// Return configured error if set
	if m.SendMessageError != nil {
		return m.SendMessageError
	}

	return nil
}

// GetSendMessageCalls returns a copy of the calls for thread-safe access
func (m *MockMessageSender) GetSendMessageCalls() []SendMessageCall {
	m.callsMutex.RLock()
	defer m.callsMutex.RUnlock()

	calls := make([]SendMessageCall, len(m.SendMessageCalls))
	copy(calls, m.SendMessageCalls)
	return calls
}

// Reset clears all recorded calls and resets state
func (m *MockMessageSender) Reset() {
	m.callsMutex.Lock()
	defer m.callsMutex.Unlock()

	m.SendMessageCalls = make([]SendMessageCall, 0)
	m.currentCalls = 0
	m.SendMessageError = nil
	m.SendMessageDelay = 0
	m.FailAfterCalls = 0
}
