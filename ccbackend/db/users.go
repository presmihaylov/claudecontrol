package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	"ccbackend/core"
	dbtx "ccbackend/db/tx"
	"ccbackend/models"
)

type PostgresUsersRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for users table
var usersColumns = []string{
	"id",
	"auth_provider",
	"auth_provider_id",
	"created_at",
	"updated_at",
}

func NewPostgresUsersRepository(db *sqlx.DB, schema string) *PostgresUsersRepository {
	return &PostgresUsersRepository{db: db, schema: schema}
}

func (r *PostgresUsersRepository) GetOrCreateUser(ctx context.Context, authProvider, authProviderID string) (*models.User, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	// Generate ULID for new users
	userID := core.NewID("u")

	insertColumns := []string{"id", "auth_provider", "auth_provider_id", "created_at", "updated_at"}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(usersColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.users (%s) 
		VALUES ($1, $2, $3, NOW(), NOW()) 
		ON CONFLICT (auth_provider, auth_provider_id) 
		DO UPDATE SET updated_at = NOW()
		RETURNING %s`, r.schema, columnsStr, returningStr)

	user := &models.User{}
	err := db.QueryRowxContext(ctx, query, userID, authProvider, authProviderID).StructScan(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	return user, nil
}
