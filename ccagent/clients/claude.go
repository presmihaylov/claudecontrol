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
	log.Info("📋 Starting to create new Claude session")
	
	// Prepend Slack context to the user prompt
	slackContext := "## Slack Integration Context\n\n" +
		"**Important: You are being accessed through a Slack integration.** Users are interacting with you from within Slack channels and threads, and your responses will be displayed directly in Slack.\n\n" +
		"### Slack Markdown Support (Limited):\n" +
		"Slack has LIMITED markdown support compared to GitHub. Use these formats:\n" +
		"- *Bold text* (use single asterisks, NOT double)\n" +
		"- _Italic text_ (use underscores only)\n" +
		"- ```code blocks``` (fenced code blocks work, but NO syntax highlighting)\n" +
		"- `inline code` (backticks work for inline code)\n" +
		"- ~strikethrough~ (single tilde, not double)\n" +
		"- > blockquotes work\n" +
		"- Lists work only if WYSIWYG editor is disabled\n" +
		"- NO support for: headings (#), tables, HTML, or most standard markdown\n" +
		"- Use :emoji: shortcodes or copy-paste emoji\n\n" +
		"### Response Length Management:\n" +
		"- *Keep responses under 4000 characters* (Slack's message limit)\n" +
		"- For longer content, break into multiple focused messages\n" +
		"- Prioritize the most important information first\n" +
		"- Use thread replies for follow-up details if needed\n\n" +
		"### Repository Context:\n" +
		"- Always mention which files you're modifying and why\n" +
		"- Provide `file_path:line_number` references when discussing code\n" +
		"- Include git status context when making changes\n" +
		"- Suggest appropriate commit messages for changes made\n" +
		"- Explain the impact of changes on the overall codebase\n\n" +
		"### Security Awareness:\n" +
		"- *NEVER expose API keys, tokens, or sensitive data* in Slack messages\n" +
		"- Redact or sanitize command outputs that might contain secrets\n" +
		"- Be mindful that Slack channels may be visible to multiple team members\n" +
		"- Mask sensitive file paths or environment variables when necessary\n" +
		"- Warn users before displaying potentially sensitive configuration\n\n" +
		"### Error Handling:\n" +
		"- Present errors in a user-friendly way with suggested next steps\n" +
		"- Avoid raw stack traces - summarize the issue and solution approach\n" +
		"- Include relevant file paths and line numbers for debugging\n" +
		"- Provide actionable remediation steps\n" +
		"- If errors are complex, break down the problem into smaller parts\n\n" +
		"### Slack-Specific Considerations:\n" +
		"- Users interact via @mentions and slash commands\n" +
		"- Responses appear in real-time in Slack threads\n" +
		"- Users expect immediate value and clear next steps\n" +
		"- Code snippets should be production-ready when possible\n" +
		"- Use emojis strategically for clarity and engagement\n\n" +
		"### Response Style:\n" +
		"- Be direct and actionable\n" +
		"- Provide clear explanations for any changes you make\n" +
		"- When working with files, explain what you're doing and why\n" +
		"- If you need to run multiple commands, group them logically\n" +
		"- Always verify your changes work by running builds/tests when appropriate\n\n" +
		"---\n\n" +
		"**User Request:**\n"

	fullPrompt := slackContext + prompt
	
	args := []string{
		"--permission-mode", "bypassPermissions",
		"-p", fullPrompt,
	}

	log.Info("Starting new Claude session", "prompt", prompt)
	log.Info("Command arguments", "args", args)

	cmd := exec.Command("claude", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "CLAUDE_CONFIG_DIR=.ccagent/claude")

	log.Info("Running Claude command", "command", "claude", "env", "CLAUDE_CONFIG_DIR=./.ccagent/claude")
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
