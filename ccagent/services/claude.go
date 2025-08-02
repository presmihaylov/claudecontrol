package services

import (
	"fmt"

	"ccagent/clients"
	"ccagent/core/log"
)

// ClaudeResult represents the result of a Claude conversation
type ClaudeResult struct {
	Output    string
	SessionID string
}

type ClaudeService struct {
	claudeClient *clients.ClaudeClient
}

func NewClaudeService(claudeClient *clients.ClaudeClient) *ClaudeService {
	return &ClaudeService{
		claudeClient: claudeClient,
	}
}

func (c *ClaudeService) StartNewConversation(prompt string) (*ClaudeResult, error) {
	log.Info("📋 Starting to start new Claude conversation")

	messages, err := c.claudeClient.StartNewSession(prompt)
	if err != nil {
		log.Error("Failed to start new Claude session: %v", err)
		return nil, fmt.Errorf("failed to start new Claude session: %w", err)
	}

	sessionID := c.extractSessionID(messages)
	log.Info("📋 Claude session ID: %s", sessionID)

	output, err := c.extractClaudeResult(messages)
	if err != nil {
		log.Error("Failed to extract Claude result: %v", err)
		return nil, fmt.Errorf("failed to extract Claude result: %w", err)
	}

	result := &ClaudeResult{
		Output:    output,
		SessionID: sessionID,
	}

	log.Info("📋 Completed successfully - started new Claude conversation with session: %s", sessionID)
	return result, nil
}


func (c *ClaudeService) StartNewConversationWithSystemPrompt(prompt, systemPrompt string) (*ClaudeResult, error) {
	log.Info("📋 Starting to start new Claude conversation with system prompt")

	messages, err := c.claudeClient.StartNewSessionWithSystemPrompt(prompt, systemPrompt)
	if err != nil {
		log.Error("Failed to start new Claude session with system prompt: %v", err)
		return nil, fmt.Errorf("failed to start new Claude session with system prompt: %w", err)
	}

	sessionID := c.extractSessionID(messages)
	log.Info("📋 Claude session ID: %s", sessionID)

	output, err := c.extractClaudeResult(messages)
	if err != nil {
		log.Error("Failed to extract Claude result: %v", err)
		return nil, fmt.Errorf("failed to extract Claude result: %w", err)
	}

	result := &ClaudeResult{
		Output:    output,
		SessionID: sessionID,
	}

	log.Info("📋 Completed successfully - started new Claude conversation with system prompt, session: %s", sessionID)
	return result, nil
}

func (c *ClaudeService) ContinueConversation(sessionID, prompt string) (*ClaudeResult, error) {
	log.Info("📋 Starting to continue Claude conversation: %s", sessionID)

	messages, err := c.claudeClient.ContinueSession(sessionID, prompt)
	if err != nil {
		log.Error("Failed to continue Claude session: %v", err)
		return nil, fmt.Errorf("failed to continue Claude session: %w", err)
	}

	actualSessionID := c.extractSessionID(messages)
	log.Info("📋 Claude session ID: %s", actualSessionID)

	output, err := c.extractClaudeResult(messages)
	if err != nil {
		log.Error("Failed to extract Claude result: %v", err)
		return nil, fmt.Errorf("failed to extract Claude result: %w", err)
	}

	result := &ClaudeResult{
		Output:    output,
		SessionID: actualSessionID,
	}

	log.Info("📋 Completed successfully - continued Claude conversation with session: %s", actualSessionID)
	return result, nil
}

func (c *ClaudeService) extractSessionID(messages []clients.ClaudeMessage) string {
	if len(messages) > 0 {
		return messages[0].GetSessionID()
	}
	return "unknown"
}

func (c *ClaudeService) extractClaudeResult(messages []clients.ClaudeMessage) (string, error) {
	for i := len(messages) - 1; i >= 0; i-- {
		if assistantMsg, ok := messages[i].(clients.AssistantMessage); ok {
			for _, content := range assistantMsg.Message.Content {
				if content.Type == "text" {
					return content.Text, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no assistant message with text content found")
}
