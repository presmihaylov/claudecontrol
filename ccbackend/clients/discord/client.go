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

	"github.com/bwmarrin/discordgo"
)

var (
	discordOAuthURL = "https://discord.com/api/oauth2/token"
)

// DiscordClient implements the clients.DiscordClient interface using the Discord Go SDK
type DiscordClient struct{}

// NewDiscordClient creates a new Discord client for OAuth operations
func NewDiscordClient() clients.DiscordClient {
	return &DiscordClient{}
}

// ExchangeCodeForToken exchanges an OAuth authorization code for access tokens
// Note: OAuth token exchange still uses HTTP directly as discordgo doesn't provide this
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

// GetGuildInfo fetches guild information using the Discord SDK with OAuth2 Bearer token
func (c *DiscordClient) GetGuildInfo(
	httpClient *http.Client,
	accessToken string,
) ([]*clients.DiscordGuild, error) {
	// Create a Discord session using the OAuth2 Bearer token
	// For OAuth2 tokens from user auth flow, we use Bearer token without "Bot " prefix
	session, err := discordgo.New("Bearer " + accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Use the provided HTTP client if available
	if httpClient != nil {
		session.Client = httpClient
	}

	// Get user's guilds using the SDK
	// Parameters: limit, before, after, withCounts, options...
	discordGuilds, err := session.UserGuilds(100, "", "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user guilds: %w", err)
	}

	// Convert discordgo guilds to our client guild type
	guilds := make([]*clients.DiscordGuild, 0, len(discordGuilds))
	for _, dg := range discordGuilds {
		guilds = append(guilds, &clients.DiscordGuild{
			ID:   dg.ID,
			Name: dg.Name,
		})
	}

	return guilds, nil
}

// GetGuildByID fetches specific guild information using the Discord SDK
// Note: This requires a Bot token with access to the guild
func (c *DiscordClient) GetGuildByID(
	httpClient *http.Client,
	accessToken string,
	guildID string,
) (*clients.DiscordGuild, error) {
	// Create a Discord session using Bot token
	// This method requires Bot token as user tokens cannot directly fetch guild details
	session, err := discordgo.New("Bot " + accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Use the provided HTTP client if available
	if httpClient != nil {
		session.Client = httpClient
	}

	// Get guild information using the SDK
	discordGuild, err := session.Guild(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guild by ID: %w", err)
	}

	// Convert discordgo guild to our client guild type
	guild := &clients.DiscordGuild{
		ID:   discordGuild.ID,
		Name: discordGuild.Name,
	}

	return guild, nil
}
