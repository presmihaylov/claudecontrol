package clients

// ClaudeClient defines the interface for Claude CLI interactions
type ClaudeClient interface {
	StartNewSession(prompt string) (string, error)
	StartNewSessionWithSystemPrompt(prompt, systemPrompt string) (string, error)
	ContinueSession(sessionID, prompt string) (string, error)
	StartNewSessionWithDisallowedTools(prompt string, disallowedTools []string) (string, error)
	StartNewSessionWithSystemPromptAndDisallowedTools(
		prompt, systemPrompt string,
		disallowedTools []string,
	) (string, error)
	ContinueSessionWithDisallowedTools(sessionID, prompt string, disallowedTools []string) (string, error)
}
