package api

import (
	"time"
)

// SlackIntegrationModel represents the slack integration data returned by the API
type SlackIntegrationModel struct {
	ID                          string     `json:"id"`
	SlackTeamID                 string     `json:"slack_team_id"`
	SlackTeamName               string     `json:"slack_team_name"`
	OrganizationID              string     `json:"organization_id"`
	CCAgentSecretKeyGeneratedAt *time.Time `json:"ccagent_secret_key_generated_at"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}
