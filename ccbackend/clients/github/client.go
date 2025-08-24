package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"ccbackend/clients"
	"ccbackend/models"
)

// GitHubClient implements the clients.GitHubClient interface
type GitHubClient struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	jwtClient    *githubJWTClient
}

// OAuth token response
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// NewGitHubClient creates a new GitHub client with the provided configuration
func NewGitHubClient(clientID, clientSecret, appID string, privateKey []byte) (clients.GitHubClient, error) {
	jwtClient, err := newGitHubJWTClient(appID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT client: %w", err)
	}

	return &GitHubClient{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
		jwtClient:    jwtClient,
	}, nil
}

// ExchangeCodeForAccessToken exchanges an OAuth authorization code for an access token
func (c *GitHubClient) ExchangeCodeForAccessToken(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"code":          {code},
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://github.com/login/oauth/access_token",
		bytes.NewBufferString(data.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

// UninstallApp uninstalls a GitHub App installation
func (c *GitHubClient) UninstallApp(ctx context.Context, installationID string) error {
	// Get JWT token (automatically handles caching)
	jwtToken, err := c.jwtClient.getToken()
	if err != nil {
		return fmt.Errorf("failed to get JWT: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%s", installationID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to uninstall app: %w", err)
	}
	defer resp.Body.Close()

	// 204 No Content is success, 404 means already uninstalled
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to uninstall app: status %d, body: %s", resp.StatusCode, string(body))
}

// ListInstallationRepositories lists repositories accessible by a GitHub App installation
func (c *GitHubClient) ListInstallationRepositories(ctx context.Context, installationID string) ([]models.GitHubRepository, error) {
	// Get JWT token for app authentication
	jwtToken, err := c.jwtClient.getToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get JWT: %w", err)
	}

	// First, get an installation access token
	tokenURL := fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", installationID)
	tokenReq, err := http.NewRequestWithContext(ctx, "POST", tokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	tokenReq.Header.Set("Accept", "application/vnd.github+json")
	tokenReq.Header.Set("Authorization", "Bearer "+jwtToken)
	tokenReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	tokenResp, err := c.httpClient.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation token: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(tokenResp.Body)
		return nil, fmt.Errorf("failed to get installation token: status %d, body: %s", tokenResp.StatusCode, string(body))
	}

	var installationToken struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&installationToken); err != nil {
		return nil, fmt.Errorf("failed to decode installation token: %w", err)
	}

	// Now list repositories accessible by the installation
	reposURL := "https://api.github.com/installation/repositories?per_page=100"
	reposReq, err := http.NewRequestWithContext(ctx, "GET", reposURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create repos request: %w", err)
	}

	reposReq.Header.Set("Accept", "application/vnd.github+json")
	reposReq.Header.Set("Authorization", "Bearer "+installationToken.Token)
	reposReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	reposResp, err := c.httpClient.Do(reposReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer reposResp.Body.Close()

	if reposResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(reposResp.Body)
		return nil, fmt.Errorf("failed to list repositories: status %d, body: %s", reposResp.StatusCode, string(body))
	}

	var reposData struct {
		Repositories []models.GitHubRepository `json:"repositories"`
	}
	if err := json.NewDecoder(reposResp.Body).Decode(&reposData); err != nil {
		return nil, fmt.Errorf("failed to decode repositories: %w", err)
	}

	return reposData.Repositories, nil
}
