package clients

import (
	"os"
	"os/exec"
	"strings"
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
	args := []string{
		"--permission-mode", c.permissionMode,
		"--verbose",
		"--output-format", "stream-json",
		"--resume", sessionID,
		"-p", prompt,
	}

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", &ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	return strings.TrimSpace(string(output)), nil
}

func (c *ClaudeClient) StartNewSession(prompt string) (string, error) {
	args := []string{
		"--permission-mode", c.permissionMode,
		"--verbose",
		"--output-format", "stream-json",
		"-p", prompt,
	}

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", &ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	return strings.TrimSpace(string(output)), nil
}

func (c *ClaudeClient) StartNewSessionWithSystemPrompt(prompt, systemPrompt string) (string, error) {
	args := []string{
		"--permission-mode", c.permissionMode,
		"--verbose",
		"--output-format", "stream-json",
		"-p", prompt,
	}

	if systemPrompt != "" {
		args = append(args, "--append-system-prompt", systemPrompt)
	}

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", &ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	return strings.TrimSpace(string(output)), nil
}
