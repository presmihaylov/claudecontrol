package clients

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ccagent/core/log"
)

type ClaudeClient struct {
	anthroApiKey string
}

func NewClaudeClient(anthroApiKey string) *ClaudeClient {
	return &ClaudeClient{
		anthroApiKey: anthroApiKey,
	}
}

func (c *ClaudeClient) ContinueSession(sessionID, prompt string) (string, error) {
	log.Info("📋 Starting to continue Claude session: %s", sessionID)
	// Not used at the moment, because claude code doesn't support continuing sessions due to a bug:
	// https://github.com/anthropics/claude-code/issues/3976
	_ = sessionID
	args := []string{
		"--permission-mode", "bypassPermissions",
		"--continue",
		"-p", prompt,
	}

	log.Info("Executing Claude command", "sessionID", sessionID, "prompt", prompt)
	log.Info("Command arguments", "args", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "CLAUDE_CONFIG_DIR=.ccagent/claude")
	cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", c.anthroApiKey))

	log.Info("Running Claude command", "command", "claude", "env", "CLAUDE_CONFIG_DIR=.ccagent/claude")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("Claude command failed", "error", err, "output", string(output))
		return "", fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully", "outputLength", len(result))
	log.Info("Claude output", "output", result)
	log.Info("📋 Completed successfully - continued Claude session")

	return result, nil
}

func (c *ClaudeClient) StartNewSession(prompt string) (string, error) {
	return c.StartNewSessionWithConfigDir(prompt, ".ccagent/claude")
}

func (c *ClaudeClient) StartNewSessionWithConfigDir(prompt, configDir string) (string, error) {
	log.Info("📋 Starting to create new Claude session with config dir: %s", configDir)
	args := []string{
		"--permission-mode", "bypassPermissions",
		"-p", prompt,
	}

	log.Info("Starting new Claude session", "prompt", prompt, "configDir", configDir)
	log.Info("Command arguments", "args", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", configDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", c.anthroApiKey))

	log.Info("Running Claude command", "command", "claude", "env", fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", configDir))
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("Claude command failed", "error", err, "output", string(output))
		return "", fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully", "outputLength", len(result))
	log.Info("Claude output", "output", result)
	log.Info("📋 Completed successfully - started new Claude session")

	return result, nil
}
