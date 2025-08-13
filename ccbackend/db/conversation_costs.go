package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	"ccbackend/core"
	dbtx "ccbackend/db/tx"
	"ccbackend/models"
)

type PostgresConversationCostRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for conversation_costs table
var conversationCostColumns = []string{
	"id",
	"organization_id",
	"job_id",
	"total_input_tokens",
	"total_output_tokens",
	"estimated_cost_usd",
	"created_at",
	"updated_at",
}

func NewPostgresConversationCostRepository(db *sqlx.DB, schema string) *PostgresConversationCostRepository {
	return &PostgresConversationCostRepository{db: db, schema: schema}
}

func (r *PostgresConversationCostRepository) GetConversationCostByJobID(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
) (*models.ConversationCost, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	returningStr := strings.Join(conversationCostColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.conversation_costs 
		WHERE organization_id = $1 AND job_id = $2`,
		returningStr, r.schema)

	cost := &models.ConversationCost{}
	err := db.QueryRowxContext(ctx, query, organizationID, jobID).StructScan(cost)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil // Cost record not found
		}
		return nil, fmt.Errorf("failed to get conversation cost by job ID: %w", err)
	}

	return cost, nil
}

func (r *PostgresConversationCostRepository) CreateConversationCost(
	ctx context.Context,
	cost *models.ConversationCost,
) error {
	db := dbtx.GetTransactional(ctx, r.db)

	// Generate ULID for new conversation cost record
	if cost.ID == "" {
		cost.ID = core.NewID("cc")
	}

	insertColumns := []string{
		"id",
		"organization_id",
		"job_id",
		"total_input_tokens",
		"total_output_tokens",
		"estimated_cost_usd",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(conversationCostColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.conversation_costs (%s) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := db.QueryRowxContext(ctx, query, 
		cost.ID, cost.OrganizationID, cost.JobID, 
		cost.TotalInputTokens, cost.TotalOutputTokens, cost.EstimatedCostUSD).StructScan(cost)
	if err != nil {
		return fmt.Errorf("failed to create conversation cost: %w", err)
	}

	return nil
}

func (r *PostgresConversationCostRepository) UpdateConversationCost(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
	inputTokens, outputTokens int,
	estimatedCost decimal.Decimal,
) (*models.ConversationCost, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	returningStr := strings.Join(conversationCostColumns, ", ")
	query := fmt.Sprintf(`
		UPDATE %s.conversation_costs 
		SET total_input_tokens = $3, 
			total_output_tokens = $4, 
			estimated_cost_usd = $5,
			updated_at = NOW()
		WHERE organization_id = $1 AND job_id = $2
		RETURNING %s`, r.schema, returningStr)

	cost := &models.ConversationCost{}
	err := db.QueryRowxContext(ctx, query, organizationID, jobID, inputTokens, outputTokens, estimatedCost).StructScan(cost)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation cost: %w", err)
	}

	return cost, nil
}

func (r *PostgresConversationCostRepository) GetConversationCostsByOrganizationID(
	ctx context.Context,
	organizationID models.OrgID,
) ([]*models.ConversationCost, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	returningStr := strings.Join(conversationCostColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.conversation_costs 
		WHERE organization_id = $1 
		ORDER BY created_at DESC`,
		returningStr, r.schema)

	var costs []*models.ConversationCost
	err := db.SelectContext(ctx, &costs, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation costs by organization ID: %w", err)
	}

	return costs, nil
}

func (r *PostgresConversationCostRepository) DeleteConversationCostByJobID(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
) error {
	db := dbtx.GetTransactional(ctx, r.db)

	query := fmt.Sprintf(`
		DELETE FROM %s.conversation_costs 
		WHERE organization_id = $1 AND job_id = $2`, r.schema)

	_, err := db.ExecContext(ctx, query, organizationID, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation cost: %w", err)
	}

	return nil
}