package models

import (
	"time"
)

type ActiveAgent struct {
	ID             string    `json:"id"               db:"id"`
	WSConnectionID string    `json:"ws_connection_id" db:"ws_connection_id"`
	OrgID          OrgID     `json:"organization_id"  db:"organization_id"`
	CCAgentID      string    `json:"ccagent_id"       db:"ccagent_id"`
	RepoURL        string    `json:"repo_url" db:"repo_url"`
	LastActiveAt   time.Time `json:"last_active_at"   db:"last_active_at"`
	CreatedAt      time.Time `json:"created_at"       db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"       db:"updated_at"`
}

type AgentJobAssignment struct {
	ID         string    `json:"id"              db:"id"`
	AgentID    string    `json:"agent_id"        db:"agent_id"`
	JobID      string    `json:"job_id"          db:"job_id"`
	OrgID      OrgID     `json:"organization_id" db:"organization_id"`
	AssignedAt time.Time `json:"assigned_at"     db:"assigned_at"`
}
