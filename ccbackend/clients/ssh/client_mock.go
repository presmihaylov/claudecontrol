package ssh

import (
	"github.com/stretchr/testify/mock"
)

// MockSSHClient is a mock implementation of SSHClientInterface
type MockSSHClient struct {
	mock.Mock
}

// ExecuteCommand mocks the ExecuteCommand method
func (m *MockSSHClient) ExecuteCommand(host, command string) error {
	args := m.Called(host, command)
	return args.Error(0)
}
