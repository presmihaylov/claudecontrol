package socketio

import (
	"github.com/stretchr/testify/mock"
)

// MockSocketIOClient is a mock implementation of SocketIOClient for testing
type MockSocketIOClient struct {
	mock.Mock
}

// SendMessage mocks sending a message to a specific connection
func (m *MockSocketIOClient) SendMessage(connectionID string, message any) error {
	args := m.Called(connectionID, message)
	return args.Error(0)
}

// BroadcastMessage mocks broadcasting a message to all connections
func (m *MockSocketIOClient) BroadcastMessage(message any) error {
	args := m.Called(message)
	return args.Error(0)
}