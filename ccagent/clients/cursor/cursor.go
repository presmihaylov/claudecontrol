package cursor

import (
	"os"
	"os/exec"
	"strings"

	"ccagent/clients"
	"ccagent/core"
	"ccagent/core/log"
)

type CursorClient struct {
	// No permissionMode needed for cursor-agent as it handles permissions differently
}

func NewCursorClient() *CursorClient {
	return &CursorClient{}
}

func (c *CursorClient) StartNewSession(prompt string, options *clients.CursorOptions) (string, error) {
	log.Info("📋 Starting to create new Cursor session")

	// Prepend system prompt if provided in options
	finalPrompt := prompt
	if options != nil && options.SystemPrompt != "" {
		finalPrompt = options.SystemPrompt + "\n\n" + prompt
		log.Info("Prepending system prompt to user prompt")
	}

	args := []string{
		"--print",
		"--output-format", "stream-json",
		finalPrompt,
	}

	log.Info("Starting new Cursor session with prompt: %s", finalPrompt)
	log.Info("Command arguments: %v", args)

	cmd := exec.Command("cursor-agent", args...)
	cmd.Env = os.Environ() // Inherit parent environment including CURSOR_API_KEY

	log.Info("Running Cursor command")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &core.ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("Cursor command completed successfully, outputLength: %d", len(result))
	log.Info("📋 Completed successfully - created new Cursor session")
	return result, nil
}

func (c *CursorClient) ContinueSession(sessionID, prompt string, options *clients.CursorOptions) (string, error) {
	log.Info("📋 Starting to continue Cursor session: %s", sessionID)
	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--resume", sessionID,
		prompt,
	}

	log.Info("Executing Cursor command with sessionID: %s, prompt: %s", sessionID, prompt)
	log.Info("Command arguments: %v", args)

	cmd := exec.Command("cursor-agent", args...)
	cmd.Env = os.Environ() // Inherit parent environment including CURSOR_API_KEY

	log.Info("Running Cursor command")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &core.ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("Cursor command completed successfully, outputLength: %d", len(result))
	log.Info("📋 Completed successfully - continued Cursor session")
	return result, nil
}
