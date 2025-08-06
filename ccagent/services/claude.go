package services

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ccagent/clients"
	"ccagent/core"
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

// writeClaudeErrorLog writes Claude output to a timestamped log file and returns the filepath
func (c *ClaudeService) writeClaudeErrorLog(rawOutput string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	logDir := filepath.Join(homeDir, ".config", "ccagent", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("claude-error-%s.log", timestamp)
	filepath := filepath.Join(logDir, filename)

	if err := os.WriteFile(filepath, []byte(rawOutput), 0600); err != nil {
		return "", fmt.Errorf("failed to write log file: %w", err)
	}

	return filepath, nil
}

func (c *ClaudeService) StartNewConversation(prompt string) (*ClaudeResult, error) {
	log.Info("📋 Starting to start new Claude conversation")

	rawOutput, err := c.claudeClient.StartNewSession(prompt)
	if err != nil {
		log.Error("Failed to start new Claude session: %v", err)
		return nil, c.handleClaudeClientError(err, "failed to start new Claude session")
	}

	messages, err := MapClaudeOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)

		// Write raw output to error log file
		logPath, writeErr := c.writeClaudeErrorLog(rawOutput)
		if writeErr != nil {
			log.Error("Failed to write Claude error log: %v", writeErr)
			return nil, fmt.Errorf("failed to parse Claude output: %w", err)
		}

		return nil, &core.ClaudeParseError{
			Message:     fmt.Sprintf("couldn't parse claude response and instead stored the response in %s", logPath),
			LogFilePath: logPath,
			OriginalErr: err,
		}
	}

	sessionID := c.extractSessionID(messages)

	output, err := c.extractClaudeResult(messages)
	if err != nil {
		log.Error("Failed to extract Claude result: %v", err)
		return nil, fmt.Errorf("failed to extract Claude result: %w", err)
	}

	log.Info("📋 Claude response extracted successfully, session: %s, output length: %d", sessionID, len(output))

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

	messages, err := MapClaudeOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)

		// Write raw output to error log file
		logPath, writeErr := c.writeClaudeErrorLog(rawOutput)
		if writeErr != nil {
			log.Error("Failed to write Claude error log: %v", writeErr)
			return nil, fmt.Errorf("failed to parse Claude output: %w", err)
		}

		return nil, &core.ClaudeParseError{
			Message:     fmt.Sprintf("couldn't parse claude response and instead stored the response in %s", logPath),
			LogFilePath: logPath,
			OriginalErr: err,
		}
	}

	sessionID := c.extractSessionID(messages)

	output, err := c.extractClaudeResult(messages)
	if err != nil {
		log.Error("Failed to extract Claude result: %v", err)
		return nil, fmt.Errorf("failed to extract Claude result: %w", err)
	}

	log.Info("📋 Claude response extracted successfully, session: %s, output length: %d", sessionID, len(output))

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

	messages, err := MapClaudeOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)

		// Write raw output to error log file
		logPath, writeErr := c.writeClaudeErrorLog(rawOutput)
		if writeErr != nil {
			log.Error("Failed to write Claude error log: %v", writeErr)
			return nil, fmt.Errorf("failed to parse Claude output: %w", err)
		}

		return nil, &core.ClaudeParseError{
			Message:     fmt.Sprintf("couldn't parse claude response and instead stored the response in %s", logPath),
			LogFilePath: logPath,
			OriginalErr: err,
		}
	}

	actualSessionID := c.extractSessionID(messages)

	output, err := c.extractClaudeResult(messages)
	if err != nil {
		log.Error("Failed to extract Claude result: %v", err)
		return nil, fmt.Errorf("failed to extract Claude result: %w", err)
	}

	log.Info("📋 Claude response extracted successfully, session: %s, output length: %d", actualSessionID, len(output))

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
	claudeErr, isClaudeErr := core.IsClaudeCommandErr(err)
	if !isClaudeErr {
		// Not a Claude command error, return original error wrapped
		return fmt.Errorf("%s: %w", operation, err)
	}

	// Try to parse the output as Claude messages using internal parsing
	messages, parseErr := MapClaudeOutputToMessages(claudeErr.Output)
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
