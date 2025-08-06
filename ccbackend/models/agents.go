package models

import (
	"time"
)

type ActiveAgent struct {
	ID                 string    `json:"id" db:"id"`
	WSConnectionID     string    `json:"ws_connection_id" db:"ws_connection_id"`
	SlackIntegrationID string    `json:"slack_integration_id" db:"slack_integration_id"`
	CCAgentID          string    `json:"ccagent_id" db:"ccagent_id"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	LastActiveAt       time.Time `json:"last_active_at" db:"last_active_at"`
}

type AgentJobAssignment struct {
	ID                 string    `json:"id" db:"id"`
	CCAgentID          string    `json:"ccagent_id" db:"ccagent_id"`
	JobID              string    `json:"job_id" db:"job_id"`
	SlackIntegrationID string    `json:"slack_integration_id" db:"slack_integration_id"`
	AssignedAt         time.Time `json:"assigned_at" db:"assigned_at"`
}
