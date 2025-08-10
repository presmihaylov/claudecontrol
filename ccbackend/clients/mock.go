package clients

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/mock"
)

// MockSocketIOClient implements SocketIOClient for testing
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

func (m *MockSocketIOClient) RegisterMessageHandler(handler MessageHandlerFunc) {
	m.Called(handler)
}

func (m *MockSocketIOClient) RegisterConnectionHook(hook ConnectionHookFunc) {
	m.Called(hook)
}

func (m *MockSocketIOClient) RegisterDisconnectionHook(hook ConnectionHookFunc) {
	m.Called(hook)
}

func (m *MockSocketIOClient) RegisterPingHook(hook PingHandlerFunc) {
	m.Called(hook)
}