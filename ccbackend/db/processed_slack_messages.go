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
	"organization_id",
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
	db := dbtx.GetTransactional(ctx, r.db)
	insertColumns := []string{
		"id",
		"job_id",
		"slack_channel_id",
		"slack_ts",
		"text_content",
		"status",
		"slack_integration_id",
		"organization_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(processedSlackMessagesColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.processed_slack_messages (%s) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := db.QueryRowxContext(ctx, query, message.ID, message.JobID, message.SlackChannelID, message.SlackTS, message.TextContent, message.Status, message.SlackIntegrationID, message.OrganizationID).
		StructScan(message)
	if err != nil {
		return fmt.Errorf("failed to create processed slack message: %w", err)
	}

	return nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedSlackMessageByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(processedSlackMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_slack_messages 
		WHERE id = $1 AND organization_id = $2`, columnsStr, r.schema)

	message := &models.ProcessedSlackMessage{}
	err := db.GetContext(ctx, message, query, id, organizationID)
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
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	db := dbtx.GetTransactional(ctx, r.db)
	returningStr := strings.Join(processedSlackMessagesColumns, ", ")
	query := fmt.Sprintf(`
		UPDATE %s.processed_slack_messages 
		SET status = $2, updated_at = NOW() 
		WHERE id = $1 AND slack_integration_id = $3 AND organization_id = $4
		RETURNING %s`, r.schema, returningStr)

	message := &models.ProcessedSlackMessage{}
	err := db.QueryRowxContext(ctx, query, id, status, slackIntegrationID, organizationID).StructScan(message)
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
	organizationID string,
) ([]*models.ProcessedSlackMessage, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(processedSlackMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND slack_integration_id = $2 AND organization_id = $3`, columnsStr, r.schema)

	var messages []*models.ProcessedSlackMessage
	err := db.SelectContext(ctx, &messages, query, jobID, slackIntegrationID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed slack messages by job id: %w", err)
	}

	return messages, nil
}

func (r *PostgresProcessedSlackMessagesRepository) DeleteProcessedSlackMessagesByJobID(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) error {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		DELETE FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND slack_integration_id = $2 AND organization_id = $3`, r.schema)

	_, err := db.ExecContext(ctx, query, jobID, slackIntegrationID, organizationID)
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
	organizationID string,
) ([]*models.ProcessedSlackMessage, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(processedSlackMessagesColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND status = $2 AND slack_integration_id = $3 AND organization_id = $4
		ORDER BY slack_ts ASC`, columnsStr, r.schema)

	var messages []*models.ProcessedSlackMessage
	err := db.SelectContext(ctx, &messages, query, jobID, status, slackIntegrationID, organizationID)
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
	organizationID string,
) (bool, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		UPDATE %s.processed_slack_messages 
		SET updated_at = $2 
		WHERE id = $1 AND slack_integration_id = $3 AND organization_id = $4`, r.schema)

	result, err := db.ExecContext(ctx, query, id, updatedAt, slackIntegrationID, organizationID)
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
	organizationID string,
) (int, error) {
	if len(jobIDs) == 0 {
		return 0, nil
	}

	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.processed_slack_messages 
		WHERE job_id = ANY($1) 
		AND status IN ($2, $3) 
		AND slack_integration_id = $4 
		AND organization_id = $5`, r.schema)

	var count int
	err := db.GetContext(ctx, &count, query,
		pq.Array(jobIDs),
		models.ProcessedSlackMessageStatusInProgress,
		models.ProcessedSlackMessageStatusQueued,
		slackIntegrationID,
		organizationID)
	if err != nil {
		return 0, fmt.Errorf("failed to get active message count for jobs: %w", err)
	}

	return count, nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetLatestProcessedMessageForJob(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	columnsStr := strings.Join(processedSlackMessagesColumns, ", ")
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND slack_integration_id = $2 AND organization_id = $3
		ORDER BY created_at DESC
		LIMIT 1`, columnsStr, r.schema)

	message := &models.ProcessedSlackMessage{}
	err := db.GetContext(ctx, message, query, jobID, slackIntegrationID, organizationID)
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
