package services

// MockClaudeClient implements the ClaudeClient interface for testing
type MockClaudeClient struct {
	StartNewSessionFunc                              func(prompt string) (string, error)
	StartNewSessionWithSystemFunc                    func(prompt, systemPrompt string) (string, error)
	ContinueSessionFunc                              func(sessionID, prompt string) (string, error)
	StartNewSessionWithDisallowedToolsFunc           func(prompt string, disallowedTools []string) (string, error)
	StartNewSessionWithSystemPromptAndDisallowedFunc func(prompt, systemPrompt string, disallowedTools []string) (string, error)
	ContinueSessionWithDisallowedToolsFunc           func(sessionID, prompt string, disallowedTools []string) (string, error)
}

func (m *MockClaudeClient) StartNewSession(prompt string) (string, error) {
	if m.StartNewSessionFunc != nil {
		return m.StartNewSessionFunc(prompt)
	}
	return "", nil
}

func (m *MockClaudeClient) StartNewSessionWithSystemPrompt(prompt, systemPrompt string) (string, error) {
	if m.StartNewSessionWithSystemFunc != nil {
		return m.StartNewSessionWithSystemFunc(prompt, systemPrompt)
	}
	return "", nil
}

func (m *MockClaudeClient) ContinueSession(sessionID, prompt string) (string, error) {
	if m.ContinueSessionFunc != nil {
		return m.ContinueSessionFunc(sessionID, prompt)
	}
	return "", nil
}

func (m *MockClaudeClient) StartNewSessionWithDisallowedTools(prompt string, disallowedTools []string) (string, error) {
	if m.StartNewSessionWithDisallowedToolsFunc != nil {
		return m.StartNewSessionWithDisallowedToolsFunc(prompt, disallowedTools)
	}
	return "", nil
}

func (m *MockClaudeClient) StartNewSessionWithSystemPromptAndDisallowedTools(
	prompt, systemPrompt string,
	disallowedTools []string,
) (string, error) {
	if m.StartNewSessionWithSystemPromptAndDisallowedFunc != nil {
		return m.StartNewSessionWithSystemPromptAndDisallowedFunc(prompt, systemPrompt, disallowedTools)
	}
	return "", nil
}

func (m *MockClaudeClient) ContinueSessionWithDisallowedTools(
	sessionID, prompt string,
	disallowedTools []string,
) (string, error) {
	if m.ContinueSessionWithDisallowedToolsFunc != nil {
		return m.ContinueSessionWithDisallowedToolsFunc(sessionID, prompt, disallowedTools)
	}
	return "", nil
}
