package db

import (
	"context"
	"database/sql"
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
	"cc_agent_system_secret_key",
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

	insertColumns := []string{
		"id",
		"cc_agent_system_secret_key",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(organizationsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.organizations (%s) 
		VALUES ($1, $2, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := db.QueryRowxContext(ctx, query, organization.ID, organization.CCAgentSystemSecretKey).
		StructScan(organization)
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

func (r *PostgresOrganizationsRepository) GenerateCCAgentSecretKey(
	ctx context.Context,
	organizationID models.OrgID,
	secretKey string,
) (bool, error) {
	if secretKey == "" {
		return false, fmt.Errorf("secret key cannot be empty")
	}

	db := dbtx.GetTransactional(ctx, r.db)

	query := fmt.Sprintf(`
		UPDATE %s.organizations 
		SET ccagent_secret_key = $1, ccagent_secret_key_generated_at = NOW(), updated_at = NOW()
		WHERE id = $2`, r.schema)

	result, err := db.ExecContext(ctx, query, secretKey, organizationID)
	if err != nil {
		return false, fmt.Errorf("failed to update organization with secret key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *PostgresOrganizationsRepository) GetOrganizationBySecretKey(
	ctx context.Context,
	secretKey string,
) (mo.Option[*models.Organization], error) {
	if secretKey == "" {
		return mo.None[*models.Organization](), fmt.Errorf("secret key cannot be empty")
	}

	db := dbtx.GetTransactional(ctx, r.db)

	columnsStr := strings.Join(organizationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.organizations 
		WHERE ccagent_secret_key = $1 AND ccagent_secret_key IS NOT NULL`, columnsStr, r.schema)

	var organization models.Organization
	err := db.GetContext(ctx, &organization, query, secretKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.Organization](), nil
		}
		return mo.None[*models.Organization](), fmt.Errorf("failed to get organization by secret key: %w", err)
	}

	return mo.Some(&organization), nil
}

func (r *PostgresOrganizationsRepository) GetOrganizationBySystemSecretKey(
	ctx context.Context,
	systemSecretKey string,
) (mo.Option[*models.Organization], error) {
	if systemSecretKey == "" {
		return mo.None[*models.Organization](), fmt.Errorf("system secret key cannot be empty")
	}

	db := dbtx.GetTransactional(ctx, r.db)

	columnsStr := strings.Join(organizationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.organizations 
		WHERE cc_agent_system_secret_key = $1`, columnsStr, r.schema)

	var organization models.Organization
	err := db.GetContext(ctx, &organization, query, systemSecretKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.Organization](), nil
		}
		return mo.None[*models.Organization](), fmt.Errorf("failed to get organization by system secret key: %w", err)
	}

	return mo.Some(&organization), nil
}

func (r *PostgresOrganizationsRepository) GetAllOrganizations(ctx context.Context) ([]*models.Organization, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	columnsStr := strings.Join(organizationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.organizations 
		ORDER BY created_at ASC`, columnsStr, r.schema)

	var organizations []*models.Organization
	err := db.SelectContext(ctx, &organizations, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all organizations: %w", err)
	}

	return organizations, nil
}
