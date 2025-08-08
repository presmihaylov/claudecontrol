package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/samber/mo"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	dbtx "ccbackend/db/tx"
	"ccbackend/models"
)

type PostgresOrganizationsRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for organizations table
var organizationsColumns = []string{
	"id",
	"ccagent_secret_key",
	"ccagent_secret_key_generated_at",
	"created_at",
	"updated_at",
}

func NewPostgresOrganizationsRepository(db *sqlx.DB, schema string) *PostgresOrganizationsRepository {
	return &PostgresOrganizationsRepository{db: db, schema: schema}
}

func (r *PostgresOrganizationsRepository) CreateOrganization(
	ctx context.Context,
	organization *models.Organization,
) error {
	db := dbtx.GetTransactional(ctx, r.db)

	columnsStr := strings.Join(organizationsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.organizations (%s) 
		VALUES ($1, NULL, NULL, NOW(), NOW())`, r.schema, columnsStr)

	_, err := db.ExecContext(ctx, query, organization.ID)
	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	return nil
}

func (r *PostgresOrganizationsRepository) GetOrganizationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.Organization], error) {
	db := dbtx.GetTransactional(ctx, r.db)

	columnsStr := strings.Join(organizationsColumns, ", ")

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.organizations 
		WHERE id = $1`, columnsStr, r.schema)

	organization := &models.Organization{}
	err := db.QueryRowxContext(ctx, query, id).StructScan(organization)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return mo.None[*models.Organization](), nil
		}
		return mo.None[*models.Organization](), fmt.Errorf("failed to get organization by ID: %w", err)
	}

	return mo.Some(organization), nil
}

func (r *PostgresOrganizationsRepository) UpdateCCAgentSecretKey(
	ctx context.Context,
	organizationID string,
	secretKey string,
) error {
	db := dbtx.GetTransactional(ctx, r.db)

	query := fmt.Sprintf(`
		UPDATE %s.organizations 
		SET ccagent_secret_key = $1, ccagent_secret_key_generated_at = NOW(), updated_at = NOW()
		WHERE id = $2`, r.schema)

	result, err := db.ExecContext(ctx, query, secretKey, organizationID)
	if err != nil {
		return fmt.Errorf("failed to update ccagent secret key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

func (r *PostgresOrganizationsRepository) GetOrganizationBySecretKey(
	ctx context.Context,
	secretKey string,
) (mo.Option[*models.Organization], error) {
	db := dbtx.GetTransactional(ctx, r.db)

	columnsStr := strings.Join(organizationsColumns, ", ")

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.organizations 
		WHERE ccagent_secret_key = $1`, columnsStr, r.schema)

	organization := &models.Organization{}
	err := db.QueryRowxContext(ctx, query, secretKey).StructScan(organization)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return mo.None[*models.Organization](), nil
		}
		return mo.None[*models.Organization](), fmt.Errorf("failed to get organization by secret key: %w", err)
	}

	return mo.Some(organization), nil
}
