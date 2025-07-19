package clients

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ccagent/core/log"
)

type ClaudeClient struct{}

func NewClaudeClient() *ClaudeClient {
	return &ClaudeClient{}
}

func (c *ClaudeClient) ContinueSession(sessionID, prompt string) (string, error) {
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
	cmd.Env = append(cmd.Env, "CLAUDE_CONFIG_DIR=\"./.ccagent/claude\"")
	
	log.Info("Running Claude command", "command", "claude", "env", "CLAUDE_CONFIG_DIR=\"./.ccagent/claude\"")
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		log.Error("Claude command failed", "error", err, "output", string(output))
		return "", fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully", "outputLength", len(result))
	log.Info("Claude output", "output", result)

	return result, nil
}

func (c *ClaudeClient) StartNewSession(prompt string) (string, error) {
	args := []string{
		"--permission-mode", "bypassPermissions",
		"-p", prompt,
	}

	log.Info("Starting new Claude session", "prompt", prompt)
	log.Info("Command arguments", "args", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "CLAUDE_CONFIG_DIR=\"./.ccagent/claude\"")
	
	log.Info("Running Claude command", "command", "claude", "env", "CLAUDE_CONFIG_DIR=\"./.ccagent/claude\"")
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		log.Error("Claude command failed", "error", err, "output", string(output))
		return "", fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully", "outputLength", len(result))
	log.Info("Claude output", "output", result)

	return result, nil
}
