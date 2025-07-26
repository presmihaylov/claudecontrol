package api

import (
	"time"

	"github.com/google/uuid"
)

// SlackIntegrationModel represents the slack integration data returned by the API
type SlackIntegrationModel struct {
	ID            uuid.UUID `json:"id"`
	SlackTeamID   string    `json:"slack_team_id"`
	SlackTeamName string    `json:"slack_team_name"`
	UserID        uuid.UUID `json:"user_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}