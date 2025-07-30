package clients

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ccagent/core/log"
)

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
	cmd.Env = append(cmd.Env, "CLAUDE_CONFIG_DIR=.ccagent/claude")
	cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", c.anthroApiKey))

	log.Info("Running Claude command with env CLAUDE_CONFIG_DIR=.ccagent/claude")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("Claude command failed: %v\nOutput: %s", err, string(output))
		return nil, fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully, outputLength: %d", len(result))
	log.Info("Claude output: %s", result)

	messages, err := mapClaudeOutputToMessages(result)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)
		return nil, fmt.Errorf("failed to parse Claude output: %w", err)
	}

	log.Info("📋 Completed successfully - continued Claude session")
	return messages, nil
}

func (c *ClaudeClient) StartNewSession(prompt string) ([]ClaudeMessage, error) {
	return c.StartNewSessionWithConfigDir(prompt, ".ccagent/claude")
}

func (c *ClaudeClient) StartNewSessionWithConfigDir(prompt, configDir string) ([]ClaudeMessage, error) {
	return c.StartNewSessionWithSystemPrompt(prompt, "", configDir)
}

func (c *ClaudeClient) StartNewSessionWithSystemPrompt(prompt, systemPrompt, configDir string) ([]ClaudeMessage, error) {
	log.Info("📋 Starting to create new Claude session with config dir: %s", configDir)
	args := []string{
		"--permission-mode", c.permissionMode,
		"--output-format", "stream-json",
		"--verbose",
		"-p", prompt,
	}

	if systemPrompt != "" {
		args = append(args, "--append-system-prompt", systemPrompt)
	}

	log.Info("Starting new Claude session with prompt: %s, configDir: %s", prompt, configDir)
	log.Info("Command arguments: %v", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", configDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", c.anthroApiKey))

	log.Info("Running Claude command with env CLAUDE_CONFIG_DIR=%s", configDir)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("Claude command failed: %v\nOutput: %s", err, string(output))
		return nil, fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully, outputLength: %d", len(result))
	log.Info("Claude output: %s", result)

	messages, err := mapClaudeOutputToMessages(result)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)
		return nil, fmt.Errorf("failed to parse Claude output: %w", err)
	}

	log.Info("📋 Completed successfully - started new Claude session")
	return messages, nil
}
