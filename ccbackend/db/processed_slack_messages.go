package db

import (
	"database/sql"
	"fmt"
	"time"

	"ccbackend/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgresProcessedSlackMessagesRepository struct {
	db     *sqlx.DB
	schema string
}

func NewPostgresProcessedSlackMessagesRepository(db *sqlx.DB, schema string) *PostgresProcessedSlackMessagesRepository {
	return &PostgresProcessedSlackMessagesRepository{db: db, schema: schema}
}

func (r *PostgresProcessedSlackMessagesRepository) CreateProcessedSlackMessage(message *models.ProcessedSlackMessage) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.processed_slack_messages (id, job_id, slack_channel_id, slack_ts, text_content, status, slack_integration_id, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) 
		RETURNING id, job_id, slack_channel_id, slack_ts, text_content, status, slack_integration_id, created_at, updated_at`, r.schema)

	err := r.db.QueryRowx(query, message.ID, message.JobID, message.SlackChannelID, message.SlackTS, message.TextContent, message.Status, message.SlackIntegrationID).StructScan(message)
	if err != nil {
		return fmt.Errorf("failed to create processed slack message: %w", err)
	}

	return nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedSlackMessageByID(id uuid.UUID, slackIntegrationID string) (*models.ProcessedSlackMessage, error) {
	query := fmt.Sprintf(`
		SELECT id, job_id, slack_channel_id, slack_ts, text_content, status, slack_integration_id, created_at, updated_at 
		FROM %s.processed_slack_messages 
		WHERE id = $1 AND slack_integration_id = $2`, r.schema)

	message := &models.ProcessedSlackMessage{}
	err := r.db.Get(message, query, id, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("processed slack message with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get processed slack message: %w", err)
	}

	return message, nil
}

func (r *PostgresProcessedSlackMessagesRepository) UpdateProcessedSlackMessageStatus(id uuid.UUID, status models.ProcessedSlackMessageStatus, slackIntegrationID string) (*models.ProcessedSlackMessage, error) {
	query := fmt.Sprintf(`
		UPDATE %s.processed_slack_messages 
		SET status = $2, updated_at = NOW() 
		WHERE id = $1 AND slack_integration_id = $3
		RETURNING id, job_id, slack_channel_id, slack_ts, text_content, status, slack_integration_id, created_at, updated_at`, r.schema)

	message := &models.ProcessedSlackMessage{}
	err := r.db.QueryRowx(query, id, status, slackIntegrationID).StructScan(message)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("processed slack message with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to update processed slack message status: %w", err)
	}

	return message, nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedSlackMessagesByJobID(jobID uuid.UUID, slackIntegrationID string) ([]*models.ProcessedSlackMessage, error) {
	query := fmt.Sprintf(`
		SELECT id, job_id, slack_channel_id, slack_ts, text_content, status, slack_integration_id, created_at, updated_at 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND slack_integration_id = $2`, r.schema)

	var messages []*models.ProcessedSlackMessage
	err := r.db.Select(&messages, query, jobID, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed slack messages by job id: %w", err)
	}

	return messages, nil
}

func (r *PostgresProcessedSlackMessagesRepository) DeleteProcessedSlackMessagesByJobID(jobID uuid.UUID, slackIntegrationID string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND slack_integration_id = $2`, r.schema)

	_, err := r.db.Exec(query, jobID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to delete processed slack messages by job id: %w", err)
	}

	return nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedMessagesByJobIDAndStatus(jobID uuid.UUID, status models.ProcessedSlackMessageStatus, slackIntegrationID string) ([]*models.ProcessedSlackMessage, error) {
	query := fmt.Sprintf(`
		SELECT id, job_id, slack_channel_id, slack_ts, text_content, status, slack_integration_id, created_at, updated_at 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND status = $2 AND slack_integration_id = $3
		ORDER BY slack_ts ASC`, r.schema)

	var messages []*models.ProcessedSlackMessage
	err := r.db.Select(&messages, query, jobID, status, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed messages by job id and status: %w", err)
	}

	return messages, nil
}

// TESTS_UpdateProcessedSlackMessageUpdatedAt updates the updated_at timestamp of a processed slack message for testing purposes
func (r *PostgresProcessedSlackMessagesRepository) TESTS_UpdateProcessedSlackMessageUpdatedAt(id uuid.UUID, updatedAt time.Time, slackIntegrationID string) error {
	query := fmt.Sprintf(`
		UPDATE %s.processed_slack_messages 
		SET updated_at = $2 
		WHERE id = $1 AND slack_integration_id = $3`, r.schema)

	result, err := r.db.Exec(query, id, updatedAt, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to update processed slack message updated_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("processed slack message with id %s not found", id)
	}

	return nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetActiveMessageCountForJobs(jobIDs []uuid.UUID, slackIntegrationID string) (int, error) {
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
	err := r.db.Get(&count, query,
		pq.Array(jobIDs),
		models.ProcessedSlackMessageStatusInProgress,
		models.ProcessedSlackMessageStatusQueued,
		slackIntegrationID)
	if err != nil {
		return 0, fmt.Errorf("failed to get active message count for jobs: %w", err)
	}

	return count, nil
}
