package models

import (
	"time"
)

type GitHubIntegration struct {
	ID                   string    `db:"id"                     json:"id"`
	GitHubInstallationID string    `db:"github_installation_id" json:"github_installation_id"`
	GitHubAccessToken    string    `db:"github_access_token"    json:"-"`
	OrgID                OrgID     `db:"organization_id"        json:"organization_id"`
	CreatedAt            time.Time `db:"created_at"             json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"             json:"updated_at"`
}

// GitHubRepository represents a GitHub repository accessible by the integration
type GitHubRepository struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description,omitempty"`
	Private     bool   `json:"private"`
}
