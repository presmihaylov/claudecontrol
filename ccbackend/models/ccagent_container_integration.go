package models

import (
	"time"
)

// CCAgentContainerIntegration represents a CCAgent container configuration for an organization
type CCAgentContainerIntegration struct {
	ID             string    `db:"id"              json:"id"`
	InstancesCount int       `db:"instances_count" json:"instances_count"`
	RepoURL        string    `db:"repo_url"        json:"repo_url"`
	OrgID          OrgID     `db:"organization_id" json:"organization_id"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"      json:"updated_at"`
}

// CCAgentContainerIntegrationCreateRequest represents the API request to create a CCAgent container integration
type CCAgentContainerIntegrationCreateRequest struct {
	InstancesCount int    `json:"instances_count"`
	RepoURL        string `json:"repo_url"`
}

// CCAgentContainerIntegrationResponse represents the API response for a CCAgent container integration
type CCAgentContainerIntegrationResponse struct {
	ID             string    `json:"id"`
	InstancesCount int       `json:"instances_count"`
	RepoURL        string    `json:"repo_url"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
