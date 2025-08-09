package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"ccbackend/clients"
)

var (
	discordAPIBase   = "https://discord.com/api"
	discordOAuthURL  = discordAPIBase + "/oauth2/token"
	discordGuildsURL = discordAPIBase + "/users/@me/guilds"
)

// DiscordClient implements the clients.DiscordClient interface
type DiscordClient struct{}

// NewDiscordClient creates a new Discord client for OAuth operations
func NewDiscordClient() clients.DiscordClient {
	return &DiscordClient{}
}

// ExchangeCodeForToken exchanges an OAuth authorization code for access tokens
func (c *DiscordClient) ExchangeCodeForToken(
	httpClient *http.Client,
	clientID, clientSecret, code, redirectURL string,
) (*clients.DiscordOAuthResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURL)

	req, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		discordOAuthURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute OAuth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OAuth request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var oauthResp clients.DiscordOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		return nil, fmt.Errorf("failed to decode OAuth response: %w", err)
	}

	return &oauthResp, nil
}

// GetGuildInfo fetches guild information using the access token
func (c *DiscordClient) GetGuildInfo(
	httpClient *http.Client,
	accessToken string,
) ([]*clients.DiscordGuild, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", discordGuildsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create guilds request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute guilds request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("guilds request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var guilds []*clients.DiscordGuild
	if err := json.NewDecoder(resp.Body).Decode(&guilds); err != nil {
		return nil, fmt.Errorf("failed to decode guilds response: %w", err)
	}

	return guilds, nil
}

// GetGuildByID fetches specific guild information using the access token and guild ID
func (c *DiscordClient) GetGuildByID(
	httpClient *http.Client,
	accessToken string,
	guildID string,
) (*clients.DiscordGuild, error) {
	guildURL := fmt.Sprintf("%s/guilds/%s", discordAPIBase, guildID)

	req, err := http.NewRequestWithContext(context.Background(), "GET", guildURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create guild request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute guild request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("guild request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var guild clients.DiscordGuild
	if err := json.NewDecoder(resp.Body).Decode(&guild); err != nil {
		return nil, fmt.Errorf("failed to decode guild response: %w", err)
	}

	return &guild, nil
}
