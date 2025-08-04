package models

import (
	"time"

	"github.com/google/uuid"
)

type SlackIntegration struct {
	ID                          uuid.UUID  `db:"id" json:"id"`
	SlackTeamID                 string     `db:"slack_team_id" json:"slack_team_id"`
	SlackAuthToken              string     `db:"slack_auth_token" json:"-"`
	SlackTeamName               string     `db:"slack_team_name" json:"slack_team_name"`
	UserID                      uuid.UUID  `db:"user_id" json:"user_id"`
	CCAgentSecretKey            *string    `db:"ccagent_secret_key" json:"-"`
	CCAgentSecretKeyGeneratedAt *time.Time `db:"ccagent_secret_key_generated_at" json:"ccagent_secret_key_generated_at"`
	CreatedAt                   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt                   time.Time  `db:"updated_at" json:"updated_at"`
}
