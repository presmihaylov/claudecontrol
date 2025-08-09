package clients

import (
	"github.com/gorilla/mux"
	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"
)

// MockSocketIOClient is a mock implementation of SocketIOClient
type MockSocketIOClient struct {
	mock.Mock
}

func (m *MockSocketIOClient) SendMessage(clientID string, message any) error {
	args := m.Called(clientID, message)
	return args.Error(0)
}

func (m *MockSocketIOClient) GetClientIDs() []string {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

func (m *MockSocketIOClient) RegisterClient(client *Client) error {
	args := m.Called(client)
	return args.Error(0)
}

func (m *MockSocketIOClient) DeregisterClient(clientID string) error {
	args := m.Called(clientID)
	return args.Error(0)
}

func (m *MockSocketIOClient) GetClient(clientID string) (mo.Option[*Client], error) {
	args := m.Called(clientID)
	return args.Get(0).(mo.Option[*Client]), args.Error(1)
}

func (m *MockSocketIOClient) GetClientByID(clientID string) any {
	args := m.Called(clientID)
	return args.Get(0)
}

func (m *MockSocketIOClient) RegisterWithRouter(router *mux.Router) {
	m.Called(router)
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
