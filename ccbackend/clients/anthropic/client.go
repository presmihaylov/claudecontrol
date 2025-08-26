package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ccbackend/clients"
)

// AnthropicClient implements the clients.AnthropicClient interface
type AnthropicClient struct {
	httpClient *http.Client
	clientID   string
}

// TokenResponse represents the OAuth token exchange response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// TokenExchangeRequest represents the token exchange request
type TokenExchangeRequest struct {
	Code         string `json:"code"`
	State        string `json:"state"`
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	CodeVerifier string `json:"code_verifier"`
}

// RefreshTokenRequest represents the refresh token request
type RefreshTokenRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
}

// NewAnthropicClient creates a new Anthropic OAuth client
func NewAnthropicClient() clients.AnthropicClient {
	return &AnthropicClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		clientID:   "9d1c250a-e61b-44d9-88ed-5944d1962f5e",
	}
}

// ExchangeCodeForTokens exchanges an OAuth authorization code for access and refresh tokens
func (c *AnthropicClient) ExchangeCodeForTokens(
	ctx context.Context,
	authCode, codeVerifier string,
) (*clients.AnthropicTokens, error) {
	// Split the authorization code at # to get code and state
	parts := strings.Split(authCode, "#")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid authorization code format, expected code#state")
	}

	code := parts[0]
	state := parts[1]

	reqBody := TokenExchangeRequest{
		Code:         code,
		State:        state,
		GrantType:    "authorization_code",
		ClientID:     c.clientID,
		RedirectURI:  "https://console.anthropic.com/oauth/code/callback",
		CodeVerifier: codeVerifier,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://console.anthropic.com/v1/oauth/token",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		return nil, fmt.Errorf("missing tokens in response")
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return &clients.AnthropicTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// RefreshAccessToken refreshes an expired access token using a refresh token
func (c *AnthropicClient) RefreshAccessToken(
	ctx context.Context,
	refreshToken string,
) (*clients.AnthropicTokens, error) {
	reqBody := RefreshTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     c.clientID,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://console.anthropic.com/v1/oauth/token",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return &clients.AnthropicTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
