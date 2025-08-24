package clients

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zishang520/socket.io/v2/socket"

	"ccbackend/models"
)

// OAuthV2Response represents our custom OAuth response with only needed fields
type OAuthV2Response struct {
	TeamID      string
	TeamName    string
	AccessToken string
}

// DiscordGuild represents Discord guild information
type DiscordGuild struct {
	ID   string
	Name string
}

// SlackOAuthClient defines the interface for Slack OAuth operations
type SlackOAuthClient interface {
	GetOAuthV2Response(
		httpClient *http.Client,
		clientID, clientSecret, code, redirectURL string,
	) (*OAuthV2Response, error)
}

// DiscordClient defines the interface for Discord operations
type DiscordClient interface {
	GetGuildByID(guildID string) (*DiscordGuild, error)
	GetBotUser() (*DiscordBotUser, error)
	GetChannelByID(channelID string) (*DiscordChannel, error)
	PostMessage(channelID string, params DiscordMessageParams) (*DiscordPostMessageResponse, error)
	AddReaction(channelID, messageID, emoji string) error
	RemoveReaction(channelID, messageID, emoji string) error
	CreatePublicThread(channelID, messageID, threadName string) (*DiscordThreadResponse, error)
}

// SlackClient defines the interface for Slack API operations
type SlackClient interface {
	SlackOAuthClient

	// Bot operations
	AuthTest() (*SlackAuthTestResponse, error)
	GetPermalink(params *SlackPermalinkParameters) (string, error)

	// User operations
	GetUserInfoContext(ctx context.Context, userID string) (*SlackUser, error)
	ResolveMentionsInMessage(ctx context.Context, message string) string

	// Message operations
	PostMessage(channelID string, params SlackMessageParams) (*SlackPostMessageResponse, error)

	// Reaction operations
	GetReactions(item SlackItemRef, params SlackGetReactionsParameters) ([]SlackItemReaction, error)
	AddReaction(name string, item SlackItemRef) error
	RemoveReaction(name string, item SlackItemRef) error
}

// GitHubClient defines the interface for GitHub API operations
type GitHubClient interface {
	ExchangeCodeForAccessToken(ctx context.Context, code string) (string, error)
	UninstallApp(ctx context.Context, installationID string) error
	ListInstalledRepositories(ctx context.Context, installationID string) ([]models.GitHubRepository, error)
}

// SocketIOClient defines the interface for Socket.IO client operations
type SocketIOClient interface {
	// Router registration
	RegisterWithRouter(router *mux.Router)

	// Client management
	GetClientIDs() []string
	GetClientByID(clientID string) any // Returns *socketio.Client but we use any for interface
	SendMessage(clientID string, msg any) error
	DisconnectClientByID(clientID string) error

	// Event handlers
	RegisterMessageHandler(handler MessageHandlerFunc)
	RegisterConnectionHook(hook ConnectionHookFunc)
	RegisterDisconnectionHook(hook ConnectionHookFunc)
	RegisterPingHook(hook PingHandlerFunc)
}

// Hook and handler function types
type MessageHandlerFunc func(client *Client, msg any) error
type ConnectionHookFunc func(client *Client) error
type PingHandlerFunc func(client *Client) error
type APIKeyValidatorFunc func(apiKey string) (string, error)

// Client represents a connected WebSocket client
type Client struct {
	ID      string
	Socket  *socket.Socket
	OrgID   models.OrgID
	AgentID string
}
