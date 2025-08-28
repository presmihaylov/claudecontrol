package models

import (
	"time"
)

// CCAgentContainerIntegration represents a CCAgent container configuration for an organization
type CCAgentContainerIntegration struct {
	ID             string    `db:"id"              json:"id"`
	InstancesCount int       `db:"instances_count" json:"instances_count"`
	RepoURL        string    `db:"repo_url"        json:"repo_url"`
	SSHHost        string    `db:"ssh_host"        json:"ssh_host"`
	OrgID          OrgID     `db:"organization_id" json:"organization_id"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"      json:"updated_at"`
}
