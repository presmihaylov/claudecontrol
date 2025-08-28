package models

import (
	"time"
)

type AnthropicIntegration struct {
	ID                            string     `db:"id"                                 json:"id"`
	AnthropicAPIKey               *string    `db:"anthropic_api_key"                  json:"-"`
	ClaudeCodeOAuthToken          *string    `db:"claude_code_oauth_token"            json:"-"`
	ClaudeCodeOAuthRefreshToken   *string    `db:"claude_code_oauth_refresh_token"    json:"-"`
	ClaudeCodeOAuthTokenExpiresAt *time.Time `db:"claude_code_oauth_token_expires_at" json:"-"`
	OrgID                         OrgID      `db:"organization_id"                    json:"organization_id"`
	CreatedAt                     time.Time  `db:"created_at"                         json:"created_at"`
	UpdatedAt                     time.Time  `db:"updated_at"                         json:"updated_at"`
}
