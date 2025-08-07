package services

import (
	"context"
)

// MockTransactionManager for testing
type MockTransactionManager struct{}

func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx) // Just execute the function directly for tests
}

func (m *MockTransactionManager) BeginTransaction(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (m *MockTransactionManager) CommitTransaction(ctx context.Context) error {
	return nil
}

func (m *MockTransactionManager) RollbackTransaction(ctx context.Context) error {
	return nil
}
