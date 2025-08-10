package discord

import (
	"context"
	"fmt"
	"net/http"

	"ccbackend/clients"

	"github.com/bwmarrin/discordgo"
)

// DiscordClient implements the clients.DiscordClient interface
type DiscordClient struct {
	// httpClient is used for HTTP requests
	httpClient *http.Client
	// sdkClient is the discordgo session initialized once for reuse
	sdkClient *discordgo.Session
}

// NewDiscordClient creates a new Discord client
func NewDiscordClient(httpClient *http.Client, botToken string) (clients.DiscordClient, error) {
	// Initialize the Discord SDK client once during construction
	sdkClient, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create discordgo client: %w", err)
	}

	// Use our HTTP client
	sdkClient.Client = httpClient
	return &DiscordClient{
		httpClient: httpClient,
		sdkClient:  sdkClient,
	}, nil
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

// GetBotUser fetches the bot user information via REST API (no WebSocket required)
func (c *DiscordClient) GetBotUser() (*clients.DiscordBotUser, error) {
	// Get the bot user using the REST API endpoint - no WebSocket session required
	botUser, err := c.sdkClient.User("@me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bot user via REST API: %w", err)
	}
	
	// Convert discordgo user to our client interface format
	return &clients.DiscordBotUser{
		ID:       botUser.ID,
		Username: botUser.Username,
		Bot:      botUser.Bot,
	}, nil
}
