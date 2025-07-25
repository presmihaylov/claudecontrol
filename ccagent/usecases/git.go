package usecases

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"ccagent/clients"
	"ccagent/core/log"
)

type GitUseCase struct {
	gitClient    *clients.GitClient
	claudeClient *clients.ClaudeClient
}

func NewGitUseCase(gitClient *clients.GitClient, claudeClient *clients.ClaudeClient) *GitUseCase {
	return &GitUseCase{
		gitClient:    gitClient,
		claudeClient: claudeClient,
	}
}

func (g *GitUseCase) ValidateGitEnvironment() error {
	log.Info("📋 Starting to validate Git environment")

	// Check if we're in a Git repository
	if err := g.gitClient.IsGitRepository(); err != nil {
		log.Error("❌ Not in a Git repository", "error", err)
		return fmt.Errorf("ccagent must be run from within a Git repository: %w", err)
	}

	// Check if remote repository exists
	if err := g.gitClient.HasRemoteRepository(); err != nil {
		log.Error("❌ No remote repository configured", "error", err)
		return fmt.Errorf("Git repository must have a remote configured: %w", err)
	}

	// Check if GitHub CLI is available (for PR creation)
	if err := g.gitClient.IsGitHubCLIAvailable(); err != nil {
		log.Error("❌ GitHub CLI not available", "error", err)
		return fmt.Errorf("GitHub CLI (gh) must be installed and configured: %w", err)
	}

	log.Info("✅ Git environment validation passed")
	log.Info("📋 Completed successfully - validated Git environment")
	return nil
}

func (g *GitUseCase) PrepareForNewConversation(conversationHint string) error {
	log.Info("📋 Starting to prepare for new conversation")

	// Generate branch name using Claude
	branchName, err := g.generateBranchNameWithClaude(conversationHint)
	if err != nil {
		log.Error("❌ Failed to generate branch name with Claude, using fallback", "error", err)
		branchName = g.generateFallbackBranchName(conversationHint)
		// Note: We continue with fallback rather than failing completely
	}

	log.Info("🌿 Generated branch name: %s", branchName)

	// Step 1: Reset hard current branch
	if err := g.gitClient.ResetHard(); err != nil {
		log.Error("❌ Failed to reset hard", "error", err)
		return fmt.Errorf("failed to reset hard: %w", err)
	}

	// Step 2: Clean untracked files
	if err := g.gitClient.CleanUntracked(); err != nil {
		log.Error("❌ Failed to clean untracked files", "error", err)
		return fmt.Errorf("failed to clean untracked files: %w", err)
	}

	// Step 3: Get default branch and checkout to it
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch", "error", err)
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	if err := g.gitClient.CheckoutBranch(defaultBranch); err != nil {
		log.Error("❌ Failed to checkout default branch", "branch", defaultBranch, "error", err)
		return fmt.Errorf("failed to checkout default branch %s: %w", defaultBranch, err)
	}

	// Step 4: Pull latest changes
	if err := g.gitClient.PullLatest(); err != nil {
		log.Error("❌ Failed to pull latest changes", "error", err)
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	// Step 5: Create and checkout new branch
	if err := g.gitClient.CreateAndCheckoutBranch(branchName); err != nil {
		log.Error("❌ Failed to create and checkout new branch", "branch", branchName, "error", err)
		return fmt.Errorf("failed to create and checkout new branch %s: %w", branchName, err)
	}

	log.Info("✅ Successfully prepared for new conversation on branch: %s", branchName)
	log.Info("📋 Completed successfully - prepared for new conversation")
	return nil
}

func (g *GitUseCase) CompleteJobAndCreatePR() error {
	log.Info("📋 Starting to complete job and create PR")

	// Step 1: Check if there are any uncommitted changes
	hasChanges, err := g.gitClient.HasUncommittedChanges()
	if err != nil {
		log.Error("❌ Failed to check for uncommitted changes", "error", err)
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if !hasChanges {
		log.Info("ℹ️ No uncommitted changes found - aborting PR creation")
		log.Info("📋 Completed successfully - no changes to commit, PR creation skipped")
		return nil
	}

	log.Info("✅ Uncommitted changes detected - proceeding with commit and PR creation")

	// Step 2: Ensure .ccagent/ is in .gitignore
	if err := g.gitClient.EnsureCCAgentInGitignore(); err != nil {
		log.Error("❌ Failed to ensure .ccagent/ is in .gitignore", "error", err)
		return fmt.Errorf("failed to ensure .ccagent/ is in .gitignore: %w", err)
	}

	// Step 3: Get current branch
	currentBranch, err := g.gitClient.GetCurrentBranch()
	if err != nil {
		log.Error("❌ Failed to get current branch", "error", err)
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Generate commit message, PR title and body using Claude
	commitMessage, err := g.generateCommitMessageWithClaude(currentBranch)
	if err != nil {
		log.Error("❌ Failed to generate commit message with Claude, using fallback", "error", err)
		commitMessage = g.generateFallbackCommitMessage(currentBranch)
	}

	prTitle, err := g.generatePRTitleWithClaude(currentBranch)
	if err != nil {
		log.Error("❌ Failed to generate PR title with Claude, using fallback", "error", err)
		prTitle = g.generateFallbackPRTitle(currentBranch)
	}

	prBody, err := g.generatePRBodyWithClaude(currentBranch)
	if err != nil {
		log.Error("❌ Failed to generate PR body with Claude, using fallback", "error", err)
		prBody = g.generateFallbackPRBody(currentBranch)
	}

	log.Info("📝 Generated commit message: %s", commitMessage)
	log.Info("📋 Generated PR title: %s", prTitle)

	// Step 4: Add all changes
	if err := g.gitClient.AddAll(); err != nil {
		log.Error("❌ Failed to add all changes", "error", err)
		return fmt.Errorf("failed to add all changes: %w", err)
	}

	// Step 5: Commit with message
	if err := g.gitClient.Commit(commitMessage); err != nil {
		log.Error("❌ Failed to commit changes", "error", err)
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Step 6: Push current branch to remote
	if err := g.gitClient.PushBranch(currentBranch); err != nil {
		log.Error("❌ Failed to push branch", "branch", currentBranch, "error", err)
		return fmt.Errorf("failed to push branch %s: %w", currentBranch, err)
	}

	// Step 7: Get default branch for PR base
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch", "error", err)
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	// Step 8: Create pull request
	if err := g.gitClient.CreatePullRequest(prTitle, prBody, defaultBranch); err != nil {
		log.Error("❌ Failed to create pull request", "error", err)
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	log.Info("✅ Successfully completed job and created PR: %s", prTitle)
	log.Info("📋 Completed successfully - completed job and created PR")
	return nil
}

func (g *GitUseCase) generateBranchNameWithClaude(conversationHint string) (string, error) {
	log.Info("🤖 Asking Claude to generate branch name")

	prompt := fmt.Sprintf(`Based on this task description: "%s"

Please generate a short, descriptive Git branch name. Follow these rules:
- Use kebab-case (lowercase with hyphens)
- Maximum 30 characters
- Be concise but descriptive
- No special characters except hyphens
- Start with an action verb when possible

Respond with ONLY the branch name, nothing else.`, conversationHint)

	claudeBranchName, err := g.claudeClient.StartNewSession(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate branch name: %w", err)
	}

	// Clean up Claude's response
	cleanBranchName := g.cleanBranchName(claudeBranchName)

	// Add timestamp and prefix to ensure uniqueness
	timestamp := time.Now().Format("20060102-150405")
	finalBranchName := fmt.Sprintf("ccagent/%s-%s", cleanBranchName, timestamp)

	log.Info("🤖 Claude suggested: %s, final: %s", claudeBranchName, finalBranchName)
	return finalBranchName, nil
}

func (g *GitUseCase) generateCommitMessageWithClaude(branchName string) (string, error) {
	log.Info("🤖 Asking Claude to generate commit message")

	prompt := fmt.Sprintf(`I'm completing work on Git branch: "%s"

Please generate a concise Git commit message. Follow these rules:
- Start with an action verb (Add, Fix, Update, Implement, etc.)
- Be descriptive but concise
- Use imperative mood
- Maximum 50 characters for the title
- End with a simple note that this was done by Claude Control

Format:
<title>

🤖 Generated with Claude Control

Respond with ONLY the commit message, nothing else.`, branchName)

	commitMessage, err := g.claudeClient.StartNewSession(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate commit message: %w", err)
	}

	return strings.TrimSpace(commitMessage), nil
}

func (g *GitUseCase) generatePRTitleWithClaude(branchName string) (string, error) {
	log.Info("🤖 Asking Claude to generate PR title")

	prompt := fmt.Sprintf(`I'm creating a pull request for Git branch: "%s"

Please generate a clear, descriptive pull request title. Follow these rules:
- Use title case
- Be descriptive but concise
- Maximum 60 characters
- Focus on what was accomplished
- Don't mention "Claude" or "agent" in the title

Respond with ONLY the PR title, nothing else.`, branchName)

	prTitle, err := g.claudeClient.StartNewSession(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR title: %w", err)
	}

	return strings.TrimSpace(prTitle), nil
}

func (g *GitUseCase) generatePRBodyWithClaude(branchName string) (string, error) {
	log.Info("🤖 Asking Claude to generate PR body")

	prompt := fmt.Sprintf(`I'm creating a pull request for Git branch: "%s"

Please generate a professional pull request description. Include:
- ## Summary section with what was accomplished
- ## Changes section with key modifications
- Note that this was generated by Claude Control
- Keep it concise but informative

Use proper markdown formatting.

Respond with ONLY the PR body, nothing else.`, branchName)

	prBody, err := g.claudeClient.StartNewSession(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR body: %w", err)
	}

	return strings.TrimSpace(prBody), nil
}

func (g *GitUseCase) cleanBranchName(branchName string) string {
	// Remove any quotes or extra whitespace
	cleaned := strings.TrimSpace(branchName)
	cleaned = strings.Trim(cleaned, "\"'`")

	// Convert to lowercase and replace spaces/underscores with hyphens
	cleaned = strings.ToLower(cleaned)
	cleaned = strings.ReplaceAll(cleaned, " ", "-")
	cleaned = strings.ReplaceAll(cleaned, "_", "-")

	// Remove any characters that aren't alphanumeric or hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	cleaned = reg.ReplaceAllString(cleaned, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	cleaned = reg.ReplaceAllString(cleaned, "-")

	// Remove leading/trailing hyphens
	cleaned = strings.Trim(cleaned, "-")

	// Limit length
	if len(cleaned) > 30 {
		cleaned = cleaned[:30]
		cleaned = strings.TrimRight(cleaned, "-")
	}

	// Ensure we have something valid
	if len(cleaned) < 3 {
		cleaned = "claude-task"
	}

	return cleaned
}

// Fallback methods for when Claude fails
func (g *GitUseCase) generateFallbackBranchName(conversationHint string) string {
	cleaned := g.cleanBranchName(conversationHint)
	if len(cleaned) < 3 {
		cleaned = "claude-task"
	}
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("ccagent/%s-%s", cleaned, timestamp)
}

func (g *GitUseCase) generateFallbackCommitMessage(branchName string) string {
	return "Complete task\n\n🤖 Generated with Claude Control"
}

func (g *GitUseCase) generateFallbackPRTitle(branchName string) string {
	return "Complete Claude Control Task"
}

func (g *GitUseCase) generateFallbackPRBody(branchName string) string {
	return fmt.Sprintf(`## Summary
Completed task via Claude Control.

## Changes
- Task implementation completed
- All changes committed and ready for review

**Branch:** %s  
**Generated:** %s

🤖 Generated with Claude Control`, branchName, time.Now().Format("2006-01-02 15:04:05"))
}

