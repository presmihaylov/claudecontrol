package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/samber/mo"

	dbtx "ccbackend/db/tx"
	"ccbackend/models"
)

type PostgresProcessedDiscordMessagesRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for processed_discord_messages table
var processedDiscordMessagesColumns = []string{
	"id",
	"job_id",
	"discord_message_id",
	"discord_thread_id",
	"text_content",
	"status",
	"discord_integration_id",
	"organization_id",
	"created_at",
	"updated_at",
}

func NewPostgresProcessedDiscordMessagesRepository(
	db *sqlx.DB,
	schema string,
) *PostgresProcessedDiscordMessagesRepository {
	return &PostgresProcessedDiscordMessagesRepository{db: db, schema: schema}
}

func (r *PostgresProcessedDiscordMessagesRepository) CreateProcessedDiscordMessage(
	ctx context.Context,
	message *models.ProcessedDiscordMessage,
) error {
	db := dbtx.GetTransactional(ctx, r.db)
	insertColumns := []string{
		"id",
		"job_id",
		"discord_message_id",
		"discord_thread_id",
		"text_content",
		"status",
		"discord_integration_id",
		"organization_id",
	}

	placeholders := make([]string, len(insertColumns))
	for i := range insertColumns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.processed_discord_messages (%s, created_at, updated_at) 
		VALUES (%s, NOW(), NOW())`,
		r.schema,
		strings.Join(insertColumns, ", "),
		strings.Join(placeholders, ", "))

	_, err := db.ExecContext(
		ctx,
		query,
		message.ID,
		message.JobID,
		message.DiscordMessageID,
		message.DiscordThreadID,
		message.TextContent,
		string(message.Status),
		message.DiscordIntegrationID,
		message.OrganizationID,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique violation
				return fmt.Errorf("processed discord message already exists")
			}
		}
		return fmt.Errorf("failed to create processed discord message: %w", err)
	}

	return nil
}

func (r *PostgresProcessedDiscordMessagesRepository) UpdateProcessedDiscordMessage(
	ctx context.Context,
	id string,
	status models.ProcessedDiscordMessageStatus,
	discordIntegrationID string,
	organizationID string,
) (*models.ProcessedDiscordMessage, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(processedDiscordMessagesColumns, ", ")
	query := fmt.Sprintf(`
		UPDATE %s.processed_discord_messages 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2 AND discord_integration_id = $3 AND organization_id = $4
		RETURNING %s`,
		r.schema, columnsStr)

	var message models.ProcessedDiscordMessage
	err := db.GetContext(ctx, &message, query, string(status), id, discordIntegrationID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("processed discord message not found")
		}
		return nil, fmt.Errorf("failed to update processed discord message: %w", err)
	}

	return &message, nil
}

func (r *PostgresProcessedDiscordMessagesRepository) GetProcessedMessagesByJobIDAndStatus(
	ctx context.Context,
	jobID string,
	status models.ProcessedDiscordMessageStatus,
	discordIntegrationID string,
	organizationID string,
) ([]*models.ProcessedDiscordMessage, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(processedDiscordMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_discord_messages 
		WHERE job_id = $1 AND status = $2 AND discord_integration_id = $3 AND organization_id = $4
		ORDER BY created_at ASC`,
		columnsStr, r.schema)

	var messages []models.ProcessedDiscordMessage
	err := db.SelectContext(ctx, &messages, query, jobID, string(status), discordIntegrationID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed discord messages: %w", err)
	}

	// Convert to slice of pointers
	result := make([]*models.ProcessedDiscordMessage, len(messages))
	for i := range messages {
		result[i] = &messages[i]
	}

	return result, nil
}

func (r *PostgresProcessedDiscordMessagesRepository) GetProcessedDiscordMessageByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.ProcessedDiscordMessage], error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(processedDiscordMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_discord_messages 
		WHERE id = $1 AND organization_id = $2`,
		columnsStr, r.schema)

	var message models.ProcessedDiscordMessage
	err := db.GetContext(ctx, &message, query, id, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ProcessedDiscordMessage](), nil
		}
		return mo.None[*models.ProcessedDiscordMessage](), fmt.Errorf(
			"failed to get processed discord message: %w",
			err,
		)
	}

	return mo.Some(&message), nil
}

func (r *PostgresProcessedDiscordMessagesRepository) GetLatestProcessedMessageForJob(
	ctx context.Context,
	jobID string,
	discordIntegrationID string,
	organizationID string,
) (mo.Option[*models.ProcessedDiscordMessage], error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(processedDiscordMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_discord_messages 
		WHERE job_id = $1 AND discord_integration_id = $2 AND organization_id = $3
		ORDER BY created_at DESC 
		LIMIT 1`,
		columnsStr, r.schema)

	var message models.ProcessedDiscordMessage
	err := db.GetContext(ctx, &message, query, jobID, discordIntegrationID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ProcessedDiscordMessage](), nil
		}
		return mo.None[*models.ProcessedDiscordMessage](), fmt.Errorf(
			"failed to get latest processed discord message: %w",
			err,
		)
	}

	return mo.Some(&message), nil
}

func (r *PostgresProcessedDiscordMessagesRepository) GetActiveMessageCountForJobs(
	ctx context.Context,
	jobIDs []string,
	discordIntegrationID string,
	organizationID string,
) (int, error) {
	if len(jobIDs) == 0 {
		return 0, nil
	}

	db := dbtx.GetTransactional(ctx, r.db)
	placeholders := make([]string, len(jobIDs))
	args := make([]interface{}, len(jobIDs)+2)

	for i, jobID := range jobIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = jobID
	}
	args[len(jobIDs)] = discordIntegrationID
	args[len(jobIDs)+1] = organizationID

	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.processed_discord_messages 
		WHERE job_id IN (%s) 
		AND discord_integration_id = $%d 
		AND organization_id = $%d
		AND status IN ('QUEUED', 'IN_PROGRESS')`,
		r.schema,
		strings.Join(placeholders, ", "),
		len(jobIDs)+1,
		len(jobIDs)+2)

	var count int
	err := db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to get active message count: %w", err)
	}

	return count, nil
}

func (r *PostgresProcessedDiscordMessagesRepository) TESTS_UpdateProcessedDiscordMessageUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	discordIntegrationID string,
	organizationID string,
) (bool, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		UPDATE %s.processed_discord_messages 
		SET updated_at = $1 
		WHERE id = $2 AND discord_integration_id = $3 AND organization_id = $4`,
		r.schema)

	result, err := db.ExecContext(ctx, query, updatedAt, id, discordIntegrationID, organizationID)
	if err != nil {
		return false, fmt.Errorf("failed to update processed discord message updated_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *PostgresProcessedDiscordMessagesRepository) DeleteProcessedDiscordMessagesByJobID(
	ctx context.Context,
	jobID string,
	discordIntegrationID string,
	organizationID string,
) error {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		DELETE FROM %s.processed_discord_messages 
		WHERE job_id = $1 AND discord_integration_id = $2 AND organization_id = $3`,
		r.schema)

	_, err := db.ExecContext(ctx, query, jobID, discordIntegrationID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to delete processed discord messages: %w", err)
	}

	return nil
}
