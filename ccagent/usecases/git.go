package usecases

import (
	"fmt"
	"strings"
	"time"

	"ccagent/clients"
	"ccagent/core/log"

	"github.com/google/uuid"
	"github.com/lucasepe/codename"
)

type GitUseCase struct {
	gitClient    *clients.GitClient
	claudeClient *clients.ClaudeClient
}

type AutoCommitResult struct {
	JustCreatedPR    bool
	PullRequestLink string
	CommitHash      string
	RepositoryURL   string
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

	// Generate random branch name
	branchName, err := g.generateRandomBranchName()
	if err != nil {
		log.Error("❌ Failed to generate random branch name", "error", err)
		return fmt.Errorf("failed to generate branch name: %w", err)
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

func (g *GitUseCase) AutoCommitChangesIfNeeded(slackThreadLink string) (*AutoCommitResult, error) {
	log.Info("📋 Starting to auto-commit changes if needed")

	// Check if there are any uncommitted changes
	hasChanges, err := g.gitClient.HasUncommittedChanges()
	if err != nil {
		log.Error("❌ Failed to check for uncommitted changes", "error", err)
		return nil, fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if !hasChanges {
		log.Info("ℹ️ No uncommitted changes found - skipping auto-commit")
		log.Info("📋 Completed successfully - no changes to commit")
		return &AutoCommitResult{
			JustCreatedPR:    false,
			PullRequestLink: "",
			CommitHash:      "",
			RepositoryURL:   "",
		}, nil
	}

	log.Info("✅ Uncommitted changes detected - proceeding with auto-commit")

	// Ensure .ccagent/ is in .gitignore
	if err := g.gitClient.EnsureCCAgentInGitignore(); err != nil {
		log.Error("❌ Failed to ensure .ccagent/ is in .gitignore", "error", err)
		return nil, fmt.Errorf("failed to ensure .ccagent/ is in .gitignore: %w", err)
	}

	// Get current branch
	currentBranch, err := g.gitClient.GetCurrentBranch()
	if err != nil {
		log.Error("❌ Failed to get current branch", "error", err)
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Generate commit message using Claude with isolated config directory
	commitMessage, err := g.generateCommitMessageWithClaudeIsolated(currentBranch)
	if err != nil {
		log.Error("❌ Failed to generate commit message with Claude, using fallback", "error", err)
		commitMessage = g.generateFallbackCommitMessage(currentBranch)
	}

	log.Info("📝 Generated commit message: %s", commitMessage)

	// Add all changes
	if err := g.gitClient.AddAll(); err != nil {
		log.Error("❌ Failed to add all changes", "error", err)
		return nil, fmt.Errorf("failed to add all changes: %w", err)
	}

	// Commit with message
	if err := g.gitClient.Commit(commitMessage); err != nil {
		log.Error("❌ Failed to commit changes", "error", err)
		return nil, fmt.Errorf("failed to commit changes: %w", err)
	}

	// Get commit hash after successful commit
	commitHash, err := g.gitClient.GetLatestCommitHash()
	if err != nil {
		log.Error("❌ Failed to get commit hash", "error", err)
		return nil, fmt.Errorf("failed to get commit hash: %w", err)
	}

	// Get repository URL for commit link
	repositoryURL, err := g.gitClient.GetRemoteURL()
	if err != nil {
		log.Error("❌ Failed to get repository URL", "error", err)
		return nil, fmt.Errorf("failed to get repository URL: %w", err)
	}

	// Push current branch to remote
	if err := g.gitClient.PushBranch(currentBranch); err != nil {
		log.Error("❌ Failed to push branch", "branch", currentBranch, "error", err)
		return nil, fmt.Errorf("failed to push branch %s: %w", currentBranch, err)
	}

	// Handle PR creation/update
	prResult, err := g.handlePRCreationOrUpdate(currentBranch, slackThreadLink)
	if err != nil {
		log.Error("❌ Failed to handle PR creation/update", "error", err)
		return nil, fmt.Errorf("failed to handle PR creation/update: %w", err)
	}

	// Update the result with commit information
	prResult.CommitHash = commitHash
	prResult.RepositoryURL = repositoryURL

	log.Info("✅ Successfully auto-committed and pushed changes")
	log.Info("📋 Completed successfully - auto-committed changes")
	return prResult, nil
}

func (g *GitUseCase) generateRandomBranchName() (string, error) {
	log.Info("🎲 Generating random branch name")

	rng, err := codename.DefaultRNG()
	if err != nil {
		return "", fmt.Errorf("failed to create random generator: %w", err)
	}

	randomName := codename.Generate(rng, 0)
	timestamp := time.Now().Format("20060102-150405")
	finalBranchName := fmt.Sprintf("ccagent/%s-%s", randomName, timestamp)

	log.Info("🎲 Generated random name: %s", finalBranchName)
	return finalBranchName, nil
}

func (g *GitUseCase) generateFallbackCommitMessage(branchName string) string {
	return "Complete task\n\n🤖 Generated with Claude Control"
}

func (g *GitUseCase) generateFallbackPRTitle(branchName string) string {
	return "Complete Claude Control Task"
}

func (g *GitUseCase) generateFallbackPRBody(branchName, slackThreadLink string) string {
	body := fmt.Sprintf(`## Summary
Completed task via Claude Control.

## Changes
- Task implementation completed
- All changes committed and ready for review

**Branch:** %s  
**Generated:** %s`, branchName, time.Now().Format("2006-01-02 15:04:05"))

	// Append footer with Slack thread link
	body += fmt.Sprintf("\n\n---\nGenerated with Claude Control from [this slack thread](%s)", slackThreadLink)
	
	return body
}

func (g *GitUseCase) generateCommitMessageWithClaudeIsolated(branchName string) (string, error) {
	log.Info("🤖 Asking Claude to generate commit message with isolated config")

	// Generate unique config directory using UUID
	configDir := fmt.Sprintf(".ccagent/git-%s", uuid.New().String())

	prompt := fmt.Sprintf(`I'm completing work on Git branch: "%s"

CRITICAL INSTRUCTIONS - READ CAREFULLY:
1. You MUST respond with ONLY the commit message text
2. NO explanations, NO additional text, NO formatting markup
3. NO "Here is the commit message:" or similar phrases
4. Maximum 50 characters total (STRICT LIMIT)
5. Start with action verb (Add, Fix, Update, etc.)
6. Use imperative mood

FORMAT EXAMPLE:
Fix user authentication validation

YOUR RESPONSE MUST BE THE COMMIT MESSAGE ONLY.`, branchName)

	commitMessage, err := g.claudeClient.StartNewSessionWithConfigDir(prompt, configDir)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate commit message: %w", err)
	}

	return strings.TrimSpace(commitMessage), nil
}

func (g *GitUseCase) handlePRCreationOrUpdate(branchName, slackThreadLink string) (*AutoCommitResult, error) {
	log.Info("📋 Starting to handle PR creation or update for branch: %s", branchName)

	// Check if a PR already exists for this branch
	hasExistingPR, err := g.gitClient.HasExistingPR(branchName)
	if err != nil {
		log.Error("❌ Failed to check for existing PR", "error", err)
		return nil, fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if hasExistingPR {
		log.Info("✅ Existing PR found for branch %s - changes have been pushed", branchName)

		// Get the PR URL for the existing PR
		prURL, err := g.gitClient.GetPRURL(branchName)
		if err != nil {
			log.Error("❌ Failed to get PR URL for existing PR", "error", err)
			// Continue without the URL rather than failing
			prURL = ""
		}

		log.Info("📋 Completed successfully - updated existing PR")
		return &AutoCommitResult{
			JustCreatedPR:    false,
			PullRequestLink: prURL,
			CommitHash:      "", // Will be filled in by caller
			RepositoryURL:   "", // Will be filled in by caller
		}, nil
	}

	log.Info("🆕 No existing PR found - creating new PR")

	// Generate PR title and body using Claude with isolated config directories
	prTitle, err := g.generatePRTitleWithClaudeIsolated(branchName)
	if err != nil {
		log.Error("❌ Failed to generate PR title with Claude, using fallback", "error", err)
		prTitle = g.generateFallbackPRTitle(branchName)
	}

	prBody, err := g.generatePRBodyWithClaudeIsolated(branchName, slackThreadLink)
	if err != nil {
		log.Error("❌ Failed to generate PR body with Claude, using fallback", "error", err)
		prBody = g.generateFallbackPRBody(branchName, slackThreadLink)
	}

	log.Info("📋 Generated PR title: %s", prTitle)

	// Get default branch for PR base
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch", "error", err)
		return nil, fmt.Errorf("failed to get default branch: %w", err)
	}

	// Create pull request
	prURL, err := g.gitClient.CreatePullRequest(prTitle, prBody, defaultBranch)
	if err != nil {
		log.Error("❌ Failed to create pull request", "error", err)
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	log.Info("✅ Successfully created PR: %s", prTitle)
	log.Info("📋 Completed successfully - created new PR")
	return &AutoCommitResult{
		JustCreatedPR:    true,
		PullRequestLink: prURL,
		CommitHash:      "", // Will be filled in by caller
		RepositoryURL:   "", // Will be filled in by caller
	}, nil
}

func (g *GitUseCase) generatePRTitleWithClaudeIsolated(branchName string) (string, error) {
	log.Info("🤖 Asking Claude to generate PR title with isolated config")

	// Generate unique config directory using UUID
	configDir := fmt.Sprintf(".ccagent/git-%s", uuid.New().String())

	prompt := fmt.Sprintf(`I'm creating a pull request for Git branch: "%s"

Generate a SHORT pull request title. Follow these strict rules:
- Maximum 40 characters (STRICT LIMIT)
- Start with action verb (Add, Fix, Update, Improve, etc.)
- Be concise and specific
- No unnecessary words or phrases
- Don't mention "Claude", "agent", or implementation details

Examples:
- "Fix error handling in message processor"
- "Add user authentication middleware"
- "Update API response format"

Respond with ONLY the short title, nothing else.`, branchName)

	prTitle, err := g.claudeClient.StartNewSessionWithConfigDir(prompt, configDir)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR title: %w", err)
	}

	return strings.TrimSpace(prTitle), nil
}

func (g *GitUseCase) generatePRBodyWithClaudeIsolated(branchName, slackThreadLink string) (string, error) {
	log.Info("🤖 Asking Claude to generate PR body with isolated config")

	// Generate unique config directory using UUID
	configDir := fmt.Sprintf(".ccagent/git-%s", uuid.New().String())

	prompt := fmt.Sprintf(`I'm creating a pull request for Git branch: "%s"

Please generate a professional pull request description. Include:
- ## Summary section with what was accomplished
- ## Changes section with key modifications
- Keep it concise but informative

Use proper markdown formatting.

IMPORTANT: Do NOT include any "Generated with Claude Control" or similar footer text. I will add that separately.

Respond with ONLY the PR body, nothing else.`, branchName)

	prBody, err := g.claudeClient.StartNewSessionWithConfigDir(prompt, configDir)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR body: %w", err)
	}

	// Append footer with Slack thread link
	cleanBody := strings.TrimSpace(prBody)
	finalBody := cleanBody + fmt.Sprintf("\n\n---\nGenerated with Claude Control from [this slack thread](%s)", slackThreadLink)

	return finalBody, nil
}

func (g *GitUseCase) ValidateAndRestorePRDescriptionFooter(slackThreadLink string) error {
	log.Info("📋 Starting to validate and restore PR description footer")

	// Get current branch
	currentBranch, err := g.gitClient.GetCurrentBranch()
	if err != nil {
		log.Error("❌ Failed to get current branch", "error", err)
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if a PR exists for this branch
	hasExistingPR, err := g.gitClient.HasExistingPR(currentBranch)
	if err != nil {
		log.Error("❌ Failed to check for existing PR", "error", err)
		return fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if !hasExistingPR {
		log.Info("ℹ️ No existing PR found - skipping footer validation")
		log.Info("📋 Completed successfully - no PR to validate")
		return nil
	}

	// Get current PR description
	currentDescription, err := g.gitClient.GetPRDescription(currentBranch)
	if err != nil {
		log.Error("❌ Failed to get PR description", "error", err)
		return fmt.Errorf("failed to get PR description: %w", err)
	}

	// Check if the expected footer is present
	expectedFooter := fmt.Sprintf("Generated with Claude Control from [this slack thread](%s)", slackThreadLink)
	
	if strings.Contains(currentDescription, expectedFooter) {
		log.Info("✅ PR description already has correct Claude Control footer")
		log.Info("📋 Completed successfully - footer validation passed")
		return nil
	}

	log.Info("🔧 PR description missing Claude Control footer - restoring it")

	// Remove any existing footer lines to avoid duplicates
	lines := strings.Split(currentDescription, "\n")
	var cleanedLines []string
	
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// Skip existing footer lines
		if strings.Contains(trimmedLine, "Generated with Claude Control") ||
		   strings.Contains(trimmedLine, "Generated with Claude Code") ||
		   (trimmedLine == "---" && len(cleanedLines) > 0 && 
		    strings.Contains(strings.Join(cleanedLines[len(cleanedLines)-5:], " "), "Generated with")) {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}

	// Remove trailing empty lines
	for len(cleanedLines) > 0 && strings.TrimSpace(cleanedLines[len(cleanedLines)-1]) == "" {
		cleanedLines = cleanedLines[:len(cleanedLines)-1]
	}

	// Add the correct footer
	restoredDescription := strings.Join(cleanedLines, "\n")
	if restoredDescription != "" {
		restoredDescription += "\n\n---\n" + expectedFooter
	} else {
		restoredDescription = "---\n" + expectedFooter
	}

	// Update the PR description
	if err := g.gitClient.UpdatePRDescription(currentBranch, restoredDescription); err != nil {
		log.Error("❌ Failed to update PR description", "error", err)
		return fmt.Errorf("failed to update PR description: %w", err)
	}

	log.Info("✅ Successfully restored Claude Control footer to PR description")
	log.Info("📋 Completed successfully - restored PR description footer")
	return nil
}
