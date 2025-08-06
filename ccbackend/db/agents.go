package db

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	"ccbackend/core"
	"ccbackend/models"
)

type PostgresAgentsRepository struct {
	db     *sqlx.DB
	schema string
}

func NewPostgresAgentsRepository(db *sqlx.DB, schema string) *PostgresAgentsRepository {
	return &PostgresAgentsRepository{db: db, schema: schema}
}

func (r *PostgresAgentsRepository) UpsertActiveAgent(agent *models.ActiveAgent) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.active_agents (id, ws_connection_id, slack_integration_id, ccagent_id, created_at, updated_at, last_active_at) 
		VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW()) 
		ON CONFLICT (slack_integration_id, ccagent_id) 
		DO UPDATE SET 
			ws_connection_id = EXCLUDED.ws_connection_id,
			updated_at = NOW(),
			last_active_at = NOW()
		RETURNING id, ws_connection_id, slack_integration_id, ccagent_id, created_at, updated_at, last_active_at`, r.schema)

	err := r.db.QueryRowx(query, agent.ID, agent.WSConnectionID, agent.SlackIntegrationID, agent.CCAgentID).StructScan(agent)
	if err != nil {
		return fmt.Errorf("failed to upsert active agent: %w", err)
	}

	return nil
}

func (r *PostgresAgentsRepository) DeleteActiveAgent(id string, slackIntegrationID string) error {
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
		return core.ErrNotFound
	}

	return nil
}

func (r *PostgresAgentsRepository) GetAgentByID(id string, slackIntegrationID string) (*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, ws_connection_id, slack_integration_id, ccagent_id, created_at, updated_at, last_active_at 
		FROM %s.active_agents 
		WHERE id = $1 AND slack_integration_id = $2`, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.Get(agent, query, id, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get active agent: %w", err)
	}

	return agent, nil
}

func (r *PostgresAgentsRepository) GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID string) (*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, ws_connection_id, slack_integration_id, ccagent_id, created_at, updated_at, last_active_at 
		FROM %s.active_agents 
		WHERE ws_connection_id = $1 AND slack_integration_id = $2`, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.Get(agent, query, wsConnectionID, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get active agent: %w", err)
	}

	return agent, nil
}

func (r *PostgresAgentsRepository) GetAvailableAgents(slackIntegrationID string) ([]*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT a.id, a.ws_connection_id, a.slack_integration_id, a.ccagent_id, a.created_at, a.updated_at, a.last_active_at 
		FROM %s.active_agents a
		LEFT JOIN %s.agent_job_assignments aja ON a.id = aja.ccagent_id
		WHERE aja.ccagent_id IS NULL AND a.slack_integration_id = $1
		ORDER BY a.created_at ASC`, r.schema, r.schema)

	var agents []*models.ActiveAgent
	err := r.db.Select(&agents, query, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	return agents, nil
}

func (r *PostgresAgentsRepository) GetAllActiveAgents(slackIntegrationID string) ([]*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, ws_connection_id, slack_integration_id, ccagent_id, created_at, updated_at, last_active_at 
		FROM %s.active_agents 
		WHERE slack_integration_id = $1
		ORDER BY created_at ASC`, r.schema)

	var agents []*models.ActiveAgent
	err := r.db.Select(&agents, query, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active agents: %w", err)
	}

	return agents, nil
}

func (r *PostgresAgentsRepository) GetAgentByJobID(jobID string, slackIntegrationID string) (*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT a.id, a.ws_connection_id, a.slack_integration_id, a.ccagent_id, a.created_at, a.updated_at, a.last_active_at 
		FROM %s.active_agents a
		JOIN %s.agent_job_assignments aja ON a.id = aja.ccagent_id
		WHERE aja.job_id = $1 AND aja.slack_integration_id = $2
		LIMIT 1`, r.schema, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.Get(agent, query, jobID, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get active agent by job ID: %w", err)
	}

	return agent, nil
}

func (r *PostgresAgentsRepository) AssignAgentToJob(assignment *models.AgentJobAssignment) error {
	// Use ON CONFLICT DO NOTHING to handle duplicate assignments gracefully
	query := fmt.Sprintf(`
		INSERT INTO %s.agent_job_assignments (id, ccagent_id, job_id, slack_integration_id, assigned_at) 
		VALUES ($1, $2, $3, $4, NOW()) 
		ON CONFLICT (ccagent_id, job_id) DO NOTHING
		RETURNING id, ccagent_id, job_id, slack_integration_id, assigned_at`, r.schema)

	err := r.db.QueryRowx(query, assignment.ID, assignment.CCAgentID, assignment.JobID, assignment.SlackIntegrationID).StructScan(assignment)
	if err != nil {
		// Check if it's a no rows error (conflict occurred, nothing was inserted)
		if err == sql.ErrNoRows {
			// Assignment already exists, not an error
			return nil
		}
		return fmt.Errorf("failed to assign agent to job: %w", err)
	}

	return nil
}

func (r *PostgresAgentsRepository) UnassignAgentFromJob(agentID, jobID string, slackIntegrationID string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.agent_job_assignments 
		WHERE ccagent_id = $1 AND job_id = $2 AND slack_integration_id = $3`, r.schema)

	result, err := r.db.Exec(query, agentID, jobID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to unassign agent from job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return core.ErrNotFound
	}

	return nil
}

func (r *PostgresAgentsRepository) GetActiveAgentJobAssignments(agentID string, slackIntegrationID string) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT job_id 
		FROM %s.agent_job_assignments
		WHERE ccagent_id = $1 AND slack_integration_id = $2
		ORDER BY assigned_at ASC`, r.schema)

	var jobIDs []string
	err := r.db.Select(&jobIDs, query, agentID, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active agent job assignments: %w", err)
	}

	return jobIDs, nil
}

func (r *PostgresAgentsRepository) UpdateAgentLastActiveAt(wsConnectionID, slackIntegrationID string) error {
	query := fmt.Sprintf(`
		UPDATE %s.active_agents 
		SET last_active_at = NOW() 
		WHERE ws_connection_id = $1 AND slack_integration_id = $2`, r.schema)

	result, err := r.db.Exec(query, wsConnectionID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to update agent last_active_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return core.ErrNotFound
	}

	return nil
}

func (r *PostgresAgentsRepository) GetInactiveAgents(slackIntegrationID string, inactiveThresholdMinutes int) ([]*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, ws_connection_id, slack_integration_id, ccagent_id, created_at, updated_at, last_active_at 
		FROM %s.active_agents 
		WHERE slack_integration_id = $1 AND last_active_at < NOW() - INTERVAL '%d minutes'
		ORDER BY last_active_at ASC`, r.schema, inactiveThresholdMinutes)

	var agents []*models.ActiveAgent
	err := r.db.Select(&agents, query, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inactive agents: %w", err)
	}

	return agents, nil
}
