package txmanager

import (
	"context"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	dbtx "ccbackend/db/tx"
)

// TransactionManager implements the TransactionManager interface
type TransactionManager struct {
	db *sqlx.DB
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *sqlx.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// WithTransaction executes the provided function within a database transaction
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	log.Printf("📋 Starting transaction")

	// Support nested transactions - if already in tx, just execute function
	if _, ok := dbtx.TransactionFromContext(ctx); ok {
		log.Printf("📋 Already in transaction, executing function directly")
		return fn(ctx)
	}

	// Begin new transaction
	tx, err := tm.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Panic protection with defer
	defer func() {
		if r := recover(); r != nil {
			log.Printf("📋 Transaction panic detected, rolling back: %v", r)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("📋 Failed to rollback after panic: %v", rollbackErr)
			}
			panic(r) // Re-panic to maintain normal panic behavior
		}
	}()

	// Create context with transaction
	txCtx := dbtx.WithTransaction(ctx, tx)

	// Execute function with transaction context
	if err := fn(txCtx); err != nil {
		log.Printf("📋 Transaction function returned error, rolling back: %v", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("transaction failed: %w, rollback failed: %v", err, rollbackErr)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("📋 Transaction completed successfully")
	return nil
}

// BeginTransaction starts a new transaction and returns context with the transaction
func (tm *TransactionManager) BeginTransaction(ctx context.Context) (context.Context, error) {
	log.Printf("📋 Starting manual transaction")

	tx, err := tm.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return dbtx.WithTransaction(ctx, tx), nil
}

// CommitTransaction commits the transaction stored in the context
func (tm *TransactionManager) CommitTransaction(ctx context.Context) error {
	log.Printf("📋 Committing manual transaction")

	tx, ok := dbtx.TransactionFromContext(ctx)
	if !ok {
		return fmt.Errorf("no transaction found in context")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("📋 Manual transaction committed successfully")
	return nil
}

// RollbackTransaction rolls back the transaction stored in the context
func (tm *TransactionManager) RollbackTransaction(ctx context.Context) error {
	log.Printf("📋 Rolling back manual transaction")

	tx, ok := dbtx.TransactionFromContext(ctx)
	if !ok {
		return fmt.Errorf("no transaction found in context")
	}

	if err := tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	log.Printf("📋 Manual transaction rolled back successfully")
	return nil
}
