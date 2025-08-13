package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	"ccbackend/core"
	dbtx "ccbackend/db/tx"
	"ccbackend/models"
)

type PostgresConversationContextRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for conversation_context table
var conversationContextColumns = []string{
	"id",
	"organization_id",
	"job_id",
	"full_context",
	"summarized_context",
	"context_size_tokens",
	"is_active",
	"created_at",
	"updated_at",
}

func NewPostgresConversationContextRepository(db *sqlx.DB, schema string) *PostgresConversationContextRepository {
	return &PostgresConversationContextRepository{db: db, schema: schema}
}

func (r *PostgresConversationContextRepository) GetConversationContextByJobID(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
) (*models.ConversationContext, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	returningStr := strings.Join(conversationContextColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.conversation_context 
		WHERE organization_id = $1 AND job_id = $2 AND is_active = true`,
		returningStr, r.schema)

	context := &models.ConversationContext{}
	err := db.QueryRowxContext(ctx, query, organizationID, jobID).StructScan(context)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil // Context record not found
		}
		return nil, fmt.Errorf("failed to get conversation context by job ID: %w", err)
	}

	return context, nil
}

func (r *PostgresConversationContextRepository) CreateConversationContext(
	ctx context.Context,
	context *models.ConversationContext,
) error {
	db := dbtx.GetTransactional(ctx, r.db)

	// Generate ULID for new conversation context record
	if context.ID == "" {
		context.ID = core.NewID("ctx")
	}

	insertColumns := []string{
		"id",
		"organization_id",
		"job_id",
		"full_context",
		"summarized_context",
		"context_size_tokens",
		"is_active",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(conversationContextColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.conversation_context (%s) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := db.QueryRowxContext(ctx, query, 
		context.ID, context.OrganizationID, context.JobID, 
		context.FullContext, context.SummarizedContext, context.ContextSizeTokens, context.IsActive).StructScan(context)
	if err != nil {
		return fmt.Errorf("failed to create conversation context: %w", err)
	}

	return nil
}

func (r *PostgresConversationContextRepository) UpdateConversationContext(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
	fullContext string,
	contextSizeTokens int,
) (*models.ConversationContext, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	returningStr := strings.Join(conversationContextColumns, ", ")
	query := fmt.Sprintf(`
		UPDATE %s.conversation_context 
		SET full_context = $3, 
			context_size_tokens = $4,
			updated_at = NOW()
		WHERE organization_id = $1 AND job_id = $2 AND is_active = true
		RETURNING %s`, r.schema, returningStr)

	context := &models.ConversationContext{}
	err := db.QueryRowxContext(ctx, query, organizationID, jobID, fullContext, contextSizeTokens).StructScan(context)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation context: %w", err)
	}

	return context, nil
}

func (r *PostgresConversationContextRepository) UpdateConversationContextWithSummary(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
	fullContext string,
	summarizedContext string,
	contextSizeTokens int,
) (*models.ConversationContext, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	returningStr := strings.Join(conversationContextColumns, ", ")
	query := fmt.Sprintf(`
		UPDATE %s.conversation_context 
		SET full_context = $3, 
			summarized_context = $4,
			context_size_tokens = $5,
			updated_at = NOW()
		WHERE organization_id = $1 AND job_id = $2 AND is_active = true
		RETURNING %s`, r.schema, returningStr)

	context := &models.ConversationContext{}
	err := db.QueryRowxContext(ctx, query, organizationID, jobID, fullContext, summarizedContext, contextSizeTokens).StructScan(context)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation context with summary: %w", err)
	}

	return context, nil
}

func (r *PostgresConversationContextRepository) DeleteConversationContextByJobID(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
) error {
	db := dbtx.GetTransactional(ctx, r.db)

	query := fmt.Sprintf(`
		UPDATE %s.conversation_context 
		SET is_active = false, updated_at = NOW()
		WHERE organization_id = $1 AND job_id = $2`, r.schema)

	_, err := db.ExecContext(ctx, query, organizationID, jobID)
	if err != nil {
		return fmt.Errorf("failed to deactivate conversation context: %w", err)
	}

	return nil
}