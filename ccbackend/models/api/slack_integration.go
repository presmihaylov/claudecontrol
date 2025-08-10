package api

import (
	"time"
	
	"ccbackend/models"
)

// SlackIntegrationModel represents the slack integration data returned by the API
type SlackIntegrationModel struct {
	ID             string    `json:"id"`
	SlackTeamID    string    `json:"slack_team_id"`
	SlackTeamName  string    `json:"slack_team_name"`
	OrganizationID models.OrganizationID `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
