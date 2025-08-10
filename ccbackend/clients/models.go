package clients

import "github.com/samber/mo"

// SlackAuthTestResponse represents the response from Slack's auth.test API
type SlackAuthTestResponse struct {
	UserID string
	TeamID string
}

// SlackPermalinkParameters represents parameters for generating a Slack permalink
type SlackPermalinkParameters struct {
	Channel string
	TS      string
}

// SlackUser represents a Slack user
type SlackUser struct {
	ID      string
	Name    string
	Profile SlackUserProfile
}

// SlackUserProfile represents a Slack user's profile information
type SlackUserProfile struct {
	DisplayName string
	RealName    string
}

// SlackPostMessageResponse represents the response from posting a message to Slack
type SlackPostMessageResponse struct {
	Channel   string
	Timestamp string
}

// SlackItemRef represents a reference to a Slack message item
type SlackItemRef struct {
	Channel   string
	Timestamp string
}

// SlackGetReactionsParameters represents parameters for getting reactions
type SlackGetReactionsParameters struct {
	// Add any specific parameters if needed in the future
}

// SlackItemReaction represents a reaction on a Slack message
type SlackItemReaction struct {
	Name  string
	Users []string
}

// SlackMessageParams holds parameters for sending Slack messages
type SlackMessageParams struct {
	Text     string
	ThreadTS mo.Option[string]
}

// DiscordBotUser represents Discord bot user information
type DiscordBotUser struct {
	ID       string
	Username string
	Bot      bool
}

// DiscordMessageParams holds parameters for sending Discord messages
type DiscordMessageParams struct {
	Content  string
	ThreadID *string // For sending messages in threads
}

// DiscordPostMessageResponse represents the response from posting a message to Discord
type DiscordPostMessageResponse struct {
	ChannelID string
	MessageID string
}

// DiscordChannel represents Discord channel information
type DiscordChannel struct {
	ID      string
	Name    string
	Type    int
	GuildID string
}

// DiscordThreadResponse represents the response from creating a Discord thread
type DiscordThreadResponse struct {
	ThreadID   string
	ThreadName string
}
