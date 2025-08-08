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
	"organization_id",
	"created_at",
	"updated_at",
}

func NewPostgresUsersRepository(db *sqlx.DB, schema string) *PostgresUsersRepository {
	return &PostgresUsersRepository{db: db, schema: schema}
}

func (r *PostgresUsersRepository) GetOrCreateUser(
	ctx context.Context,
	authProvider, authProviderID string,
) (*models.User, error) {
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

func (r *PostgresUsersRepository) UpdateUserOrganization(
	ctx context.Context,
	userID, organizationID string,
) error {
	db := dbtx.GetTransactional(ctx, r.db)

	query := fmt.Sprintf(`
		UPDATE %s.users 
		SET organization_id = $1, updated_at = NOW() 
		WHERE id = $2`, r.schema)

	result, err := db.ExecContext(ctx, query, organizationID, userID)
	if err != nil {
		return fmt.Errorf("failed to update user organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
