package models

import (
	"time"
)

type AnthropicIntegration struct {
	ID                   string    `db:"id"                      json:"id"`
	AnthropicAPIKey      *string   `db:"anthropic_api_key"       json:"-"`
	ClaudeCodeOAuthToken *string   `db:"claude_code_oauth_token" json:"-"`
	OrgID                OrgID     `db:"organization_id"         json:"organization_id"`
	CreatedAt            time.Time `db:"created_at"              json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"              json:"updated_at"`
}
