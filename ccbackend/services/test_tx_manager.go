package services

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	dbtx "ccbackend/db/tx"
)

// TestTransactionManager implements TransactionManager interface for tests
type TestTransactionManager struct {
	db *sqlx.DB
}

func (tm *TestTransactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	// Begin new transaction
	tx, err := tm.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create context with transaction
	txCtx := dbtx.WithTransaction(ctx, tx)

	// Execute function with transaction context
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit()
}

func (tm *TestTransactionManager) BeginTransaction(ctx context.Context) (context.Context, error) {
	tx, err := tm.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return dbtx.WithTransaction(ctx, tx), nil
}

func (tm *TestTransactionManager) CommitTransaction(ctx context.Context) error {
	tx, ok := dbtx.TransactionFromContext(ctx)
	if !ok {
		return fmt.Errorf("no transaction found in context")
	}
	return tx.Commit()
}

func (tm *TestTransactionManager) RollbackTransaction(ctx context.Context) error {
	tx, ok := dbtx.TransactionFromContext(ctx)
	if !ok {
		return fmt.Errorf("no transaction found in context")
	}
	return tx.Rollback()
}
