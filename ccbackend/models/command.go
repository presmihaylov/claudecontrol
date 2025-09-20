package models

// CommandResult represents the result of processing a command
type CommandResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CommandRequest represents a command request from a platform
type CommandRequest struct {
	Command     string `json:"command"`      // The parsed command text
	UserID      string `json:"user_id"`      // User who issued the command
	MessageText string `json:"message_text"` // Original message text for context
}