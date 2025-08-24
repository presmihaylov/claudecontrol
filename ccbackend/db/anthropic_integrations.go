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

type PostgresAnthropicIntegrationsRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for anthropic_integrations table
var anthropicIntegrationsColumns = []string{
	"id",
	"anthropic_api_key",
	"claude_code_oauth_token",
	"organization_id",
	"created_at",
	"updated_at",
}

func NewPostgresAnthropicIntegrationsRepository(db *sqlx.DB, schema string) *PostgresAnthropicIntegrationsRepository {
	return &PostgresAnthropicIntegrationsRepository{db: db, schema: schema}
}

func (r *PostgresAnthropicIntegrationsRepository) CreateAnthropicIntegration(
	ctx context.Context,
	integration *models.AnthropicIntegration,
) error {
	insertColumns := []string{
		"id",
		"anthropic_api_key",
		"claude_code_oauth_token",
		"organization_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(anthropicIntegrationsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.anthropic_integrations (%s) 
		VALUES ($1, $2, $3, $4, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query, integration.ID, integration.AnthropicAPIKey, integration.ClaudeCodeOAuthToken, integration.OrgID).
		StructScan(integration)
	if err != nil {
		return fmt.Errorf("failed to create anthropic integration: %w", err)
	}

	return nil
}

func (r *PostgresAnthropicIntegrationsRepository) GetAnthropicIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID models.OrgID,
) ([]models.AnthropicIntegration, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	columnsStr := strings.Join(anthropicIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.anthropic_integrations 
		WHERE organization_id = $1 
		ORDER BY created_at DESC`, columnsStr, r.schema)

	integrations := []models.AnthropicIntegration{}
	err := r.db.SelectContext(ctx, &integrations, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get anthropic integrations: %w", err)
	}

	return integrations, nil
}

func (r *PostgresAnthropicIntegrationsRepository) GetAnthropicIntegrationByID(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) (mo.Option[*models.AnthropicIntegration], error) {
	columnsStr := strings.Join(anthropicIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.anthropic_integrations 
		WHERE id = $1 AND organization_id = $2`, columnsStr, r.schema)

	var integration models.AnthropicIntegration
	err := r.db.GetContext(ctx, &integration, query, id, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.AnthropicIntegration](), nil
		}
		return mo.None[*models.AnthropicIntegration](), fmt.Errorf("failed to get anthropic integration: %w", err)
	}

	return mo.Some(&integration), nil
}

func (r *PostgresAnthropicIntegrationsRepository) DeleteAnthropicIntegration(
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
		DELETE FROM %s.anthropic_integrations 
		WHERE id = $1 AND organization_id = $2`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, organizationID)
	if err != nil {
		return fmt.Errorf("failed to delete anthropic integration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("anthropic integration not found")
	}

	return nil
}
