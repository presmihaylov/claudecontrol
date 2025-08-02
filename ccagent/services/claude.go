package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"ccagent/clients"
	"ccagent/core/log"
)

// ClaudeResult represents the result of a Claude conversation
type ClaudeResult struct {
	Output    string
	SessionID string
}

// ClaudeMessage represents a message from Claude command output
type ClaudeMessage interface {
	GetType() string
	GetSessionID() string
}

// AssistantMessage represents an assistant message from Claude
type AssistantMessage struct {
	Type    string `json:"type"`
	Message struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
	SessionID string `json:"session_id"`
}

func (a AssistantMessage) GetType() string {
	return a.Type
}

func (a AssistantMessage) GetSessionID() string {
	return a.SessionID
}

// UnknownClaudeMessage represents an unknown message type from Claude
type UnknownClaudeMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

func (u UnknownClaudeMessage) GetType() string {
	return u.Type
}

func (u UnknownClaudeMessage) GetSessionID() string {
	return u.SessionID
}

// mapClaudeOutputToMessages parses Claude command output into structured messages
func (c *ClaudeService) mapClaudeOutputToMessages(output string) ([]ClaudeMessage, error) {
	var messages []ClaudeMessage
	
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		// Try to parse as AssistantMessage first
		var assistantMsg AssistantMessage
		if err := json.Unmarshal([]byte(line), &assistantMsg); err == nil && assistantMsg.Type == "assistant" {
			messages = append(messages, assistantMsg)
			continue
		}
		
		// Fallback to UnknownClaudeMessage
		var unknownMsg struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
		}
		
		if err := json.Unmarshal([]byte(line), &unknownMsg); err == nil {
			messages = append(messages, UnknownClaudeMessage{
				Type:      unknownMsg.Type,
				SessionID: unknownMsg.SessionID,
			})
		} else {
			// If even basic parsing fails, create unknown message
			messages = append(messages, UnknownClaudeMessage{
				Type:      "unknown",
				SessionID: "",
			})
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	return messages, nil
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

	rawOutput, err := c.claudeClient.StartNewSession(prompt)
	if err != nil {
		log.Error("Failed to start new Claude session: %v", err)
		return nil, c.handleClaudeClientError(err, "failed to start new Claude session")
	}

	messages, err := c.mapClaudeOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)
		return nil, fmt.Errorf("failed to parse Claude output: %w", err)
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

	rawOutput, err := c.claudeClient.StartNewSessionWithSystemPrompt(prompt, systemPrompt)
	if err != nil {
		log.Error("Failed to start new Claude session with system prompt: %v", err)
		return nil, c.handleClaudeClientError(err, "failed to start new Claude session with system prompt")
	}

	messages, err := c.mapClaudeOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)
		return nil, fmt.Errorf("failed to parse Claude output: %w", err)
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

	rawOutput, err := c.claudeClient.ContinueSession(sessionID, prompt)
	if err != nil {
		log.Error("Failed to continue Claude session: %v", err)
		return nil, c.handleClaudeClientError(err, "failed to continue Claude session")
	}

	messages, err := c.mapClaudeOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)
		return nil, fmt.Errorf("failed to parse Claude output: %w", err)
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

func (c *ClaudeService) extractSessionID(messages []ClaudeMessage) string {
	if len(messages) > 0 {
		return messages[0].GetSessionID()
	}
	return "unknown"
}

func (c *ClaudeService) extractClaudeResult(messages []ClaudeMessage) (string, error) {
	for i := len(messages) - 1; i >= 0; i-- {
		if assistantMsg, ok := messages[i].(AssistantMessage); ok {
			for _, content := range assistantMsg.Message.Content {
				if content.Type == "text" {
					return content.Text, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no assistant message with text content found")
}

// handleClaudeClientError processes errors from Claude client calls.
// If the error is a Claude command error, it attempts to extract the assistant message
// and returns a new error with the clean message. Otherwise, returns the original error.
func (c *ClaudeService) handleClaudeClientError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check if this is a Claude command error  
	claudeErr, isClaudeErr := clients.IsClaudeCommandErr(err)
	if !isClaudeErr {
		// Not a Claude command error, return original error wrapped
		return fmt.Errorf("%s: %w", operation, err)
	}

	// Try to parse the output as Claude messages using internal parsing
	messages, parseErr := c.mapClaudeOutputToMessages(claudeErr.Output)
	if parseErr != nil {
		// If parsing fails, return original error wrapped
		log.Error("Failed to parse Claude output from error: %v", parseErr)
		return fmt.Errorf("%s: %w", operation, err)
	}

	// Try to extract the assistant message using existing logic
	for i := len(messages) - 1; i >= 0; i-- {
		if assistantMsg, ok := messages[i].(AssistantMessage); ok {
			for _, content := range assistantMsg.Message.Content {
				if content.Type == "text" {
					log.Info("✅ Successfully extracted Claude message from error: %s", content.Text)
					return fmt.Errorf("%s: %s", operation, content.Text)
				}
			}
		}
	}

	// No assistant message found, return original error wrapped
	log.Info("⚠️ No assistant message found in Claude command output, returning original error")
	return fmt.Errorf("%s: %w", operation, err)
}
