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
