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
	discordAPIBase  = "https://discord.com/api"
	discordOAuthURL = discordAPIBase + "/oauth2/token"
	// discordGuildsURL is no longer needed as we use discordgo for guild operations
)

// DiscordClient implements the clients.DiscordClient interface
type DiscordClient struct {
	// httpClient is used for OAuth2 token exchange since discordgo doesn't support it
	httpClient *http.Client
	// botToken is the Discord bot token used for API requests
	botToken string
	// sdkClient is the discordgo session initialized once for reuse
	sdkClient *discordgo.Session
}

// NewDiscordClient creates a new Discord client for OAuth operations
func NewDiscordClient(httpClient *http.Client, botToken string) clients.DiscordClient {
	// Initialize the Discord SDK client once during construction
	sdkClient, err := discordgo.New("Bot " + botToken)
	if err != nil {
		// If we can't create the SDK client, log the error but still return a client
		// The OAuth functionality will still work, but guild operations will fail
		return &DiscordClient{
			httpClient: httpClient,
			botToken:   botToken,
			sdkClient:  nil,
		}
	}

	// Use our HTTP client
	sdkClient.Client = httpClient

	return &DiscordClient{
		httpClient: httpClient,
		botToken:   botToken,
		sdkClient:  sdkClient,
	}
}

// ExchangeCodeForToken exchanges an OAuth authorization code for access tokens
// Note: This still uses HTTP directly as discordgo doesn't support OAuth2 token exchange
func (c *DiscordClient) ExchangeCodeForToken(
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute OAuth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OAuth request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OAuth response body: %w", err)
	}

	var oauthResp clients.DiscordOAuthResponse
	if err := json.Unmarshal(body, &oauthResp); err != nil {
		return nil, fmt.Errorf("failed to decode OAuth response: %w", err)
	}

	return &oauthResp, nil
}

// GetGuildByID fetches specific guild information using the bot token
func (c *DiscordClient) GetGuildByID(guildID string) (*clients.DiscordGuild, error) {
	// Get the guild using the pre-initialized discordgo client
	discordGuild, err := c.sdkClient.Guild(guildID, discordgo.WithContext(context.Background()))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guild: %w", err)
	}
	if discordGuild == nil {
		return nil, fmt.Errorf("guild not found")
	}

	// Convert discordgo guild to our client interface format
	return &clients.DiscordGuild{
		ID:   discordGuild.ID,
		Name: discordGuild.Name,
	}, nil
}
