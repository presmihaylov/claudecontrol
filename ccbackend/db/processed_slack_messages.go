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

type PostgresProcessedSlackMessagesRepository struct {
	db     *sqlx.DB
	schema string
}

func NewPostgresProcessedSlackMessagesRepository(db *sqlx.DB, schema string) *PostgresProcessedSlackMessagesRepository {
	return &PostgresProcessedSlackMessagesRepository{db: db, schema: schema}
}

func (r *PostgresProcessedSlackMessagesRepository) CreateProcessedSlackMessage(message *models.ProcessedSlackMessage) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.processed_slack_messages (id, job_id, slack_channel_id, slack_ts, text_content, status, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) 
		RETURNING id, job_id, slack_channel_id, slack_ts, text_content, status, created_at, updated_at`, r.schema)

	err := r.db.QueryRowx(query, message.ID, message.JobID, message.SlackChannelID, message.SlackTS, message.TextContent, message.Status).StructScan(message)
	if err != nil {
		return fmt.Errorf("failed to create processed slack message: %w", err)
	}

	return nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedSlackMessageByID(id uuid.UUID) (*models.ProcessedSlackMessage, error) {
	query := fmt.Sprintf(`
		SELECT id, job_id, slack_channel_id, slack_ts, text_content, status, created_at, updated_at 
		FROM %s.processed_slack_messages 
		WHERE id = $1`, r.schema)

	message := &models.ProcessedSlackMessage{}
	err := r.db.Get(message, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("processed slack message with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get processed slack message: %w", err)
	}

	return message, nil
}

func (r *PostgresProcessedSlackMessagesRepository) UpdateProcessedSlackMessageStatus(id uuid.UUID, status models.ProcessedSlackMessageStatus) (*models.ProcessedSlackMessage, error) {
	query := fmt.Sprintf(`
		UPDATE %s.processed_slack_messages 
		SET status = $2, updated_at = NOW() 
		WHERE id = $1
		RETURNING id, job_id, slack_channel_id, slack_ts, text_content, status, created_at, updated_at`, r.schema)

	message := &models.ProcessedSlackMessage{}
	err := r.db.QueryRowx(query, id, status).StructScan(message)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("processed slack message with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to update processed slack message status: %w", err)
	}

	return message, nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedSlackMessagesByJobID(jobID uuid.UUID) ([]*models.ProcessedSlackMessage, error) {
	query := fmt.Sprintf(`
		SELECT id, job_id, slack_channel_id, slack_ts, text_content, status, created_at, updated_at 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1`, r.schema)

	var messages []*models.ProcessedSlackMessage
	err := r.db.Select(&messages, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed slack messages by job id: %w", err)
	}

	return messages, nil
}

func (r *PostgresProcessedSlackMessagesRepository) DeleteProcessedSlackMessagesByJobID(jobID uuid.UUID) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.processed_slack_messages 
		WHERE job_id = $1`, r.schema)

	_, err := r.db.Exec(query, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete processed slack messages by job id: %w", err)
	}

	return nil
}

func (r *PostgresProcessedSlackMessagesRepository) GetProcessedMessagesByJobIDAndStatus(jobID uuid.UUID, status models.ProcessedSlackMessageStatus) ([]*models.ProcessedSlackMessage, error) {
	query := fmt.Sprintf(`
		SELECT id, job_id, slack_channel_id, slack_ts, text_content, status, created_at, updated_at 
		FROM %s.processed_slack_messages 
		WHERE job_id = $1 AND status = $2 
		ORDER BY slack_ts ASC`, r.schema)

	var messages []*models.ProcessedSlackMessage
	err := r.db.Select(&messages, query, jobID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed messages by job id and status: %w", err)
	}

	return messages, nil
}