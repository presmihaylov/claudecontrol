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
}

// NewDiscordClient creates a new Discord client for OAuth operations
func NewDiscordClient() clients.DiscordClient {
	return &DiscordClient{
		httpClient: &http.Client{},
	}
}

// ExchangeCodeForToken exchanges an OAuth authorization code for access tokens
// Note: This still uses HTTP directly as discordgo doesn't support OAuth2 token exchange
func (c *DiscordClient) ExchangeCodeForToken(
	httpClient *http.Client,
	clientID, clientSecret, code, redirectURL string,
) (*clients.DiscordOAuthResponse, error) {
	// Override httpClient if provided, otherwise use default
	client := c.httpClient
	if httpClient != nil {
		client = httpClient
	}

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

	resp, err := client.Do(req)
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
	// Create a new Discord session using the OAuth2 access token
	session, err := discordgo.New("Bearer " + accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Override HTTP client if provided
	if httpClient != nil {
		session.Client = httpClient
	}

	// Fetch user guilds using discordgo
	// Parameters: limit, beforeID, afterID, withCounts, guildOwnedByCurrentUser
	discordGuilds, err := session.UserGuilds(100, "", "", false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guilds: %w", err)
	}

	// Convert discordgo guilds to our client interface format
	guilds := make([]*clients.DiscordGuild, 0, len(discordGuilds))
	for _, dg := range discordGuilds {
		guilds = append(guilds, &clients.DiscordGuild{
			ID:   dg.ID,
			Name: dg.Name,
		})
	}

	return guilds, nil
}

// GetGuildByID fetches specific guild information using the access token and guild ID
// Note: For OAuth2 user tokens, this will only work if the user is a member of the guild
func (c *DiscordClient) GetGuildByID(
	httpClient *http.Client,
	accessToken string,
	guildID string,
) (*clients.DiscordGuild, error) {
	// Create a new Discord session using the OAuth2 access token
	session, err := discordgo.New("Bearer " + accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Override HTTP client if provided
	if httpClient != nil {
		session.Client = httpClient
	}

	// Try to get the guild directly using discordgo
	// Note: This may fail if using OAuth2 token without proper permissions
	discordGuild, err := session.Guild(guildID, discordgo.WithContext(context.Background()))
	if err != nil {
		// Fallback: Get all user guilds and filter by ID
		// This is more reliable for OAuth2 user tokens
		userGuilds, fetchErr := session.UserGuilds(100, "", "", false, nil)
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch guild %s: %w (fallback also failed: %v)", guildID, err, fetchErr)
		}

		// Find the specific guild in user's guilds
		for _, ug := range userGuilds {
			if ug.ID == guildID {
				return &clients.DiscordGuild{
					ID:   ug.ID,
					Name: ug.Name,
				}, nil
			}
		}

		return nil, fmt.Errorf("guild %s not found in user's guilds", guildID)
	}

	// Convert discordgo guild to our client interface format
	return &clients.DiscordGuild{
		ID:   discordGuild.ID,
		Name: discordGuild.Name,
	}, nil
}
