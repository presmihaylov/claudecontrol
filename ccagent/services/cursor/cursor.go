package cursor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ccagent/clients"
	"ccagent/core"
	"ccagent/core/log"
	"ccagent/services"
)

type CursorService struct {
	cursorClient clients.CursorClient
	logDir       string
}

func NewCursorService(cursorClient clients.CursorClient, logDir string) *CursorService {
	return &CursorService{
		cursorClient: cursorClient,
		logDir:       logDir,
	}
}

// writeCursorSessionLog writes Cursor output to a timestamped log file and returns the filepath
func (c *CursorService) writeCursorSessionLog(rawOutput string) (string, error) {
	if err := os.MkdirAll(c.logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("cursor-session-%s.log", timestamp)
	filepath := filepath.Join(c.logDir, filename)

	if err := os.WriteFile(filepath, []byte(rawOutput), 0600); err != nil {
		return "", fmt.Errorf("failed to write log file: %w", err)
	}

	return filepath, nil
}

// CleanupOldLogs removes log files older than the specified number of days
func (c *CursorService) CleanupOldLogs(maxAgeDays int) error {
	log.Info("📋 Starting to cleanup old Cursor session logs older than %d days", maxAgeDays)

	if maxAgeDays <= 0 {
		return fmt.Errorf("maxAgeDays must be greater than 0")
	}

	files, err := os.ReadDir(c.logDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("📋 Log directory does not exist, nothing to clean up")
			return nil
		}
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	cutoffTime := time.Now().AddDate(0, 0, -maxAgeDays)
	removedCount := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Only clean up cursor session log files
		if !strings.HasPrefix(file.Name(), "cursor-session-") || !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		filePath := filepath.Join(c.logDir, file.Name())
		info, err := file.Info()
		if err != nil {
			log.Error("Failed to get file info for %s: %v", filePath, err)
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(filePath); err != nil {
				log.Error("Failed to remove old log file %s: %v", filePath, err)
				continue
			}
			removedCount++
		}
	}

	log.Info("📋 Completed successfully - removed %d old Cursor session log files", removedCount)
	return nil
}

func (c *CursorService) StartNewConversation(prompt string) (*services.CLIAgentResult, error) {
	return c.StartNewConversationWithOptions(prompt, nil)
}

func (c *CursorService) StartNewConversationWithOptions(
	prompt string,
	options *clients.ClaudeOptions, // Reusing Claude options for consistency
) (*services.CLIAgentResult, error) {
	log.Info("📋 Starting to start new Cursor conversation")
	rawOutput, err := c.cursorClient.StartNewSession(prompt)
	if err != nil {
		log.Error("Failed to start new Cursor session: %v", err)
		return nil, c.handleCursorClientError(err, "failed to start new Cursor session")
	}

	// Always log the Cursor session
	logPath, writeErr := c.writeCursorSessionLog(rawOutput)
	if writeErr != nil {
		log.Error("Failed to write Cursor session log: %v", writeErr)
	}

	messages, err := c.mapCursorOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Cursor output: %v", err)

		return nil, &core.ClaudeParseError{ // Reusing Claude parse error for consistency
			Message:     fmt.Sprintf("couldn't parse cursor response and instead stored the response in %s", logPath),
			LogFilePath: logPath,
			OriginalErr: err,
		}
	}

	sessionID := c.extractSessionID(messages)
	output, err := c.extractCursorResult(messages)
	if err != nil {
		log.Error("Failed to extract Cursor result: %v", err)
		return nil, fmt.Errorf("failed to extract Cursor result: %w", err)
	}

	log.Info("📋 Cursor response extracted successfully, session: %s, output length: %d", sessionID, len(output))
	result := &services.CLIAgentResult{
		Output:    output,
		SessionID: sessionID,
	}

	log.Info("📋 Completed successfully - started new Cursor conversation with session: %s", sessionID)
	return result, nil
}

func (c *CursorService) StartNewConversationWithSystemPrompt(
	prompt, systemPrompt string,
) (*services.CLIAgentResult, error) {
	return c.StartNewConversationWithOptions(prompt, &clients.ClaudeOptions{
		SystemPrompt: systemPrompt,
	})
}

func (c *CursorService) StartNewConversationWithDisallowedTools(
	prompt string,
	disallowedTools []string,
) (*services.CLIAgentResult, error) {
	return c.StartNewConversationWithOptions(prompt, &clients.ClaudeOptions{
		DisallowedTools: disallowedTools,
	})
}

func (c *CursorService) ContinueConversation(sessionID, prompt string) (*services.CLIAgentResult, error) {
	return c.ContinueConversationWithOptions(sessionID, prompt, nil)
}

func (c *CursorService) ContinueConversationWithOptions(
	sessionID, prompt string,
	options *clients.ClaudeOptions,
) (*services.CLIAgentResult, error) {
	log.Info("📋 Starting to continue Cursor conversation: %s", sessionID)
	rawOutput, err := c.cursorClient.ContinueSession(sessionID, prompt)
	if err != nil {
		log.Error("Failed to continue Cursor session: %v", err)
		return nil, c.handleCursorClientError(err, "failed to continue Cursor session")
	}

	// Always log the Cursor session
	logPath, writeErr := c.writeCursorSessionLog(rawOutput)
	if writeErr != nil {
		log.Error("Failed to write Cursor session log: %v", writeErr)
	}

	messages, err := c.mapCursorOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Cursor output: %v", err)

		return nil, &core.ClaudeParseError{
			Message:     fmt.Sprintf("couldn't parse cursor response and instead stored the response in %s", logPath),
			LogFilePath: logPath,
			OriginalErr: err,
		}
	}

	actualSessionID := c.extractSessionID(messages)
	output, err := c.extractCursorResult(messages)
	if err != nil {
		log.Error("Failed to extract Cursor result: %v", err)
		return nil, fmt.Errorf("failed to extract Cursor result: %w", err)
	}

	log.Info("📋 Cursor response extracted successfully, session: %s, output length: %d", actualSessionID, len(output))
	result := &services.CLIAgentResult{
		Output:    output,
		SessionID: actualSessionID,
	}

	log.Info("📋 Completed successfully - continued Cursor conversation with session: %s", actualSessionID)
	return result, nil
}

// CursorMessage represents a simplified message interface for Cursor
type CursorMessage interface {
	GetType() string
	GetSessionID() string
}

// CursorResultMessage represents a result message from Cursor (simplified version)
type CursorResultMessage struct {
	Type      string `json:"type"`
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

func (r CursorResultMessage) GetType() string {
	return r.Type
}

func (r CursorResultMessage) GetSessionID() string {
	return r.SessionID
}

// UnknownCursorMessage represents an unknown message type from Cursor
type UnknownCursorMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

func (u UnknownCursorMessage) GetType() string {
	return u.Type
}

func (u UnknownCursorMessage) GetSessionID() string {
	return u.SessionID
}

// mapCursorOutputToMessages parses Cursor command output focusing only on result messages
func (c *CursorService) mapCursorOutputToMessages(output string) ([]CursorMessage, error) {
	var messages []CursorMessage

	// Use a scanner with a larger buffer to handle long lines
	scanner := bufio.NewScanner(strings.NewReader(output))
	// Set a 1MB buffer to handle very long JSON lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse the message focusing only on result type
		message := c.parseCursorMessage([]byte(line))
		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// parseCursorMessage attempts to parse a JSON line into the appropriate message type
func (c *CursorService) parseCursorMessage(lineBytes []byte) CursorMessage {
	// First, extract just the type to determine which struct to use
	var typeCheck struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(lineBytes, &typeCheck); err != nil {
		// If we can't even parse the type, return unknown message
		return UnknownCursorMessage{
			Type:      "unknown",
			SessionID: "",
		}
	}

	// Parse based on type - only focus on result messages for simplicity
	switch typeCheck.Type {
	case "result":
		var resultMsg CursorResultMessage
		if err := json.Unmarshal(lineBytes, &resultMsg); err == nil {
			return resultMsg
		}
	}

	// For all other types, extract basic info for unknown message
	var unknownMsg struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
	}

	if err := json.Unmarshal(lineBytes, &unknownMsg); err == nil {
		return UnknownCursorMessage{
			Type:      unknownMsg.Type,
			SessionID: unknownMsg.SessionID,
		}
	}

	// Return default unknown message
	return UnknownCursorMessage{
		Type:      "unknown",
		SessionID: "",
	}
}

func (c *CursorService) extractSessionID(messages []CursorMessage) string {
	if len(messages) > 0 {
		return messages[0].GetSessionID()
	}
	return "unknown"
}

func (c *CursorService) extractCursorResult(messages []CursorMessage) (string, error) {
	// Look for result message type (only type we care about)
	for i := len(messages) - 1; i >= 0; i-- {
		if resultMsg, ok := messages[i].(CursorResultMessage); ok {
			if resultMsg.Result != "" {
				return resultMsg.Result, nil
			}
		}
	}
	return "", fmt.Errorf("no result message found")
}

// handleCursorClientError processes errors from Cursor client calls.
func (c *CursorService) handleCursorClientError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check if this is a Cursor command error (reusing Claude error type)
	claudeErr, isClaudeErr := core.IsClaudeCommandErr(err)
	if !isClaudeErr {
		// Not a command error, return original error wrapped
		return fmt.Errorf("%s: %w", operation, err)
	}

	// Try to parse the output as Cursor messages using internal parsing
	messages, parseErr := c.mapCursorOutputToMessages(claudeErr.Output)
	if parseErr != nil {
		// If parsing fails, return original error wrapped
		log.Error("Failed to parse Cursor output from error: %v", parseErr)
		return fmt.Errorf("%s: %w", operation, err)
	}

	// Try to extract the result message
	for i := len(messages) - 1; i >= 0; i-- {
		if resultMsg, ok := messages[i].(CursorResultMessage); ok {
			if resultMsg.Result != "" {
				log.Info("✅ Successfully extracted Cursor result message from error: %s", resultMsg.Result)
				return fmt.Errorf("%s: %s", operation, resultMsg.Result)
			}
		}
	}

	// No result message found, return original error wrapped
	log.Info("⚠️ No result message found in Cursor command output, returning original error")
	return fmt.Errorf("%s: %w", operation, err)
}
