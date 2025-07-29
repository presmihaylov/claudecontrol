package models

import (
	"time"

	"github.com/google/uuid"
)

type ActiveAgent struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	WSConnectionID     string    `json:"ws_connection_id" db:"ws_connection_id"`
	SlackIntegrationID uuid.UUID `json:"slack_integration_id" db:"slack_integration_id"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

type AgentJobAssignment struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	AgentID            uuid.UUID `json:"agent_id" db:"agent_id"`
	JobID              uuid.UUID `json:"job_id" db:"job_id"`
	SlackIntegrationID uuid.UUID `json:"slack_integration_id" db:"slack_integration_id"`
	AssignedAt         time.Time `json:"assigned_at" db:"assigned_at"`
}
