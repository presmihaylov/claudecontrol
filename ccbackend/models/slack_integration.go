package models

import (
	"time"
)

type SlackIntegration struct {
	ID             string    `db:"id"               json:"id"`
	SlackTeamID    string    `db:"slack_team_id"    json:"slack_team_id"`
	SlackAuthToken string    `db:"slack_auth_token" json:"-"`
	SlackTeamName  string    `db:"slack_team_name"  json:"slack_team_name"`
	OrganizationID string    `db:"organization_id"  json:"organization_id"`
	CreatedAt      time.Time `db:"created_at"       json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"       json:"updated_at"`
}
