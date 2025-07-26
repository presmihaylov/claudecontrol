package db

import (
	"fmt"
	"strings"

	"ccbackend/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"
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
	"created_at",
	"updated_at",
}

func NewPostgresSlackIntegrationsRepository(db *sqlx.DB, schema string) *PostgresSlackIntegrationsRepository {
	return &PostgresSlackIntegrationsRepository{db: db, schema: schema}
}

func (r *PostgresSlackIntegrationsRepository) CreateSlackIntegration(integration *models.SlackIntegration) error {
	insertColumns := []string{"id", "slack_team_id", "slack_auth_token", "slack_team_name", "user_id", "created_at", "updated_at"}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(slackIntegrationsColumns, ", ")
	
	query := fmt.Sprintf(`
		INSERT INTO %s.slack_integrations (%s) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowx(query, integration.ID, integration.SlackTeamID, integration.SlackAuthToken, integration.SlackTeamName, integration.UserID).StructScan(integration)
	if err != nil {
		return fmt.Errorf("failed to create slack integration: %w", err)
	}

	return nil
}

func (r *PostgresSlackIntegrationsRepository) GetSlackIntegrationsByUserID(userID uuid.UUID) ([]*models.SlackIntegration, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID cannot be nil")
	}

	columnsStr := strings.Join(slackIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.slack_integrations 
		WHERE user_id = $1 
		ORDER BY created_at DESC`, columnsStr, r.schema)

	var integrations []*models.SlackIntegration
	err := r.db.Select(&integrations, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integrations by user ID: %w", err)
	}

	return integrations, nil
}

func (r *PostgresSlackIntegrationsRepository) DeleteSlackIntegrationByID(integrationID, userID uuid.UUID) error {
	if integrationID == uuid.Nil {
		return fmt.Errorf("integration ID cannot be nil")
	}

	if userID == uuid.Nil {
		return fmt.Errorf("user ID cannot be nil")
	}

	query := fmt.Sprintf(`DELETE FROM %s.slack_integrations WHERE id = $1 AND user_id = $2`, r.schema)
	
	result, err := r.db.Exec(query, integrationID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete slack integration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("slack integration not found or does not belong to user")
	}

	return nil
}