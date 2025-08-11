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
	"email",
	"organization_id",
	"created_at",
	"updated_at",
}

func NewPostgresUsersRepository(db *sqlx.DB, schema string) *PostgresUsersRepository {
	return &PostgresUsersRepository{db: db, schema: schema}
}

func (r *PostgresUsersRepository) GetUserByAuthProvider(
	ctx context.Context,
	authProvider, authProviderID string,
	forUpdate bool,
) (*models.User, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	returningStr := strings.Join(usersColumns, ", ")
	forUpdateClause := ""
	if forUpdate {
		forUpdateClause = " FOR UPDATE"
	}

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.users 
		WHERE auth_provider = $1 AND auth_provider_id = $2%s`,
		returningStr, r.schema, forUpdateClause)

	user := &models.User{}
	err := db.QueryRowxContext(ctx, query, authProvider, authProviderID).StructScan(user)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by auth provider: %w", err)
	}

	return user, nil
}

func (r *PostgresUsersRepository) GetUserByClerkID(
	ctx context.Context,
	clerkID string,
) (*models.User, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	returningStr := strings.Join(usersColumns, ", ")

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.users 
		WHERE auth_provider = 'clerk' AND auth_provider_id = $1`,
		returningStr, r.schema)

	user := &models.User{}
	err := db.QueryRowxContext(ctx, query, clerkID).StructScan(user)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by clerk ID: %w", err)
	}

	return user, nil
}

func (r *PostgresUsersRepository) CreateUser(
	ctx context.Context,
	authProvider, authProviderID, email string,
	organizationID models.OrgID,
) (*models.User, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	// Generate ULID for new users
	userID := core.NewID("u")

	insertColumns := []string{
		"id",
		"auth_provider",
		"auth_provider_id",
		"email",
		"organization_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(usersColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.users (%s) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	user := &models.User{}
	err := db.QueryRowxContext(ctx, query, userID, authProvider, authProviderID, email, organizationID).StructScan(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}
