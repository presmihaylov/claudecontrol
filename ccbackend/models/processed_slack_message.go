package models

import (
	"time"

	"github.com/google/uuid"
)

type ProcessedSlackMessageStatus string

const (
	ProcessedSlackMessageStatusQueued     ProcessedSlackMessageStatus = "QUEUED"
	ProcessedSlackMessageStatusInProgress ProcessedSlackMessageStatus = "IN_PROGRESS"
	ProcessedSlackMessageStatusCompleted  ProcessedSlackMessageStatus = "COMPLETED"
)

type ProcessedSlackMessage struct {
	ID                 uuid.UUID                   `json:"id" db:"id"`
	JobID              uuid.UUID                   `json:"job_id" db:"job_id"`
	SlackChannelID     string                      `json:"slack_channel_id" db:"slack_channel_id"`
	SlackTS            string                      `json:"slack_ts" db:"slack_ts"`
	TextContent        string                      `json:"text_content" db:"text_content"`
	Status             ProcessedSlackMessageStatus `json:"status" db:"status"`
	SlackIntegrationID uuid.UUID                   `json:"slack_integration_id" db:"slack_integration_id"`
	CreatedAt          time.Time                   `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at" db:"updated_at"`
}
