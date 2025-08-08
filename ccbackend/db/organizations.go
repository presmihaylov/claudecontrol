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
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.organizations (%s) 
		VALUES ($1, NOW(), NOW())`,
		r.schema, columnsStr)

	_, err := db.ExecContext(ctx, query,
		organization.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert organization: %w", err)
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
		WHERE id = $1`,
		columnsStr, r.schema)

	var organization models.Organization
	err := db.GetContext(ctx, &organization, query, id)
	if err != nil {
		return mo.None[*models.Organization](), fmt.Errorf("failed to get organization by ID: %w", err)
	}

	return mo.Some(&organization), nil
}
