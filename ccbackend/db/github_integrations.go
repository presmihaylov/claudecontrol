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

	"ccbackend/models"
)

type PostgresGitHubIntegrationsRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for github_integrations table
var githubIntegrationsColumns = []string{
	"id",
	"github_installation_id",
	"github_access_token",
	"organization_id",
	"created_at",
	"updated_at",
}

func NewPostgresGitHubIntegrationsRepository(db *sqlx.DB, schema string) *PostgresGitHubIntegrationsRepository {
	return &PostgresGitHubIntegrationsRepository{db: db, schema: schema}
}

func (r *PostgresGitHubIntegrationsRepository) CreateGitHubIntegration(
	ctx context.Context,
	integration *models.GitHubIntegration,
) error {
	insertColumns := []string{
		"id",
		"github_installation_id",
		"github_access_token",
		"organization_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(githubIntegrationsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.github_integrations (%s) 
		VALUES ($1, $2, $3, $4, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query, integration.ID, integration.GitHubInstallationID, integration.GitHubAccessToken, integration.OrgID).
		StructScan(integration)
	if err != nil {
		return fmt.Errorf("failed to create github integration: %w", err)
	}

	return nil
}

func (r *PostgresGitHubIntegrationsRepository) GetGitHubIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID models.OrgID,
) ([]models.GitHubIntegration, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	columnsStr := strings.Join(githubIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.github_integrations 
		WHERE organization_id = $1 
		ORDER BY created_at DESC`, columnsStr, r.schema)

	integrations := []models.GitHubIntegration{}
	err := r.db.SelectContext(ctx, &integrations, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get github integrations: %w", err)
	}

	return integrations, nil
}

func (r *PostgresGitHubIntegrationsRepository) GetGitHubIntegrationByID(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) (mo.Option[*models.GitHubIntegration], error) {
	columnsStr := strings.Join(githubIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.github_integrations 
		WHERE id = $1 AND organization_id = $2`, columnsStr, r.schema)

	var integration models.GitHubIntegration
	err := r.db.GetContext(ctx, &integration, query, id, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.GitHubIntegration](), nil
		}
		return mo.None[*models.GitHubIntegration](), fmt.Errorf("failed to get github integration: %w", err)
	}

	return mo.Some(&integration), nil
}

func (r *PostgresGitHubIntegrationsRepository) DeleteGitHubIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) error {
	if organizationID == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	if id == "" {
		return fmt.Errorf("integration ID cannot be empty")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s.github_integrations 
		WHERE id = $1 AND organization_id = $2`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, organizationID)
	if err != nil {
		return fmt.Errorf("failed to delete github integration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("github integration not found")
	}

	return nil
}
