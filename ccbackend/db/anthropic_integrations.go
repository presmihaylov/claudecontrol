package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/samber/mo"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	"ccbackend/core"
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
	"claude_code_oauth_refresh_token",
	"claude_code_oauth_token_expires_at",
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
		"claude_code_oauth_refresh_token",
		"claude_code_oauth_token_expires_at",
		"organization_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(anthropicIntegrationsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.anthropic_integrations (%s) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query,
		integration.ID,
		integration.AnthropicAPIKey,
		integration.ClaudeCodeOAuthToken,
		integration.ClaudeCodeOAuthRefreshToken,
		integration.ClaudeCodeOAuthTokenExpiresAt,
		integration.OrgID).StructScan(integration)
	if err != nil {
		log.Printf("📋 DB: Failed to create Anthropic integration: %v", err)
		return fmt.Errorf("failed to create anthropic integration: %w", err)
	}

	log.Printf("📋 DB: Successfully created Anthropic integration with ID: %s", integration.ID)
	return nil
}

func (r *PostgresAnthropicIntegrationsRepository) GetAnthropicIntegrationsByOrganizationID(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.AnthropicIntegration, error) {
	if !core.IsValidULID(orgID) {
		return nil, fmt.Errorf("organization ID must be a valid ULID")
	}

	columnsStr := strings.Join(anthropicIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.anthropic_integrations 
		WHERE organization_id = $1 
		ORDER BY created_at DESC`, columnsStr, r.schema)

	integrations := []models.AnthropicIntegration{}
	err := r.db.SelectContext(ctx, &integrations, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get anthropic integrations: %w", err)
	}

	return integrations, nil
}

func (r *PostgresAnthropicIntegrationsRepository) GetAnthropicIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.AnthropicIntegration], error) {
	columnsStr := strings.Join(anthropicIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.anthropic_integrations 
		WHERE id = $1 AND organization_id = $2`, columnsStr, r.schema)

	var integration models.AnthropicIntegration
	err := r.db.GetContext(ctx, &integration, query, id, orgID)
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
	orgID models.OrgID,
	id string,
) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.anthropic_integrations 
		WHERE id = $1 AND organization_id = $2`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, orgID)
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

func (r *PostgresAnthropicIntegrationsRepository) UpdateAnthropicIntegration(
	ctx context.Context,
	integration *models.AnthropicIntegration,
) error {
	log.Printf("📋 DB: Updating Anthropic integration: %s", integration.ID)

	query := fmt.Sprintf(`
		UPDATE %s.anthropic_integrations 
		SET anthropic_api_key = $1,
		    claude_code_oauth_token = $2,
		    claude_code_oauth_refresh_token = $3,
		    claude_code_oauth_token_expires_at = $4,
		    updated_at = NOW()
		WHERE id = $5 AND organization_id = $6
		RETURNING %s`, r.schema, strings.Join(anthropicIntegrationsColumns, ", "))

	err := r.db.QueryRowxContext(ctx, query,
		integration.AnthropicAPIKey,
		integration.ClaudeCodeOAuthToken,
		integration.ClaudeCodeOAuthRefreshToken,
		integration.ClaudeCodeOAuthTokenExpiresAt,
		integration.ID,
		integration.OrgID).StructScan(integration)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("anthropic integration not found")
		}
		log.Printf("📋 DB: Failed to update Anthropic integration: %v", err)
		return fmt.Errorf("failed to update anthropic integration: %w", err)
	}

	log.Printf("📋 DB: Successfully updated Anthropic integration: %s", integration.ID)
	return nil
}
