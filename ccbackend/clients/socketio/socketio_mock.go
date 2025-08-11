package socketio

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/mock"

	"ccbackend/clients"
)

// MockSocketIOClient is a mock implementation of the SocketIOClient interface
type MockSocketIOClient struct {
	mock.Mock
}

func (m *MockSocketIOClient) RegisterWithRouter(router *mux.Router) {
	m.Called(router)
}

func (m *MockSocketIOClient) GetClientIDs() []string {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

func (m *MockSocketIOClient) GetClientByID(clientID string) any {
	args := m.Called(clientID)
	return args.Get(0)
}

func (m *MockSocketIOClient) SendMessage(clientID string, msg any) error {
	args := m.Called(clientID, msg)
	return args.Error(0)
}

func (m *MockSocketIOClient) DisconnectClientByID(clientID string) error {
	args := m.Called(clientID)
	return args.Error(0)
}

func (m *MockSocketIOClient) RegisterMessageHandler(handler clients.MessageHandlerFunc) {
	m.Called(handler)
}

func (m *MockSocketIOClient) RegisterConnectionHook(hook clients.ConnectionHookFunc) {
	m.Called(hook)
}

func (m *MockSocketIOClient) RegisterDisconnectionHook(hook clients.ConnectionHookFunc) {
	m.Called(hook)
}

func (m *MockSocketIOClient) RegisterPingHook(hook clients.PingHandlerFunc) {
	m.Called(hook)
}
