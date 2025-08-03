package usecases

import (
	"fmt"
	"strings"
	"time"

	"ccagent/clients"
	"ccagent/core/log"
	"ccagent/services"

	"github.com/lucasepe/codename"
)

type GitUseCase struct {
	gitClient     *clients.GitClient
	claudeService *services.ClaudeService
}

type AutoCommitResult struct {
	JustCreatedPR   bool
	PullRequestLink string
	PullRequestID   string // GitHub PR number (e.g., "123")
	CommitHash      string
	RepositoryURL   string
	BranchName      string
}

func NewGitUseCase(gitClient *clients.GitClient, claudeService *services.ClaudeService) *GitUseCase {
	return &GitUseCase{
		gitClient:     gitClient,
		claudeService: claudeService,
	}
}

func (g *GitUseCase) ValidateGitEnvironment() error {
	log.Info("📋 Starting to validate Git environment")

	// Check if we're in a Git repository
	if err := g.gitClient.IsGitRepository(); err != nil {
		log.Error("❌ Not in a Git repository: %v", err)
		return fmt.Errorf("ccagent must be run from within a Git repository: %w", err)
	}

	// Check if we're at the Git repository root
	if err := g.gitClient.IsGitRepositoryRoot(); err != nil {
		log.Error("❌ Not at Git repository root: %v", err)
		return fmt.Errorf("ccagent must be run from the Git repository root: %w", err)
	}

	// Check if remote repository exists
	if err := g.gitClient.HasRemoteRepository(); err != nil {
		log.Error("❌ No remote repository configured: %v", err)
		return fmt.Errorf("Git repository must have a remote configured: %w", err)
	}

	// Check if GitHub CLI is available (for PR creation)
	if err := g.gitClient.IsGitHubCLIAvailable(); err != nil {
		log.Error("❌ GitHub CLI not available: %v", err)
		return fmt.Errorf("GitHub CLI (gh) must be installed and configured: %w", err)
	}

	log.Info("✅ Git environment validation passed")
	log.Info("📋 Completed successfully - validated Git environment")
	return nil
}

// SwitchToJobBranch switches to the specified branch, discarding local changes and pulling latest from main
func (g *GitUseCase) SwitchToJobBranch(branchName string) error {
	log.Info("📋 Starting to switch to job branch: %s", branchName)

	// Step 1: Reset hard current branch to discard uncommitted changes
	if err := g.gitClient.ResetHard(); err != nil {
		log.Error("❌ Failed to reset hard: %v", err)
		return fmt.Errorf("failed to reset hard: %w", err)
	}

	// Step 2: Clean untracked files
	if err := g.gitClient.CleanUntracked(); err != nil {
		log.Error("❌ Failed to clean untracked files: %v", err)
		return fmt.Errorf("failed to clean untracked files: %w", err)
	}

	// Step 3: Get default branch and checkout to it
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch: %v", err)
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	if err := g.gitClient.CheckoutBranch(defaultBranch); err != nil {
		log.Error("❌ Failed to checkout default branch %s: %v", defaultBranch, err)
		return fmt.Errorf("failed to checkout default branch %s: %w", defaultBranch, err)
	}

	// Step 4: Pull latest changes
	if err := g.gitClient.PullLatest(); err != nil {
		log.Error("❌ Failed to pull latest changes: %v", err)
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	// Step 5: Checkout target branch
	if err := g.gitClient.CheckoutBranch(branchName); err != nil {
		log.Error("❌ Failed to checkout target branch %s: %v", branchName, err)
		return fmt.Errorf("failed to checkout target branch %s: %w", branchName, err)
	}

	log.Info("✅ Successfully switched to job branch: %s", branchName)
	log.Info("📋 Completed successfully - switched to job branch")
	return nil
}

func (g *GitUseCase) PrepareForNewConversation(conversationHint string) (string, error) {
	log.Info("📋 Starting to prepare for new conversation")

	// Generate random branch name
	branchName, err := g.generateRandomBranchName()
	if err != nil {
		log.Error("❌ Failed to generate random branch name: %v", err)
		return "", fmt.Errorf("failed to generate branch name: %w", err)
	}

	log.Info("🌿 Generated branch name: %s", branchName)

	// Use the common branch switching logic to get to main and pull latest
	if err := g.resetAndPullDefaultBranch(); err != nil {
		log.Error("❌ Failed to reset and pull main: %v", err)
		return "", fmt.Errorf("failed to reset and pull main: %w", err)
	}

	// Create and checkout new branch
	if err := g.gitClient.CreateAndCheckoutBranch(branchName); err != nil {
		log.Error("❌ Failed to create and checkout new branch %s: %v", branchName, err)
		return "", fmt.Errorf("failed to create and checkout new branch %s: %w", branchName, err)
	}

	log.Info("✅ Successfully prepared for new conversation on branch: %s", branchName)
	log.Info("📋 Completed successfully - prepared for new conversation")
	return branchName, nil
}

// resetAndPullDefaultBranch is a helper that resets current branch, goes to main, and pulls latest
func (g *GitUseCase) resetAndPullDefaultBranch() error {
	log.Info("📋 Starting to reset and pull default branch")

	// Step 1: Reset hard current branch to discard uncommitted changes
	if err := g.gitClient.ResetHard(); err != nil {
		log.Error("❌ Failed to reset hard: %v", err)
		return fmt.Errorf("failed to reset hard: %w", err)
	}

	// Step 2: Clean untracked files
	if err := g.gitClient.CleanUntracked(); err != nil {
		log.Error("❌ Failed to clean untracked files: %v", err)
		return fmt.Errorf("failed to clean untracked files: %w", err)
	}

	// Step 3: Get default branch and checkout to it
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch: %v", err)
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	if err := g.gitClient.CheckoutBranch(defaultBranch); err != nil {
		log.Error("❌ Failed to checkout default branch %s: %v", defaultBranch, err)
		return fmt.Errorf("failed to checkout default branch %s: %w", defaultBranch, err)
	}

	// Step 4: Pull latest changes
	if err := g.gitClient.PullLatest(); err != nil {
		log.Error("❌ Failed to pull latest changes: %v", err)
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	log.Info("✅ Successfully reset and pulled main")
	log.Info("📋 Completed successfully - reset and pulled main")
	return nil
}

func (g *GitUseCase) AutoCommitChangesIfNeeded(slackThreadLink string) (*AutoCommitResult, error) {
	log.Info("📋 Starting to auto-commit changes if needed")

	// Get current branch first (needed for both cases)
	currentBranch, err := g.gitClient.GetCurrentBranch()
	if err != nil {
		log.Error("❌ Failed to get current branch: %v", err)
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if there are any uncommitted changes
	hasChanges, err := g.gitClient.HasUncommittedChanges()
	if err != nil {
		log.Error("❌ Failed to check for uncommitted changes: %v", err)
		return nil, fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if !hasChanges {
		log.Info("ℹ️ No uncommitted changes found - skipping auto-commit")
		log.Info("📋 Completed successfully - no changes to commit")
		return &AutoCommitResult{
			JustCreatedPR:   false,
			PullRequestLink: "",
			PullRequestID:   "",
			CommitHash:      "",
			RepositoryURL:   "",
			BranchName:      currentBranch,
		}, nil
	}

	log.Info("✅ Uncommitted changes detected - proceeding with auto-commit")


	// Generate commit message using Claude
	commitMessage, err := g.generateCommitMessageWithClaude(currentBranch)
	if err != nil {
		log.Error("❌ Failed to generate commit message with Claude: %v", err)
		return nil, fmt.Errorf("failed to generate commit message with Claude: %w", err)
	}

	log.Info("📝 Generated commit message: %s", commitMessage)

	// Add all changes
	if err := g.gitClient.AddAll(); err != nil {
		log.Error("❌ Failed to add all changes: %v", err)
		return nil, fmt.Errorf("failed to add all changes: %w", err)
	}

	// Commit with message
	if err := g.gitClient.Commit(commitMessage); err != nil {
		log.Error("❌ Failed to commit changes: %v", err)
		return nil, fmt.Errorf("failed to commit changes: %w", err)
	}

	// Get commit hash after successful commit
	commitHash, err := g.gitClient.GetLatestCommitHash()
	if err != nil {
		log.Error("❌ Failed to get commit hash: %v", err)
		return nil, fmt.Errorf("failed to get commit hash: %w", err)
	}

	// Get repository URL for commit link
	repositoryURL, err := g.gitClient.GetRemoteURL()
	if err != nil {
		log.Error("❌ Failed to get repository URL: %v", err)
		return nil, fmt.Errorf("failed to get repository URL: %w", err)
	}

	// Push current branch to remote
	if err := g.gitClient.PushBranch(currentBranch); err != nil {
		log.Error("❌ Failed to push branch %s: %v", currentBranch, err)
		return nil, fmt.Errorf("failed to push branch %s: %w", currentBranch, err)
	}

	// Handle PR creation/update
	prResult, err := g.handlePRCreationOrUpdate(currentBranch, slackThreadLink)
	if err != nil {
		log.Error("❌ Failed to handle PR creation/update: %v", err)
		return nil, fmt.Errorf("failed to handle PR creation/update: %w", err)
	}

	// Update the result with commit information
	prResult.CommitHash = commitHash
	prResult.RepositoryURL = repositoryURL

	// Extract and store PR ID from the PR URL if available
	if prResult.PullRequestLink != "" {
		prResult.PullRequestID = g.gitClient.ExtractPRIDFromURL(prResult.PullRequestLink)
	}

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

func (g *GitUseCase) generateCommitMessageWithClaude(branchName string) (string, error) {
	log.Info("🤖 Asking Claude to generate commit message")

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

	result, err := g.claudeService.StartNewConversation(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate commit message: %w", err)
	}

	return strings.TrimSpace(result.Output), nil
}

func (g *GitUseCase) handlePRCreationOrUpdate(branchName, slackThreadLink string) (*AutoCommitResult, error) {
	log.Info("📋 Starting to handle PR creation or update for branch: %s", branchName)

	// Check if a PR already exists for this branch
	hasExistingPR, err := g.gitClient.HasExistingPR(branchName)
	if err != nil {
		log.Error("❌ Failed to check for existing PR: %v", err)
		return nil, fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if hasExistingPR {
		log.Info("✅ Existing PR found for branch %s - changes have been pushed", branchName)

		// Get the PR URL for the existing PR
		prURL, err := g.gitClient.GetPRURL(branchName)
		if err != nil {
			log.Error("❌ Failed to get PR URL for existing PR: %v", err)
			// Continue without the URL rather than failing
			prURL = ""
		}

		log.Info("📋 Completed successfully - updated existing PR")
		return &AutoCommitResult{
			JustCreatedPR:   false,
			PullRequestLink: prURL,
			PullRequestID:   g.gitClient.ExtractPRIDFromURL(prURL),
			CommitHash:      "", // Will be filled in by caller
			RepositoryURL:   "", // Will be filled in by caller
			BranchName:      branchName,
		}, nil
	}

	log.Info("🆕 No existing PR found - creating new PR")

	// Generate PR title and body using Claude
	prTitle, err := g.generatePRTitleWithClaude(branchName)
	if err != nil {
		log.Error("❌ Failed to generate PR title with Claude: %v", err)
		return nil, fmt.Errorf("failed to generate PR title with Claude: %w", err)
	}

	prBody, err := g.generatePRBodyWithClaude(branchName, slackThreadLink)
	if err != nil {
		log.Error("❌ Failed to generate PR body with Claude: %v", err)
		return nil, fmt.Errorf("failed to generate PR body with Claude: %w", err)
	}

	log.Info("📋 Generated PR title: %s", prTitle)

	// Get default branch for PR base
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch: %v", err)
		return nil, fmt.Errorf("failed to get default branch: %w", err)
	}

	// Create pull request
	prURL, err := g.gitClient.CreatePullRequest(prTitle, prBody, defaultBranch)
	if err != nil {
		log.Error("❌ Failed to create pull request: %v", err)
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	log.Info("✅ Successfully created PR: %s", prTitle)
	log.Info("📋 Completed successfully - created new PR")
	return &AutoCommitResult{
		JustCreatedPR:   true,
		PullRequestLink: prURL,
		PullRequestID:   g.gitClient.ExtractPRIDFromURL(prURL),
		CommitHash:      "", // Will be filled in by caller
		RepositoryURL:   "", // Will be filled in by caller
		BranchName:      branchName,
	}, nil
}

func (g *GitUseCase) generatePRTitleWithClaude(branchName string) (string, error) {
	log.Info("🤖 Asking Claude to generate PR title")

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

	result, err := g.claudeService.StartNewConversation(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR title: %w", err)
	}

	return strings.TrimSpace(result.Output), nil
}

func (g *GitUseCase) generatePRBodyWithClaude(branchName, slackThreadLink string) (string, error) {
	log.Info("🤖 Asking Claude to generate PR body")

	prompt := fmt.Sprintf(`I'm creating a pull request for Git branch: "%s"

Please generate a professional pull request description. Include:
- ## Summary section with what was accomplished
- ## Changes section with key modifications
- Keep it concise but informative

Use proper markdown formatting.

IMPORTANT: Do NOT include any "Generated with Claude Control" or similar footer text. I will add that separately.

Respond with ONLY the PR body, nothing else.`, branchName)

	result, err := g.claudeService.StartNewConversation(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR body: %w", err)
	}

	// Append footer with Slack thread link (backend sends proper deep links)
	cleanBody := strings.TrimSpace(result.Output)
	finalBody := cleanBody + fmt.Sprintf("\n\n---\nGenerated with [Claude Control](https://claudecontrol.com) from this [slack thread](%s)", slackThreadLink)

	return finalBody, nil
}

func (g *GitUseCase) ValidateAndRestorePRDescriptionFooter(slackThreadLink string) error {
	log.Info("📋 Starting to validate and restore PR description footer")

	// Get current branch
	currentBranch, err := g.gitClient.GetCurrentBranch()
	if err != nil {
		log.Error("❌ Failed to get current branch: %v", err)
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if a PR exists for this branch
	hasExistingPR, err := g.gitClient.HasExistingPR(currentBranch)
	if err != nil {
		log.Error("❌ Failed to check for existing PR: %v", err)
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
		log.Error("❌ Failed to get PR description: %v", err)
		return fmt.Errorf("failed to get PR description: %w", err)
	}

	// Check if the expected footer is present (backend sends proper deep links)
	expectedFooter := fmt.Sprintf("Generated with [Claude Control](https://claudecontrol.com) from this [slack thread](%s)", slackThreadLink)

	if strings.Contains(currentDescription, expectedFooter) {
		log.Info("✅ PR description already has correct Claude Control footer")
		log.Info("📋 Completed successfully - footer validation passed")
		return nil
	}

	log.Info("🔧 PR description missing Claude Control footer - restoring it")

	// Remove any existing footer lines to avoid duplicates
	lines := strings.Split(currentDescription, "\n")
	var cleanedLines []string
	inFooterSection := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we've hit a footer section
		if strings.Contains(trimmedLine, "Generated with Claude Control") ||
			strings.Contains(trimmedLine, "Generated with Claude Code") {
			inFooterSection = true
			continue
		}

		// Skip separator lines that are part of footer
		if trimmedLine == "---" {
			// Look ahead to see if this separator is followed by footer content
			isFooterSeparator := false
			for i := len(cleanedLines); i < len(lines)-1; i++ {
				nextLine := strings.TrimSpace(lines[i+1])
				if nextLine == "" {
					continue
				}
				if strings.Contains(nextLine, "Generated with Claude") {
					isFooterSeparator = true
				}
				break
			}

			if isFooterSeparator || inFooterSection {
				continue
			}
		}

		// Skip empty lines in footer section
		if inFooterSection && trimmedLine == "" {
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
		// Check if description already ends with a separator
		if strings.HasSuffix(strings.TrimSpace(restoredDescription), "---") {
			restoredDescription += "\n" + expectedFooter
		} else {
			restoredDescription += "\n\n---\n" + expectedFooter
		}
	} else {
		restoredDescription = "---\n" + expectedFooter
	}

	// Update the PR description
	if err := g.gitClient.UpdatePRDescription(currentBranch, restoredDescription); err != nil {
		log.Error("❌ Failed to update PR description: %v", err)
		return fmt.Errorf("failed to update PR description: %w", err)
	}

	log.Info("✅ Successfully restored Claude Control footer to PR description")
	log.Info("📋 Completed successfully - restored PR description footer")
	return nil
}

func (g *GitUseCase) CheckPRStatus(branchName string) (string, error) {
	log.Info("📋 Starting to check PR status for branch: %s", branchName)

	// First check if a PR exists for this branch
	hasExistingPR, err := g.gitClient.HasExistingPR(branchName)
	if err != nil {
		log.Error("❌ Failed to check for existing PR for branch %s: %v", branchName, err)
		return "", fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if !hasExistingPR {
		log.Info("📋 No PR found for branch %s", branchName)
		return "no_pr", nil
	}

	// Get PR status using GitHub CLI
	prStatus, err := g.gitClient.GetPRState(branchName)
	if err != nil {
		log.Error("❌ Failed to get PR state for branch %s: %v", branchName, err)
		return "", fmt.Errorf("failed to get PR state: %w", err)
	}

	log.Info("📋 Completed successfully - PR status for branch %s: %s", branchName, prStatus)
	return prStatus, nil
}

func (g *GitUseCase) CheckPRStatusByID(prID string) (string, error) {
	log.Info("📋 Starting to check PR status by ID: %s", prID)

	// Get PR status directly by PR ID using GitHub CLI
	prStatus, err := g.gitClient.GetPRStateByID(prID)
	if err != nil {
		log.Error("❌ Failed to get PR state for PR ID %s: %v", prID, err)
		return "", fmt.Errorf("failed to get PR state by ID: %w", err)
	}

	log.Info("📋 Completed successfully - PR status for ID %s: %s", prID, prStatus)
	return prStatus, nil
}
