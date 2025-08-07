package models

import (
	"time"
)

type Job struct {
	ID                 string    `json:"id"                   db:"id"`
	CreatedAt          time.Time `json:"created_at"           db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"           db:"updated_at"`
	SlackThreadTS      string    `json:"slack_thread_ts"      db:"slack_thread_ts"`
	SlackChannelID     string    `json:"slack_channel_id"     db:"slack_channel_id"`
	SlackUserID        string    `json:"slack_user_id"        db:"slack_user_id"`
	SlackIntegrationID string    `json:"slack_integration_id" db:"slack_integration_id"`
}

type JobCreationStatus string

const (
	JobCreationStatusCreated JobCreationStatus = "CREATED"
	JobCreationStatusNA      JobCreationStatus = "NA"
)

type JobCreationResult struct {
	Job    *Job              `json:"job"`
	Status JobCreationStatus `json:"status"`
}
