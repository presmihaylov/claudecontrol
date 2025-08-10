package models

import (
	"time"
)

type ProcessedDiscordMessageStatus string

const (
	ProcessedDiscordMessageStatusQueued     ProcessedDiscordMessageStatus = "QUEUED"
	ProcessedDiscordMessageStatusInProgress ProcessedDiscordMessageStatus = "IN_PROGRESS"
	ProcessedDiscordMessageStatusCompleted  ProcessedDiscordMessageStatus = "COMPLETED"
	ProcessedDiscordMessageStatusFailed     ProcessedDiscordMessageStatus = "FAILED"
)

type ProcessedDiscordMessage struct {
	ID                   string                        `json:"id"                     db:"id"`
	JobID                string                        `json:"job_id"                 db:"job_id"`
	DiscordMessageID     string                        `json:"discord_message_id"     db:"discord_message_id"`
	DiscordThreadID      string                        `json:"discord_thread_id"      db:"discord_thread_id"`
	TextContent          string                        `json:"text_content"           db:"text_content"`
	Status               ProcessedDiscordMessageStatus `json:"status"                 db:"status"`
	DiscordIntegrationID string                        `json:"discord_integration_id" db:"discord_integration_id"`
	OrganizationID       string                        `json:"organization_id"        db:"organization_id"`
	CreatedAt            time.Time                     `json:"created_at"             db:"created_at"`
	UpdatedAt            time.Time                     `json:"updated_at"             db:"updated_at"`
}
