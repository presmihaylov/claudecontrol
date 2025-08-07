package clients

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zishang520/socket.io/v2/socket"
)

// OAuthV2Response represents our custom OAuth response with only needed fields
type OAuthV2Response struct {
	TeamID      string
	TeamName    string
	AccessToken string
}

// SlackOAuthClient defines the interface for Slack OAuth operations
type SlackOAuthClient interface {
	GetOAuthV2Response(
		httpClient *http.Client,
		clientID, clientSecret, code, redirectURL string,
	) (*OAuthV2Response, error)
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

// SocketIOClient defines the interface for Socket.IO client operations
type SocketIOClient interface {
	// Router registration
	RegisterWithRouter(router *mux.Router)

	// Client management
	GetClientIDs() []string
	GetClientByID(clientID string) any // Returns *socketio.Client but we use any for interface
	SendMessage(clientID string, msg any) error

	// Event handlers
	RegisterMessageHandler(handler func(client any, msg any))
	RegisterConnectionHook(hook func(client any) error)
	RegisterDisconnectionHook(hook func(client any) error)
	RegisterPingHook(hook func(client any) error)
}

// Client represents a connected WebSocket client
type Client struct {
	ID                 string
	Socket             *socket.Socket
	SlackIntegrationID string
	AgentID            string
}
