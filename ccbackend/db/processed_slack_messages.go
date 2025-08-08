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

	"ccbackend/models"
)

type PostgresProcessedSlackMessagesRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for processed_slack_messages table
var processedSlackMessagesColumns = []string{
	"id",
	"job_id",
	"slack_channel_id",
	"slack_ts",
	"text_content",
	"status",
	"slack_integration_id",
	"created_at",
	"updated_at",
}

func NewPostgresProcessedSlackMessagesRepository(db *sqlx.DB, schema string) *PostgresProcessedSlackMessagesRepository {
	return &PostgresProcessedSlackMessagesRepository{db: db, schema: schema}
}

func (r *PostgresProcessedSlackMessagesRepository) CreateProcessedSlackMessage(
	ctx context.Context,
	message *models.ProcessedSlackMessage,
) error {
	insertColumns := []string{
		"id",
		"job_id",
		"slack_channel_id",
		"slack_ts",
		"text_content",
		"status",
		"slack_integration_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(processedSlackMessagesColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.processed_slack_messages (%s) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query, message.ID, message.JobID, message.SlackChannelID, message.SlackTS, message.TextContent, message.Status, message.OrganizationID).
		StructScan(message)
	if err != nil {
		return fmt.Errorf("failed to create processed slack message: %w", err)
	}

	return nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedSlackMessageByID(
	ctx context.Context,
	id string,
	slackIntegrationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	columnsStr := strings.Join(processedSlackMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_slack_messages 
		WHERE id = $1 AND slack_integration_id = $2`, columnsStr, r.schema)

	message := &models.ProcessedSlackMessage{}
	err := r.db.GetContext(ctx, message, query, id, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ProcessedSlackMessage](), nil
		}
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("failed to get processed slack message: %w", err)
	}

	return mo.Some(message), nil
}

func (r *PostgresProcessedSlackMessagesRepository) UpdateProcessedSlackMessageStatus(
	ctx context.Context,
	id string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	returningStr := strings.Join(processedSlackMessagesColumns, ", ")
	query := fmt.Sprintf(`
		UPDATE %s.processed_slack_messages 
		SET status = $2, updated_at = NOW() 
		WHERE id = $1 AND slack_integration_id = $3
		RETURNING %s`, r.schema, returningStr)

	message := &models.ProcessedSlackMessage{}
	err := r.db.QueryRowxContext(ctx, query, id, status, slackIntegrationID).StructScan(message)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ProcessedSlackMessage](), nil
		}
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf(
			"failed to update processed slack message status: %w",
			err,
		)
	}

	return mo.Some(message), nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedSlackMessagesByJobID(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
) ([]*models.ProcessedSlackMessage, error) {
	columnsStr := strings.Join(processedSlackMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND slack_integration_id = $2`, columnsStr, r.schema)

	var messages []*models.ProcessedSlackMessage
	err := r.db.SelectContext(ctx, &messages, query, jobID, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed slack messages by job id: %w", err)
	}

	return messages, nil
}

func (r *PostgresProcessedSlackMessagesRepository) DeleteProcessedSlackMessagesByJobID(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND slack_integration_id = $2`, r.schema)

	_, err := r.db.ExecContext(ctx, query, jobID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to delete processed slack messages by job id: %w", err)
	}

	return nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedMessagesByJobIDAndStatus(
	ctx context.Context,
	jobID string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
) ([]*models.ProcessedSlackMessage, error) {
	columnsStr := strings.Join(processedSlackMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND status = $2 AND slack_integration_id = $3
		ORDER BY slack_ts ASC`, columnsStr, r.schema)

	var messages []*models.ProcessedSlackMessage
	err := r.db.SelectContext(ctx, &messages, query, jobID, status, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed messages by job id and status: %w", err)
	}

	return messages, nil
}

// TESTS_UpdateProcessedSlackMessageUpdatedAt updates the updated_at timestamp of a processed slack message for testing purposes
func (r *PostgresProcessedSlackMessagesRepository) TESTS_UpdateProcessedSlackMessageUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
) (bool, error) {
	query := fmt.Sprintf(`
		UPDATE %s.processed_slack_messages 
		SET updated_at = $2 
		WHERE id = $1 AND slack_integration_id = $3`, r.schema)

	result, err := r.db.ExecContext(ctx, query, id, updatedAt, slackIntegrationID)
	if err != nil {
		return false, fmt.Errorf("failed to update processed slack message updated_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetActiveMessageCountForJobs(
	ctx context.Context,
	jobIDs []string,
	slackIntegrationID string,
) (int, error) {
	if len(jobIDs) == 0 {
		return 0, nil
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.processed_slack_messages 
		WHERE job_id = ANY($1) 
		AND status IN ($2, $3) 
		AND slack_integration_id = $4`, r.schema)

	var count int
	err := r.db.GetContext(ctx, &count, query,
		pq.Array(jobIDs),
		models.ProcessedSlackMessageStatusInProgress,
		models.ProcessedSlackMessageStatusQueued,
		slackIntegrationID)
	if err != nil {
		return 0, fmt.Errorf("failed to get active message count for jobs: %w", err)
	}

	return count, nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetLatestProcessedMessageForJob(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	columnsStr := strings.Join(processedSlackMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND slack_integration_id = $2
		ORDER BY created_at DESC
		LIMIT 1`, columnsStr, r.schema)

	message := &models.ProcessedSlackMessage{}
	err := r.db.GetContext(ctx, message, query, jobID, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ProcessedSlackMessage](), nil
		}
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf(
			"failed to get latest processed message for job: %w",
			err,
		)
	}

	return mo.Some(message), nil
}
