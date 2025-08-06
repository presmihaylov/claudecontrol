package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	"ccbackend/core"
	"ccbackend/models"
)

type PostgresSlackIntegrationsRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for slack_integrations table
var slackIntegrationsColumns = []string{
	"id",
	"slack_team_id",
	"slack_auth_token",
	"slack_team_name",
	"user_id",
	"ccagent_secret_key",
	"ccagent_secret_key_generated_at",
	"created_at",
	"updated_at",
}

func NewPostgresSlackIntegrationsRepository(db *sqlx.DB, schema string) *PostgresSlackIntegrationsRepository {
	return &PostgresSlackIntegrationsRepository{db: db, schema: schema}
}

func (r *PostgresSlackIntegrationsRepository) CreateSlackIntegration(ctx context.Context, integration *models.SlackIntegration) error {
	insertColumns := []string{"id", "slack_team_id", "slack_auth_token", "slack_team_name", "user_id", "created_at", "updated_at"}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(slackIntegrationsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.slack_integrations (%s) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query, integration.ID, integration.SlackTeamID, integration.SlackAuthToken, integration.SlackTeamName, integration.UserID).StructScan(integration)
	if err != nil {
		return fmt.Errorf("failed to create slack integration: %w", err)
	}

	return nil
}

func (r *PostgresSlackIntegrationsRepository) GetSlackIntegrationsByUserID(ctx context.Context, userID string) ([]*models.SlackIntegration, error) {
	if !core.IsValidULID(userID) {
		return nil, fmt.Errorf("user ID must be a valid ULID")
	}

	columnsStr := strings.Join(slackIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.slack_integrations 
		WHERE user_id = $1 
		ORDER BY created_at DESC`, columnsStr, r.schema)

	var integrations []*models.SlackIntegration
	err := r.db.SelectContext(ctx, &integrations, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integrations by user ID: %w", err)
	}

	return integrations, nil
}

func (r *PostgresSlackIntegrationsRepository) GetAllSlackIntegrations(ctx context.Context) ([]*models.SlackIntegration, error) {
	columnsStr := strings.Join(slackIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.slack_integrations 
		ORDER BY created_at DESC`, columnsStr, r.schema)

	var integrations []*models.SlackIntegration
	err := r.db.SelectContext(ctx, &integrations, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all slack integrations: %w", err)
	}

	return integrations, nil
}

func (r *PostgresSlackIntegrationsRepository) DeleteSlackIntegrationByID(ctx context.Context, integrationID, userID string) error {
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("integration ID must be a valid ULID")
	}

	if !core.IsValidULID(userID) {
		return fmt.Errorf("user ID must be a valid ULID")
	}

	query := fmt.Sprintf(`DELETE FROM %s.slack_integrations WHERE id = $1 AND user_id = $2`, r.schema)

	result, err := r.db.ExecContext(ctx, query, integrationID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete slack integration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return core.ErrNotFound
	}

	return nil
}

func (r *PostgresSlackIntegrationsRepository) GenerateCCAgentSecretKey(ctx context.Context, integrationID string, userID string, secretKey string) error {
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("integration ID must be a valid ULID")
	}

	if !core.IsValidULID(userID) {
		return fmt.Errorf("user ID must be a valid ULID")
	}

	if secretKey == "" {
		return fmt.Errorf("secret key cannot be empty")
	}

	query := fmt.Sprintf(`
		UPDATE %s.slack_integrations 
		SET ccagent_secret_key = $1, ccagent_secret_key_generated_at = NOW(), updated_at = NOW()
		WHERE id = $2 AND user_id = $3`, r.schema)

	result, err := r.db.ExecContext(ctx, query, secretKey, integrationID, userID)
	if err != nil {
		return fmt.Errorf("failed to update slack integration with secret key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return core.ErrNotFound
	}

	return nil
}

func (r *PostgresSlackIntegrationsRepository) GetSlackIntegrationBySecretKey(ctx context.Context, secretKey string) (*models.SlackIntegration, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("secret key cannot be empty")
	}

	columnsStr := strings.Join(slackIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.slack_integrations 
		WHERE ccagent_secret_key = $1 AND ccagent_secret_key IS NOT NULL`, columnsStr, r.schema)

	var integration models.SlackIntegration
	err := r.db.GetContext(ctx, &integration, query, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integration by secret key: %w", err)
	}

	return &integration, nil
}

func (r *PostgresSlackIntegrationsRepository) GetSlackIntegrationByTeamID(ctx context.Context, teamID string) (*models.SlackIntegration, error) {
	if teamID == "" {
		return nil, fmt.Errorf("team ID cannot be empty")
	}

	columnsStr := strings.Join(slackIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.slack_integrations 
		WHERE slack_team_id = $1`, columnsStr, r.schema)

	var integration models.SlackIntegration
	err := r.db.GetContext(ctx, &integration, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integration by team ID: %w", err)
	}

	return &integration, nil
}

func (r *PostgresSlackIntegrationsRepository) GetSlackIntegrationByID(ctx context.Context, id string) (*models.SlackIntegration, error) {
	if !core.IsValidULID(id) {
		return nil, fmt.Errorf("integration ID must be a valid ULID")
	}

	columnsStr := strings.Join(slackIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.slack_integrations 
		WHERE id = $1`, columnsStr, r.schema)

	var integration models.SlackIntegration
	err := r.db.GetContext(ctx, &integration, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integration by ID: %w", err)
	}

	return &integration, nil
}
