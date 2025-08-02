package clients

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ccagent/core/log"
)

// ErrClaudeCommandErr represents an error from the Claude command that includes the command output
type ErrClaudeCommandErr struct {
	Err    error  // The original command error
	Output string // The Claude command output (may contain JSON response)
}

func (e *ErrClaudeCommandErr) Error() string {
	return fmt.Sprintf("claude command failed: %v\nOutput: %s", e.Err, e.Output)
}

func (e *ErrClaudeCommandErr) Unwrap() error {
	return e.Err
}

// IsClaudeCommandErr checks if an error is a Claude command error
func IsClaudeCommandErr(err error) (*ErrClaudeCommandErr, bool) {
	var claudeErr *ErrClaudeCommandErr
	if errors.As(err, &claudeErr) {
		return claudeErr, true
	}
	return nil, false
}

type ClaudeClient struct {
	anthroApiKey   string
	permissionMode string
}

func NewClaudeClient(anthroApiKey string, permissionMode string) *ClaudeClient {
	return &ClaudeClient{
		anthroApiKey:   anthroApiKey,
		permissionMode: permissionMode,
	}
}

func (c *ClaudeClient) ContinueSession(sessionID, prompt string) ([]ClaudeMessage, error) {
	log.Info("📋 Starting to continue Claude session: %s", sessionID)
	args := []string{
		"--permission-mode", c.permissionMode,
		"--output-format", "stream-json",
		"--verbose",
		"--resume", sessionID,
		"-p", prompt,
	}

	log.Info("Executing Claude command with sessionID: %s, prompt: %s", sessionID, prompt)
	log.Info("Command arguments: %v", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()
	if c.anthroApiKey != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", c.anthroApiKey))
	}

	log.Info("Running Claude command")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("Claude command failed: %v\nOutput: %s", err, string(output))
		return nil, &ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully, outputLength: %d", len(result))
	log.Info("Claude output: %s", result)

	messages, err := MapClaudeOutputToMessages(result)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)
		return nil, fmt.Errorf("failed to parse Claude output: %w", err)
	}

	log.Info("📋 Completed successfully - continued Claude session")
	return messages, nil
}

func (c *ClaudeClient) StartNewSession(prompt string) ([]ClaudeMessage, error) {
	log.Info("📋 Starting to create new Claude session")
	args := []string{
		"--permission-mode", c.permissionMode,
		"--output-format", "stream-json",
		"--verbose",
		"-p", prompt,
	}

	log.Info("Starting new Claude session with prompt: %s", prompt)
	log.Info("Command arguments: %v", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()
	if c.anthroApiKey != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", c.anthroApiKey))
	}

	log.Info("Running Claude command")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("Claude command failed: %v\nOutput: %s", err, string(output))
		return nil, &ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully, outputLength: %d", len(result))
	log.Info("Claude output: %s", result)

	messages, err := MapClaudeOutputToMessages(result)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)
		return nil, fmt.Errorf("failed to parse Claude output: %w", err)
	}

	log.Info("📋 Completed successfully - started new Claude session")
	return messages, nil
}


func (c *ClaudeClient) StartNewSessionWithSystemPrompt(prompt, systemPrompt string) ([]ClaudeMessage, error) {
	log.Info("📋 Starting to create new Claude session with system prompt")
	args := []string{
		"--permission-mode", c.permissionMode,
		"--output-format", "stream-json",
		"--verbose",
		"-p", prompt,
	}

	if systemPrompt != "" {
		args = append(args, "--append-system-prompt", systemPrompt)
	}

	log.Info("Starting new Claude session with prompt: %s", prompt)
	log.Info("Command arguments: %v", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()
	if c.anthroApiKey != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", c.anthroApiKey))
	}

	log.Info("Running Claude command")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("Claude command failed: %v\nOutput: %s", err, string(output))
		return nil, &ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully, outputLength: %d", len(result))
	log.Info("Claude output: %s", result)

	messages, err := MapClaudeOutputToMessages(result)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)
		return nil, fmt.Errorf("failed to parse Claude output: %w", err)
	}

	log.Info("📋 Completed successfully - started new Claude session")
	return messages, nil
}
