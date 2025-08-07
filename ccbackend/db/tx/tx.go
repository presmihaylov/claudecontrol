package tx

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// contextKey type for storing transaction in context
type contextKey string

const txContextKey contextKey = "database_transaction"

// WithTransaction stores a transaction in the context
func WithTransaction(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey, tx)
}

// TransactionFromContext extracts a transaction from the context
func TransactionFromContext(ctx context.Context) (*sqlx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey).(*sqlx.Tx)
	return tx, ok
}

// Transactional interface that both *sqlx.DB and *sqlx.Tx implement
type Transactional interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row
}

// GetTransactional returns transaction if available in context, otherwise returns db
// This is the key function that repositories will use to get the correct queryable
func GetTransactional(ctx context.Context, db *sqlx.DB) Transactional {
	if tx, ok := TransactionFromContext(ctx); ok {
		return tx // Return transaction if available
	}
	return db // Return regular DB connection
}
