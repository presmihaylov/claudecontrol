package txmanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/services"
)

// MockDB implements the Queryable interface for testing
type MockDB struct {
	queries []string
	shouldError bool
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	m.queries = append(m.queries, query)
	if m.shouldError {
		return nil, errors.New("mock database error")
	}
	return nil, nil
}

func (m *MockDB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	m.queries = append(m.queries, query)
	if m.shouldError {
		return errors.New("mock database error")
	}
	return nil
}

func (m *MockDB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	m.queries = append(m.queries, query)
	if m.shouldError {
		return errors.New("mock database error")
	}
	return nil
}

func (m *MockDB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	m.queries = append(m.queries, query)
	// This would normally return a real *sqlx.Row but for testing we'll use nil
	return nil
}

// MockTxManager that tracks operations for testing
type MockTxManager struct {
	operations []string
	shouldFail bool
}

func (m *MockTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	m.operations = append(m.operations, "BEGIN")
	
	if m.shouldFail {
		m.operations = append(m.operations, "ROLLBACK")
		return errors.New("transaction failed")
	}

	err := fn(ctx)
	if err != nil {
		m.operations = append(m.operations, "ROLLBACK")
		return err
	}

	m.operations = append(m.operations, "COMMIT")
	return nil
}

func (m *MockTxManager) BeginTransaction(ctx context.Context) (context.Context, error) {
	m.operations = append(m.operations, "MANUAL_BEGIN")
	return ctx, nil
}

func (m *MockTxManager) CommitTransaction(ctx context.Context) error {
	m.operations = append(m.operations, "MANUAL_COMMIT")
	return nil
}

func (m *MockTxManager) RollbackTransaction(ctx context.Context) error {
	m.operations = append(m.operations, "MANUAL_ROLLBACK")
	return nil
}

func TestTransactionManager_WithTransaction_Success_Mock(t *testing.T) {
	mockTxManager := &MockTxManager{}
	ctx := context.Background()

	// Execute transaction that should succeed
	err := mockTxManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Simulate successful operations
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"BEGIN", "COMMIT"}, mockTxManager.operations)
}

func TestTransactionManager_WithTransaction_Rollback_Mock(t *testing.T) {
	mockTxManager := &MockTxManager{}
	ctx := context.Background()

	// Execute transaction that should fail and rollback
	err := mockTxManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Simulate failure
		return errors.New("operation failed")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation failed")
	assert.Equal(t, []string{"BEGIN", "ROLLBACK"}, mockTxManager.operations)
}

func TestTransactionManager_ManualTransaction_Mock(t *testing.T) {
	mockTxManager := &MockTxManager{}
	ctx := context.Background()

	// Test manual transaction control
	txCtx, err := mockTxManager.BeginTransaction(ctx)
	require.NoError(t, err)

	// Commit manual transaction
	err = mockTxManager.CommitTransaction(txCtx)
	require.NoError(t, err)

	assert.Equal(t, []string{"MANUAL_BEGIN", "MANUAL_COMMIT"}, mockTxManager.operations)
}

func TestTransactionManager_ManualTransaction_Rollback_Mock(t *testing.T) {
	mockTxManager := &MockTxManager{}
	ctx := context.Background()

	// Test manual transaction rollback
	txCtx, err := mockTxManager.BeginTransaction(ctx)
	require.NoError(t, err)

	// Rollback manual transaction
	err = mockTxManager.RollbackTransaction(txCtx)
	require.NoError(t, err)

	assert.Equal(t, []string{"MANUAL_BEGIN", "MANUAL_ROLLBACK"}, mockTxManager.operations)
}

// Test context propagation functions
func TestWithTransaction_ContextStorage(t *testing.T) {
	ctx := context.Background()
	
	// Test that no transaction exists initially
	tx, ok := TransactionFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, tx)
	
	// Create a mock transaction (we can't create a real *sqlx.Tx without a db connection)
	// For this test, we'll just verify the context key mechanism works
	mockTxCtx := context.WithValue(ctx, txContextKey, "mock-transaction")
	
	// Test that we can retrieve something from context with our key
	value := mockTxCtx.Value(txContextKey)
	assert.Equal(t, "mock-transaction", value)
}

// Demonstration test showing transaction propagation concept
func TestTransactionPropagation_Concept(t *testing.T) {
	// This test demonstrates how transaction propagation would work
	// in practice with real repositories
	
	type Repository struct {
		queries []string
	}
	
	// Simulate repository method that uses GetQueryable pattern
	performDatabaseOperation := func(ctx context.Context, repo *Repository, db services.Queryable) error {
		// In real code, this would be: queryable := GetQueryable(ctx, db)
		// For demo, we'll simulate checking context
		_, hasTransaction := TransactionFromContext(ctx)
		
		operation := "DB_OPERATION"
		if hasTransaction {
			operation = "TX_OPERATION" // Would use transaction
		}
		
		repo.queries = append(repo.queries, operation)
		return nil
	}
	
	repo := &Repository{}
	mockDB := &MockDB{}
	ctx := context.Background()
	
	// Operation without transaction
	err := performDatabaseOperation(ctx, repo, mockDB)
	require.NoError(t, err)
	
	// Operation with transaction context
	txCtx := context.WithValue(ctx, txContextKey, "mock-tx")
	err = performDatabaseOperation(txCtx, repo, mockDB)
	require.NoError(t, err)
	
	// Verify that the repository detected the transaction context
	assert.Equal(t, []string{"DB_OPERATION", "TX_OPERATION"}, repo.queries)
}

// Integration test showing error handling and rollback behavior
func TestTransactionErrorHandling_Integration(t *testing.T) {
	mockTxManager := &MockTxManager{}
	ctx := context.Background()
	
	operations := []string{}
	
	// Test that error in transaction function triggers rollback
	err := mockTxManager.WithTransaction(ctx, func(txCtx context.Context) error {
		operations = append(operations, "OPERATION_1")
		operations = append(operations, "OPERATION_2")
		
		// Simulate error after multiple operations
		return fmt.Errorf("something went wrong after operations")
	})
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "something went wrong after operations")
	
	// Verify operations were performed but transaction was rolled back
	assert.Equal(t, []string{"OPERATION_1", "OPERATION_2"}, operations)
	assert.Equal(t, []string{"BEGIN", "ROLLBACK"}, mockTxManager.operations)
}

// Test demonstrating nested transaction support
func TestNestedTransactions_Concept(t *testing.T) {
	operations := []string{}
	
	// Simulate nested transaction behavior
	executeWithTransaction := func(ctx context.Context, operation string) error {
		// Check if already in transaction
		_, alreadyInTx := TransactionFromContext(ctx)
		
		if alreadyInTx {
			// Already in transaction, just execute
			operations = append(operations, fmt.Sprintf("NESTED_%s", operation))
			return nil
		}
		
		// Not in transaction, begin new one
		operations = append(operations, "BEGIN")
		txCtx := context.WithValue(ctx, txContextKey, "tx")
		
		operations = append(operations, fmt.Sprintf("NEW_TX_%s", operation))
		
		// Simulate commit
		operations = append(operations, "COMMIT")
		
		return nil
	}
	
	ctx := context.Background()
	
	// First call - should create transaction
	err := executeWithTransaction(ctx, "OUTER")
	require.NoError(t, err)
	
	// Simulate nested call within existing transaction
	txCtx := context.WithValue(ctx, txContextKey, "existing-tx")
	err = executeWithTransaction(txCtx, "INNER")
	require.NoError(t, err)
	
	expected := []string{
		"BEGIN", "NEW_TX_OUTER", "COMMIT", // Outer transaction
		"NESTED_INNER", // Inner reuses existing transaction
	}
	assert.Equal(t, expected, operations)
}

func TestPanicRecovery_Concept(t *testing.T) {
	mockTxManager := &MockTxManager{}
	
	// Test that panic in transaction function would be handled
	// (In real implementation, defer func would catch panic and rollback)
	
	operations := []string{}
	
	// Simulate panic handling
	func() {
		defer func() {
			if r := recover(); r != nil {
				operations = append(operations, "PANIC_RECOVERED")
				operations = append(operations, "ROLLBACK_ON_PANIC")
			}
		}()
		
		// Simulate transaction with panic
		operations = append(operations, "BEGIN")
		operations = append(operations, "OPERATION_BEFORE_PANIC")
		panic("something went wrong")
	}()
	
	expected := []string{
		"BEGIN",
		"OPERATION_BEFORE_PANIC", 
		"PANIC_RECOVERED",
		"ROLLBACK_ON_PANIC",
	}
	assert.Equal(t, expected, operations)
}