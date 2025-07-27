package db

import (
	"database/sql"
	"fmt"

	"ccbackend/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"
)

type PostgresAgentsRepository struct {
	db     *sqlx.DB
	schema string
}

func NewPostgresAgentsRepository(db *sqlx.DB, schema string) *PostgresAgentsRepository {
	return &PostgresAgentsRepository{db: db, schema: schema}
}

func (r *PostgresAgentsRepository) CreateActiveAgent(agent *models.ActiveAgent) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.active_agents (id, assigned_job_id, ws_connection_id, slack_integration_id, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, NOW(), NOW()) 
		RETURNING id, assigned_job_id, ws_connection_id, slack_integration_id, created_at, updated_at`, r.schema)

	err := r.db.QueryRowx(query, agent.ID, agent.AssignedJobID, agent.WSConnectionID, agent.SlackIntegrationID).StructScan(agent)
	if err != nil {
		return fmt.Errorf("failed to create active agent: %w", err)
	}

	return nil
}

func (r *PostgresAgentsRepository) DeleteActiveAgent(id uuid.UUID, slackIntegrationID string) error {
	query := fmt.Sprintf("DELETE FROM %s.active_agents WHERE id = $1 AND slack_integration_id = $2", r.schema)

	result, err := r.db.Exec(query, id, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to delete active agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("active agent with id %s not found", id)
	}

	return nil
}

func (r *PostgresAgentsRepository) GetAgentByID(id uuid.UUID, slackIntegrationID string) (*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, assigned_job_id, ws_connection_id, slack_integration_id, created_at, updated_at 
		FROM %s.active_agents 
		WHERE id = $1 AND slack_integration_id = $2`, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.Get(agent, query, id, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("active agent with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get active agent: %w", err)
	}

	return agent, nil
}

func (r *PostgresAgentsRepository) GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID string) (*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, assigned_job_id, ws_connection_id, slack_integration_id, created_at, updated_at 
		FROM %s.active_agents 
		WHERE ws_connection_id = $1 AND slack_integration_id = $2`, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.Get(agent, query, wsConnectionID, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("active agent with ws_connection_id %s not found", wsConnectionID)
		}
		return nil, fmt.Errorf("failed to get active agent: %w", err)
	}

	return agent, nil
}

func (r *PostgresAgentsRepository) GetAvailableAgents(slackIntegrationID string) ([]*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, assigned_job_id, ws_connection_id, slack_integration_id, created_at, updated_at 
		FROM %s.active_agents 
		WHERE assigned_job_id IS NULL AND slack_integration_id = $1
		ORDER BY created_at ASC`, r.schema)

	var agents []*models.ActiveAgent
	err := r.db.Select(&agents, query, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	return agents, nil
}

func (r *PostgresAgentsRepository) DeleteAllActiveAgents() error {
	query := fmt.Sprintf("DELETE FROM %s.active_agents", r.schema)

	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete all active agents: %w", err)
	}

	return nil
}

func (r *PostgresAgentsRepository) GetAgentByJobID(jobID uuid.UUID, slackIntegrationID string) (*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, assigned_job_id, ws_connection_id, slack_integration_id, created_at, updated_at 
		FROM %s.active_agents 
		WHERE assigned_job_id = $1 AND slack_integration_id = $2`, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.Get(agent, query, jobID, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("active agent with assigned_job_id %s not found", jobID)
		}
		return nil, fmt.Errorf("failed to get active agent by job ID: %w", err)
	}

	return agent, nil
}

func (r *PostgresAgentsRepository) UpdateAgentJobAssignment(agentID uuid.UUID, jobID *uuid.UUID, slackIntegrationID string) error {
	query := fmt.Sprintf(`
		UPDATE %s.active_agents 
		SET assigned_job_id = $2, updated_at = NOW() 
		WHERE id = $1 AND slack_integration_id = $3`, r.schema)

	result, err := r.db.Exec(query, agentID, jobID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to update agent job assignment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("active agent with id %s not found", agentID)
	}

	return nil
}

func (r *PostgresAgentsRepository) DeleteAllActiveAgentsBySlackIntegrationID(slackIntegrationID uuid.UUID) error {
	query := fmt.Sprintf("DELETE FROM %s.active_agents WHERE slack_integration_id = $1", r.schema)

	result, err := r.db.Exec(query, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to delete active agents by slack integration ID: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	return nil
}


