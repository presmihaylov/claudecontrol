package models

// CommandResult represents the result of processing a command
type CommandResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CommandRequest represents a command request from a platform
type CommandRequest struct {
	Command     string      `json:"command"`
	Platform    ChannelType `json:"platform"`
	TeamID      string      `json:"team_id"`      // Slack team ID or Discord guild ID
	ChannelID   string      `json:"channel_id"`
	UserID      string      `json:"user_id"`
	MessageText string      `json:"message_text"` // Original message text for context
}