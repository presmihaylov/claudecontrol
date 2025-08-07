package db

import (
	"context"
	"fmt"

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

func NewPostgresUsersRepository(db *sqlx.DB, schema string) *PostgresUsersRepository {
	return &PostgresUsersRepository{db: db, schema: schema}
}

func (r *PostgresUsersRepository) GetOrCreateUser(ctx context.Context, authProvider, authProviderID string) (*models.User, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	// Generate ULID for new users
	userID := core.NewID("u")

	query := fmt.Sprintf(`
		INSERT INTO %s.users (id, auth_provider, auth_provider_id, created_at, updated_at) 
		VALUES ($1, $2, $3, NOW(), NOW()) 
		ON CONFLICT (auth_provider, auth_provider_id) 
		DO UPDATE SET updated_at = NOW()
		RETURNING id, auth_provider, auth_provider_id, created_at, updated_at`, r.schema)

	user := &models.User{}
	err := db.QueryRowxContext(ctx, query, userID, authProvider, authProviderID).StructScan(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	return user, nil
}
