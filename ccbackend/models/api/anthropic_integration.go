package api

import (
	"time"
)

// AnthropicIntegration represents the Anthropic integration data returned by the API
type AnthropicIntegration struct {
	ID                string    `json:"id"`
	HasAPIKey         bool      `json:"has_api_key"`
	HasOAuthToken     bool      `json:"has_oauth_token"`
	HasOAuthTokens    bool      `json:"has_oauth_tokens"`
	OAuthTokenExpired bool      `json:"oauth_token_expired,omitempty"`
	OrgID             string    `json:"organization_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
