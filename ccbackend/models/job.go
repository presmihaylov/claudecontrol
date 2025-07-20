package models

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID              uuid.UUID `json:"id" db:"id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	SlackThreadTS   string    `json:"slack_thread_ts" db:"slack_thread_ts"`
	SlackChannelID  string    `json:"slack_channel_id" db:"slack_channel_id"`
}