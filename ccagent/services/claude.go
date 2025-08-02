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

// ExtractMessageFromError attempts to extract a Claude assistant message from a Claude command error.
// If the error is not a Claude command error or message extraction fails, it returns the original error message.
func (c *ClaudeService) ExtractMessageFromError(err error) string {
	if err == nil {
		return ""
	}

	// Check if this is a Claude command error
	claudeErr, isClaudeErr := clients.IsClaudeCommandErr(err)
	if !isClaudeErr {
		// Not a Claude command error, return original error message
		return err.Error()
	}

	// Try to parse the output as Claude messages
	messages, parseErr := clients.MapClaudeOutputToMessages(claudeErr.Output)
	if parseErr != nil {
		// If parsing fails, return original error message
		log.Error("Failed to parse Claude output from error: %v", parseErr)
		return err.Error()
	}

	// Try to extract the assistant message using existing logic
	for i := len(messages) - 1; i >= 0; i-- {
		if assistantMsg, ok := messages[i].(clients.AssistantMessage); ok {
			for _, content := range assistantMsg.Message.Content {
				if content.Type == "text" {
					log.Info("✅ Successfully extracted Claude message from error: %s", content.Text)
					return content.Text
				}
			}
		}
	}

	// No assistant message found, return original error message
	log.Info("⚠️ No assistant message found in Claude command output, returning original error")
	return err.Error()
}
