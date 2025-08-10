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

type PostgresAgentsRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for active_agents table
var activeAgentsColumns = []string{
	"id",
	"ws_connection_id",
	"organization_id",
	"ccagent_id",
	"created_at",
	"updated_at",
	"last_active_at",
}

// Column names for agent_job_assignments table
var agentJobAssignmentsColumns = []string{
	"id",
	"agent_id",
	"job_id",
	"organization_id",
	"assigned_at",
}

func NewPostgresAgentsRepository(db *sqlx.DB, schema string) *PostgresAgentsRepository {
	return &PostgresAgentsRepository{db: db, schema: schema}
}

func (r *PostgresAgentsRepository) UpsertActiveAgent(ctx context.Context, agent *models.ActiveAgent) error {
	insertColumns := []string{
		"id",
		"ws_connection_id",
		"organization_id",
		"ccagent_id",
		"created_at",
		"updated_at",
		"last_active_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(activeAgentsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.active_agents (%s) 
		VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW()) 
		ON CONFLICT (organization_id, ccagent_id) 
		DO UPDATE SET 
			ws_connection_id = EXCLUDED.ws_connection_id,
			updated_at = NOW(),
			last_active_at = NOW()
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query, agent.ID, agent.WSConnectionID, agent.OrganizationID, agent.CCAgentID).
		StructScan(agent)
	if err != nil {
		return fmt.Errorf("failed to upsert active agent: %w", err)
	}

	return nil
}

func (r *PostgresAgentsRepository) DeleteActiveAgent(
	ctx context.Context,
	id string,
	organizationID models.OrganizationID,
) (bool, error) {
	query := fmt.Sprintf("DELETE FROM %s.active_agents WHERE id = $1 AND organization_id = $2", r.schema)

	result, err := r.db.ExecContext(ctx, query, id, organizationID)
	if err != nil {
		return false, fmt.Errorf("failed to delete active agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *PostgresAgentsRepository) GetAgentByID(
	ctx context.Context,
	id string,
	organizationID models.OrganizationID,
) (mo.Option[*models.ActiveAgent], error) {
	columnsStr := strings.Join(activeAgentsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.active_agents 
		WHERE id = $1 AND organization_id = $2`, columnsStr, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.GetContext(ctx, agent, query, id, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ActiveAgent](), nil
		}
		return mo.None[*models.ActiveAgent](), fmt.Errorf("failed to get active agent: %w", err)
	}

	return mo.Some(agent), nil
}

func (r *PostgresAgentsRepository) GetAgentByWSConnectionID(
	ctx context.Context,
	wsConnectionID string,
	organizationID models.OrganizationID,
) (mo.Option[*models.ActiveAgent], error) {
	columnsStr := strings.Join(activeAgentsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.active_agents 
		WHERE ws_connection_id = $1 AND organization_id = $2`, columnsStr, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.GetContext(ctx, agent, query, wsConnectionID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ActiveAgent](), nil
		}
		return mo.None[*models.ActiveAgent](), fmt.Errorf("failed to get active agent: %w", err)
	}

	return mo.Some(agent), nil
}

func (r *PostgresAgentsRepository) GetAvailableAgents(
	ctx context.Context,
	organizationID models.OrganizationID,
) ([]*models.ActiveAgent, error) {
	// Build column list with a. prefix for table alias
	var aliasedColumns []string
	for _, col := range activeAgentsColumns {
		aliasedColumns = append(aliasedColumns, "a."+col)
	}
	columnsStr := strings.Join(aliasedColumns, ", ")

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.active_agents a
		LEFT JOIN %s.agent_job_assignments aja ON a.id = aja.agent_id
		WHERE aja.agent_id IS NULL AND a.organization_id = $1
		ORDER BY a.created_at ASC`, columnsStr, r.schema, r.schema)

	var agents []*models.ActiveAgent
	err := r.db.SelectContext(ctx, &agents, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	return agents, nil
}

func (r *PostgresAgentsRepository) GetAllActiveAgents(
	ctx context.Context,
	organizationID models.OrganizationID,
) ([]*models.ActiveAgent, error) {
	columnsStr := strings.Join(activeAgentsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.active_agents 
		WHERE organization_id = $1
		ORDER BY created_at ASC`, columnsStr, r.schema)

	var agents []*models.ActiveAgent
	err := r.db.SelectContext(ctx, &agents, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active agents: %w", err)
	}

	return agents, nil
}

func (r *PostgresAgentsRepository) GetAgentByJobID(
	ctx context.Context,
	jobID string,
	organizationID models.OrganizationID,
) (mo.Option[*models.ActiveAgent], error) {
	// Build column list with a. prefix for table alias
	var aliasedColumns []string
	for _, col := range activeAgentsColumns {
		aliasedColumns = append(aliasedColumns, "a."+col)
	}
	columnsStr := strings.Join(aliasedColumns, ", ")

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.active_agents a
		JOIN %s.agent_job_assignments aja ON a.id = aja.agent_id
		WHERE aja.job_id = $1 AND aja.organization_id = $2
		LIMIT 1`, columnsStr, r.schema, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.GetContext(ctx, agent, query, jobID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ActiveAgent](), nil
		}
		return mo.None[*models.ActiveAgent](), fmt.Errorf("failed to get active agent by job ID: %w", err)
	}

	return mo.Some(agent), nil
}

func (r *PostgresAgentsRepository) AssignAgentToJob(ctx context.Context, assignment *models.AgentJobAssignment) error {
	// Use ON CONFLICT DO NOTHING to handle duplicate assignments gracefully
	insertColumns := []string{"id", "agent_id", "job_id", "organization_id", "assigned_at"}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(agentJobAssignmentsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.agent_job_assignments (%s) 
		VALUES ($1, $2, $3, $4, NOW()) 
		ON CONFLICT (agent_id, job_id) DO NOTHING
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query, assignment.ID, assignment.AgentID, assignment.JobID, assignment.OrganizationID).
		StructScan(assignment)
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

func (r *PostgresAgentsRepository) UnassignAgentFromJob(
	ctx context.Context,
	agentID, jobID string,
	organizationID models.OrganizationID,
) (bool, error) {
	query := fmt.Sprintf(`
		DELETE FROM %s.agent_job_assignments 
		WHERE agent_id = $1 AND job_id = $2 AND organization_id = $3`, r.schema)

	result, err := r.db.ExecContext(ctx, query, agentID, jobID, organizationID)
	if err != nil {
		return false, fmt.Errorf("failed to unassign agent from job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *PostgresAgentsRepository) GetActiveAgentJobAssignments(
	ctx context.Context,
	agentID string,
	organizationID models.OrganizationID,
) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT job_id 
		FROM %s.agent_job_assignments
		WHERE agent_id = $1 AND organization_id = $2
		ORDER BY assigned_at ASC`, r.schema)

	var jobIDs []string
	err := r.db.SelectContext(ctx, &jobIDs, query, agentID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active agent job assignments: %w", err)
	}

	return jobIDs, nil
}

func (r *PostgresAgentsRepository) UpdateAgentLastActiveAt(
	ctx context.Context,
	wsConnectionID string,
	organizationID models.OrganizationID,
) (bool, error) {
	query := fmt.Sprintf(`
		UPDATE %s.active_agents 
		SET last_active_at = NOW() 
		WHERE ws_connection_id = $1 AND organization_id = $2`, r.schema)

	result, err := r.db.ExecContext(ctx, query, wsConnectionID, organizationID)
	if err != nil {
		return false, fmt.Errorf("failed to update agent last_active_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *PostgresAgentsRepository) GetInactiveAgents(
	ctx context.Context,
	organizationID models.OrganizationID,
	inactiveThresholdMinutes int,
) ([]*models.ActiveAgent, error) {
	columnsStr := strings.Join(activeAgentsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.active_agents 
		WHERE organization_id = $1 AND last_active_at < NOW() - INTERVAL '%d minutes'
		ORDER BY last_active_at ASC`, columnsStr, r.schema, inactiveThresholdMinutes)

	var agents []*models.ActiveAgent
	err := r.db.SelectContext(ctx, &agents, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inactive agents: %w", err)
	}

	return agents, nil
}
