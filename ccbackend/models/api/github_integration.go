package api

import (
	"time"
)

// GitHubIntegration represents the GitHub integration data returned by the API
type GitHubIntegration struct {
	ID                   string    `json:"id"`
	GitHubInstallationID string    `json:"github_installation_id"`
	OrgID                string    `json:"organization_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}