package api

import (
	"time"

	"ccbackend/models"
)

// SlackIntegration represents the slack integration data returned by the API
type SlackIntegration struct {
	ID            string       `json:"id"`
	SlackTeamID   string       `json:"slack_team_id"`
	SlackTeamName string       `json:"slack_team_name"`
	OrgID         models.OrgID `json:"organization_id"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}
