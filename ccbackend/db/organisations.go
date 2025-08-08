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

type PostgresOrganisationsRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for organizations table
var organisationsColumns = []string{
	"id",
	"created_at",
	"updated_at",
}

func NewPostgresOrganisationsRepository(db *sqlx.DB, schema string) *PostgresOrganisationsRepository {
	return &PostgresOrganisationsRepository{db: db, schema: schema}
}

func (r *PostgresOrganisationsRepository) CreateOrganisation(
	ctx context.Context,
	organization *models.Organisation,
) error {
	db := dbtx.GetTransactional(ctx, r.db)

	columnsStr := strings.Join(organisationsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.organizations (%s) 
		VALUES ($1, NOW(), NOW())`, r.schema, columnsStr)

	_, err := db.ExecContext(ctx, query, organization.ID)
	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	return nil
}

func (r *PostgresOrganisationsRepository) GetOrganisationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.Organisation], error) {
	db := dbtx.GetTransactional(ctx, r.db)

	columnsStr := strings.Join(organisationsColumns, ", ")

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.organizations 
		WHERE id = $1`, columnsStr, r.schema)

	organization := &models.Organisation{}
	err := db.QueryRowxContext(ctx, query, id).StructScan(organization)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return mo.None[*models.Organisation](), nil
		}
		return mo.None[*models.Organisation](), fmt.Errorf("failed to get organization by ID: %w", err)
	}

	return mo.Some(organization), nil
}
