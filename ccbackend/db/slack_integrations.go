package db

import (
	"fmt"

	"ccbackend/models"

	"github.com/jmoiron/sqlx"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"
)

type PostgresSlackIntegrationsRepository struct {
	db     *sqlx.DB
	schema string
}

func NewPostgresSlackIntegrationsRepository(db *sqlx.DB, schema string) *PostgresSlackIntegrationsRepository {
	return &PostgresSlackIntegrationsRepository{db: db, schema: schema}
}

func (r *PostgresSlackIntegrationsRepository) CreateSlackIntegration(integration *models.SlackIntegration) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.slack_integrations (id, slack_team_id, slack_auth_token, slack_team_name, user_id, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING id, slack_team_id, slack_auth_token, slack_team_name, user_id, created_at, updated_at`, r.schema)

	err := r.db.QueryRowx(query, integration.ID, integration.SlackTeamID, integration.SlackAuthToken, integration.SlackTeamName, integration.UserID).StructScan(integration)
	if err != nil {
		return fmt.Errorf("failed to create slack integration: %w", err)
	}

	return nil
}