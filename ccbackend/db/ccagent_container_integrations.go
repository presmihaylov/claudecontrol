package db

import (
	"context"
	"database/sql"
	"fmt"

	"ccbackend/models"

	"github.com/jmoiron/sqlx"
	"github.com/samber/mo"
)

// PostgresCCAgentContainerIntegrationsRepository handles database operations for CCAgent container integrations
type PostgresCCAgentContainerIntegrationsRepository struct {
	db     *sqlx.DB
	schema string
}

// NewPostgresCCAgentContainerIntegrationsRepository creates a new repository instance
func NewPostgresCCAgentContainerIntegrationsRepository(
	db *sqlx.DB,
	schema string,
) *PostgresCCAgentContainerIntegrationsRepository {
	return &PostgresCCAgentContainerIntegrationsRepository{
		db:     db,
		schema: schema,
	}
}

// CreateCCAgentContainerIntegration creates a new CCAgent container integration
func (r *PostgresCCAgentContainerIntegrationsRepository) CreateCCAgentContainerIntegration(
	ctx context.Context,
	integration *models.CCAgentContainerIntegration,
) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.ccagent_container_integrations (id, instances_count, repo_url, organization_id)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`, r.schema)

	err := r.db.QueryRowxContext(ctx, query,
		integration.ID,
		integration.InstancesCount,
		integration.RepoURL,
		integration.OrgID,
	).Scan(&integration.CreatedAt, &integration.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create CCAgent container integration: %w", err)
	}

	return nil
}

// ListCCAgentContainerIntegrations retrieves all CCAgent container integrations for an organization
func (r *PostgresCCAgentContainerIntegrationsRepository) ListCCAgentContainerIntegrations(
	ctx context.Context,
	orgID string,
) ([]models.CCAgentContainerIntegration, error) {
	integrations := []models.CCAgentContainerIntegration{}
	query := fmt.Sprintf(`
		SELECT id, instances_count, repo_url, organization_id, created_at, updated_at
		FROM %s.ccagent_container_integrations
		WHERE organization_id = $1
		ORDER BY created_at DESC`, r.schema)

	err := r.db.SelectContext(ctx, &integrations, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list CCAgent container integrations: %w", err)
	}

	return integrations, nil
}

// GetCCAgentContainerIntegrationByID retrieves a CCAgent container integration by ID
func (r *PostgresCCAgentContainerIntegrationsRepository) GetCCAgentContainerIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.CCAgentContainerIntegration], error) {
	var integration models.CCAgentContainerIntegration
	query := fmt.Sprintf(`
		SELECT id, instances_count, repo_url, organization_id, created_at, updated_at
		FROM %s.ccagent_container_integrations
		WHERE id = $1`, r.schema)

	err := r.db.GetContext(ctx, &integration, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.CCAgentContainerIntegration](), nil
		}
		return mo.None[*models.CCAgentContainerIntegration](), fmt.Errorf(
			"failed to get CCAgent container integration: %w",
			err,
		)
	}

	return mo.Some(&integration), nil
}

// DeleteCCAgentContainerIntegration deletes a CCAgent container integration
func (r *PostgresCCAgentContainerIntegrationsRepository) DeleteCCAgentContainerIntegration(
	ctx context.Context,
	id string,
) error {
	query := fmt.Sprintf("DELETE FROM %s.ccagent_container_integrations WHERE id = $1", r.schema)

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete CCAgent container integration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("CCAgent container integration not found")
	}

	return nil
}

// ListCCAgentContainerIntegrationsByOrgIDs retrieves CCAgent container integrations for multiple organizations
func (r *PostgresCCAgentContainerIntegrationsRepository) ListCCAgentContainerIntegrationsByOrgIDs(
	ctx context.Context,
	orgIDs []string,
) ([]*models.CCAgentContainerIntegration, error) {
	if len(orgIDs) == 0 {
		return []*models.CCAgentContainerIntegration{}, nil
	}

	query := fmt.Sprintf(`
		SELECT id, instances_count, repo_url, organization_id, created_at, updated_at
		FROM %s.ccagent_container_integrations
		WHERE organization_id = ANY($1)
		ORDER BY created_at DESC`, r.schema)

	integrations := []*models.CCAgentContainerIntegration{}
	err := r.db.SelectContext(ctx, &integrations, query, orgIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list CCAgent container integrations: %w", err)
	}

	return integrations, nil
}
