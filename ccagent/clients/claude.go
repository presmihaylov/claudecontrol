package clients

import (
	"os"
	"os/exec"
	"strings"

	"ccagent/core"
	"ccagent/core/log"
)

type ClaudeClient struct {
	permissionMode string
}

func NewClaudeClient(permissionMode string) *ClaudeClient {
	return &ClaudeClient{
		permissionMode: permissionMode,
	}
}

func (c *ClaudeClient) ContinueSession(sessionID, prompt string) (string, error) {
	log.Info("📋 Starting to continue Claude session: %s", sessionID)
	args := []string{
		"--permission-mode", c.permissionMode,
		"--verbose",
		"--output-format", "stream-json",
		"--resume", sessionID,
		"-p", prompt,
	}

	log.Info("Executing Claude command with sessionID: %s, prompt: %s", sessionID, prompt)
	log.Info("Command arguments: %v", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()

	log.Info("Running Claude command")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", &core.ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully, outputLength: %d", len(result))
	log.Info("📋 Completed successfully - continued Claude session")
	return result, nil
}

func (c *ClaudeClient) StartNewSession(prompt string) (string, error) {
	log.Info("📋 Starting to create new Claude session")
	args := []string{
		"--permission-mode", c.permissionMode,
		"--verbose",
		"--output-format", "stream-json",
		"-p", prompt,
	}

	log.Info("Starting new Claude session with prompt: %s", prompt)
	log.Info("Command arguments: %v", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()

	log.Info("Running Claude command")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", &core.ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully, outputLength: %d", len(result))
	return result, nil
}

func (c *ClaudeClient) StartNewSessionWithSystemPrompt(prompt, systemPrompt string) (string, error) {
	log.Info("📋 Starting to create new Claude session with system prompt")
	args := []string{
		"--permission-mode", c.permissionMode,
		"--verbose",
		"--output-format", "stream-json",
		"-p", prompt,
		"--append-system-prompt", systemPrompt,
	}

	log.Info("Starting new Claude session with prompt: %s", prompt)
	log.Info("Command arguments: %v", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()

	log.Info("Running Claude command")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", &core.ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("Claude command completed successfully, outputLength: %d", len(result))
	log.Info("📋 Completed successfully - created new Claude session with system prompt")
	return result, nil
}
