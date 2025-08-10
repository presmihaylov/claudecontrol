package txmanager

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// MockTransactionManager is a mock implementation of the TransactionManager interface
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockTransactionManager) BeginTransaction(ctx context.Context) (context.Context, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(context.Context), args.Error(1)
}

func (m *MockTransactionManager) CommitTransaction(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTransactionManager) RollbackTransaction(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
