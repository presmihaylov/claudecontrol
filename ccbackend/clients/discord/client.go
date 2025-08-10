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

// GetChannelByID fetches channel information by ID
func (c *DiscordClient) GetChannelByID(channelID string) (*clients.DiscordChannel, error) {
	channel, err := c.sdkClient.Channel(channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Discord channel: %w", err)
	}
	if channel == nil {
		return nil, fmt.Errorf("discord channel not found")
	}

	return &clients.DiscordChannel{
		ID:      channel.ID,
		Name:    channel.Name,
		Type:    int(channel.Type),
		GuildID: channel.GuildID,
	}, nil
}

// PostMessage sends a message to a Discord channel or thread
func (c *DiscordClient) PostMessage(
	channelID string,
	params clients.DiscordMessageParams,
) (*clients.DiscordPostMessageResponse, error) {
	targetChannelID := channelID
	if params.ThreadID != nil && *params.ThreadID != "" {
		// For Discord threads, we post to the thread ID (which is actually a channel ID for thread channels)
		targetChannelID = *params.ThreadID
	}

	message, err := c.sdkClient.ChannelMessageSend(targetChannelID, params.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to send Discord message: %w", err)
	}

	return &clients.DiscordPostMessageResponse{
		ChannelID: message.ChannelID,
		MessageID: message.ID,
	}, nil
}

// AddReaction adds a reaction emoji to a Discord message
func (c *DiscordClient) AddReaction(channelID, messageID, emoji string) error {
	err := c.sdkClient.MessageReactionAdd(channelID, messageID, emoji)
	if err != nil {
		return fmt.Errorf("failed to add Discord reaction: %w", err)
	}
	return nil
}

// RemoveReaction removes a reaction emoji from a Discord message
func (c *DiscordClient) RemoveReaction(channelID, messageID, emoji string) error {
	// Remove the bot's own reaction
	err := c.sdkClient.MessageReactionRemove(channelID, messageID, emoji, "@me")
	if err != nil {
		return fmt.Errorf("failed to remove Discord reaction: %w", err)
	}
	return nil
}

// CreatePublicThread creates a public thread from a message in Discord
func (c *DiscordClient) CreatePublicThread(
	channelID, messageID, threadName string,
) (*clients.DiscordThreadResponse, error) {
	// Use discordgo to create a public thread from the message
	thread, err := c.sdkClient.MessageThreadStart(channelID, messageID, threadName, 60)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord thread: %w", err)
	}

	return &clients.DiscordThreadResponse{
		ThreadID:   thread.ID,
		ThreadName: thread.Name,
	}, nil
}
