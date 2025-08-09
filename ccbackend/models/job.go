package models

import (
	"time"
)

type JobType string

const (
	JobTypeSlack JobType = "slack"
)

type Job struct {
	// Common fields
	ID             string    `json:"id"              db:"id"`
	JobType        JobType   `json:"job_type"        db:"job_type"`
	OrganizationID string    `json:"organization_id" db:"organization_id"`
	CreatedAt      time.Time `json:"created_at"      db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"      db:"updated_at"`

	// Polymorphic payload - only one populated based on JobType
	SlackPayload *SlackJobPayload `json:"slack_payload,omitempty"`
}

type SlackJobPayload struct {
	ThreadTS      string `json:"thread_ts"      db:"slack_thread_ts"`
	ChannelID     string `json:"channel_id"     db:"slack_channel_id"`
	UserID        string `json:"user_id"        db:"slack_user_id"`
	IntegrationID string `json:"integration_id" db:"slack_integration_id"`
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
