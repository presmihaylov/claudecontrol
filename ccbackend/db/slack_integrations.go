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
	"organization_id",
	"created_at",
	"updated_at",
}

func NewPostgresSlackIntegrationsRepository(db *sqlx.DB, schema string) *PostgresSlackIntegrationsRepository {
	return &PostgresSlackIntegrationsRepository{db: db, schema: schema}
}

func (r *PostgresSlackIntegrationsRepository) CreateSlackIntegration(
	ctx context.Context,
	integration *models.SlackIntegration,
) error {
	insertColumns := []string{
		"id",
		"slack_team_id",
		"slack_auth_token",
		"slack_team_name",
		"organization_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(slackIntegrationsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.slack_integrations (%s) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query, integration.ID, integration.SlackTeamID, integration.SlackAuthToken, integration.SlackTeamName, integration.OrgID).
		StructScan(integration)
	if err != nil {
		return fmt.Errorf("failed to create slack integration: %w", err)
	}

	return nil
}

func (r *PostgresSlackIntegrationsRepository) GetSlackIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID models.OrgID,
) ([]*models.SlackIntegration, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	columnsStr := strings.Join(slackIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.slack_integrations 
		WHERE organization_id = $1 
		ORDER BY created_at DESC`, columnsStr, r.schema)

	var integrations []*models.SlackIntegration
	err := r.db.SelectContext(ctx, &integrations, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integrations by organization ID: %w", err)
	}

	return integrations, nil
}

func (r *PostgresSlackIntegrationsRepository) GetAllSlackIntegrations(
	ctx context.Context,
) ([]*models.SlackIntegration, error) {
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

func (r *PostgresSlackIntegrationsRepository) DeleteSlackIntegrationByID(
	ctx context.Context,
	integrationID string,
	organizationID models.OrgID,
) (bool, error) {
	query := fmt.Sprintf(`DELETE FROM %s.slack_integrations WHERE id = $1 AND organization_id = $2`, r.schema)

	result, err := r.db.ExecContext(ctx, query, integrationID, organizationID)
	if err != nil {
		return false, fmt.Errorf("failed to delete slack integration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *PostgresSlackIntegrationsRepository) GetSlackIntegrationByTeamID(
	ctx context.Context,
	teamID string,
) (mo.Option[*models.SlackIntegration], error) {
	if teamID == "" {
		return mo.None[*models.SlackIntegration](), fmt.Errorf("team ID cannot be empty")
	}

	columnsStr := strings.Join(slackIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.slack_integrations 
		WHERE slack_team_id = $1`, columnsStr, r.schema)

	var integration models.SlackIntegration
	err := r.db.GetContext(ctx, &integration, query, teamID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.SlackIntegration](), nil
		}
		return mo.None[*models.SlackIntegration](), fmt.Errorf("failed to get slack integration by team ID: %w", err)
	}

	return mo.Some(&integration), nil
}

func (r *PostgresSlackIntegrationsRepository) GetSlackIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.SlackIntegration], error) {
	columnsStr := strings.Join(slackIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.slack_integrations 
		WHERE id = $1`, columnsStr, r.schema)

	var integration models.SlackIntegration
	err := r.db.GetContext(ctx, &integration, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.SlackIntegration](), nil
		}
		return mo.None[*models.SlackIntegration](), fmt.Errorf("failed to get slack integration by ID: %w", err)
	}

	return mo.Some(&integration), nil
}
