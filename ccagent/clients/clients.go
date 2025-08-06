package clients

// ClaudeClient defines the interface for Claude CLI interactions
type ClaudeClient interface {
	StartNewSession(prompt string) (string, error)
	StartNewSessionWithSystemPrompt(prompt, systemPrompt string) (string, error)
	ContinueSession(sessionID, prompt string) (string, error)
}
