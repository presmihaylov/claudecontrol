package usecases

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lucasepe/codename"

	"ccagent/clients"
	"ccagent/core/log"
	"ccagent/models"
	"ccagent/services"
)

type GitUseCase struct {
	gitClient     *clients.GitClient
	claudeService *services.ClaudeService
	appState      *models.AppState
}

type AutoCommitResult struct {
	JustCreatedPR   bool
	PullRequestLink string
	PullRequestID   string // GitHub PR number (e.g., "123")
	CommitHash      string
	RepositoryURL   string
	BranchName      string
}

func NewGitUseCase(
	gitClient *clients.GitClient,
	claudeService *services.ClaudeService,
	appState *models.AppState,
) *GitUseCase {
	return &GitUseCase{
		gitClient:     gitClient,
		claudeService: claudeService,
		appState:      appState,
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
		return fmt.Errorf("git repository must have a remote configured: %w", err)
	}

	// Check if GitHub CLI is available (for PR creation)
	if err := g.gitClient.IsGitHubCLIAvailable(); err != nil {
		log.Error("❌ GitHub CLI not available: %v", err)
		return fmt.Errorf("GitHub CLI (gh) must be installed and configured: %w", err)
	}

	// Validate remote repository access credentials
	if err := g.gitClient.ValidateRemoteAccess(); err != nil {
		log.Error("❌ Remote repository access validation failed: %v", err)
		return fmt.Errorf("remote repository access validation failed: %w", err)
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

		// Update PR title and description based on new changes
		if err := g.updatePRTitleAndDescriptionIfNeeded(branchName, slackThreadLink); err != nil {
			log.Error("❌ Failed to update PR title/description: %v", err)
			// Log error but don't fail the entire operation
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

	// Get default branch to compare against
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get commit messages specific to this branch
	commitMessages, err := g.gitClient.GetBranchCommitMessages(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit messages: %w", err)
	}

	// Get diff summary
	diffSummary, err := g.gitClient.GetBranchDiffSummary(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff summary: %w", err)
	}

	// Get actual diff content for better context
	diffContent, err := g.gitClient.GetBranchDiffContent(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff content: %w", err)
	}

	// Build commit info for context
	commitInfo := "No commits found"
	if len(commitMessages) > 0 {
		commitInfo = fmt.Sprintf("Recent commits:\n%s", strings.Join(commitMessages, "\n"))
	}

	prompt := fmt.Sprintf(`I'm creating a pull request for Git branch: "%s"

Here are the commits made on this branch (not including main branch commits):
%s

Files changed:
%s

Actual code changes:
%s

Generate a SHORT pull request title. Follow these strict rules:
- Maximum 40 characters (STRICT LIMIT)
- Start with action verb (Add, Fix, Update, Improve, etc.)
- Be concise and specific
- No unnecessary words or phrases
- Don't mention "Claude", "agent", or implementation details
- Base the title on the actual changes shown above

Examples:
- "Fix error handling in message processor"
- "Add user authentication middleware"
- "Update API response format"

CRITICAL: Your response must contain ONLY the PR title text. Do not include:
- Any explanations or reasoning
- Quotes around the title
- "Here is the title:" or similar phrases
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the short title text, nothing else.`, branchName, commitInfo, diffSummary, diffContent)

	result, err := g.claudeService.StartNewConversation(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR title: %w", err)
	}

	return strings.TrimSpace(result.Output), nil
}

func (g *GitUseCase) generatePRBodyWithClaude(branchName, slackThreadLink string) (string, error) {
	log.Info("🤖 Asking Claude to generate PR body")

	// Get default branch to compare against
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get commit messages specific to this branch
	commitMessages, err := g.gitClient.GetBranchCommitMessages(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit messages: %w", err)
	}

	// Get diff summary
	diffSummary, err := g.gitClient.GetBranchDiffSummary(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff summary: %w", err)
	}

	// Get actual diff content for better context
	diffContent, err := g.gitClient.GetBranchDiffContent(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff content: %w", err)
	}

	// Build commit info for context
	commitInfo := "No commits found"
	if len(commitMessages) > 0 {
		commitInfo = strings.Join(commitMessages, "\n- ")
		commitInfo = "- " + commitInfo // Add bullet to first item
	}

	prompt := fmt.Sprintf(`I'm creating a pull request for Git branch: "%s"

Here are the commits made on this branch (not including main branch commits):
%s

Files changed:
%s

Actual code changes:
%s

Generate a concise pull request description with:
- ## Summary: High-level overview of what changed (2-3 bullet points max)
- ## Why: Brief explanation of the motivation/reasoning behind the change

Keep it professional but brief. Focus on WHAT changed at a high level and WHY the change was necessary, not detailed implementation specifics.

Use proper markdown formatting.

IMPORTANT: 
- Do NOT include any "Generated with Claude Control" or similar footer text. I will add that separately.
- Keep the summary concise - avoid listing every single file or detailed code changes
- Focus on the business/functional purpose of the changes
- Do NOT include any introductory text like "Here is your description"

CRITICAL: Your response must contain ONLY the PR description in markdown format. Do not include:
- Any explanations or reasoning about your response
- "Here is the description:" or similar phrases
- Any text before or after the description
- Any commentary about the changes
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the PR description in markdown format, nothing else.`, branchName, commitInfo, diffSummary, diffContent)

	result, err := g.claudeService.StartNewConversation(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR body: %w", err)
	}

	// Append footer with Slack thread link
	cleanBody := strings.TrimSpace(result.Output)
	finalBody := cleanBody + fmt.Sprintf(
		"\n\n---\nGenerated by [Claude Control](https://claudecontrol.com) from this [slack thread](%s)",
		slackThreadLink,
	)

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

	// Check if the expected footer pattern is present (using regex to handle different permalinks)
	footerPattern := `---\s*\n.*Generated by \[Claude Control\]\(https://claudecontrol\.com\) from this \[slack thread\]\([^)]+\)`

	matched, err := regexp.MatchString(footerPattern, currentDescription)
	if err != nil {
		log.Error("❌ Failed to match footer pattern: %v", err)
		return fmt.Errorf("failed to match footer pattern: %w", err)
	}

	if matched {
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
		if strings.Contains(trimmedLine, "Generated by Claude Control") ||
			strings.Contains(trimmedLine, "Generated by Claude Code") {
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
				if strings.Contains(nextLine, "Generated by Claude") {
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
	expectedFooter := fmt.Sprintf(
		"Generated by [Claude Control](https://claudecontrol.com) from this [slack thread](%s)",
		slackThreadLink,
	)
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

func (g *GitUseCase) CleanupStaleBranches() error {
	log.Info("📋 Starting to cleanup stale ccagent branches")

	// Get all local branches
	localBranches, err := g.gitClient.GetLocalBranches()
	if err != nil {
		log.Error("❌ Failed to get local branches: %v", err)
		return fmt.Errorf("failed to get local branches: %w", err)
	}

	// Get current branch to avoid deleting it
	currentBranch, err := g.gitClient.GetCurrentBranch()
	if err != nil {
		log.Error("❌ Failed to get current branch: %v", err)
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get default branch to avoid deleting it
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch: %v", err)
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get all tracked job branches from app state
	trackedJobs := g.appState.GetAllJobs()
	trackedBranches := make(map[string]bool)
	for _, jobData := range trackedJobs {
		if jobData.BranchName != "" {
			trackedBranches[jobData.BranchName] = true
		}
	}

	// Filter branches for cleanup
	var branchesToDelete []string
	protectedBranches := []string{"main", "master", currentBranch, defaultBranch}

	for _, branch := range localBranches {
		// Only process ccagent/ branches
		if !strings.HasPrefix(branch, "ccagent/") {
			continue
		}

		// Skip protected branches
		isProtected := false
		for _, protected := range protectedBranches {
			if branch == protected {
				isProtected = true
				break
			}
		}
		if isProtected {
			log.Info("⚠️ Skipping protected branch: %s", branch)
			continue
		}

		// Skip tracked branches
		if trackedBranches[branch] {
			log.Info("⚠️ Skipping tracked branch: %s", branch)
			continue
		}

		// This branch is stale - mark for deletion
		branchesToDelete = append(branchesToDelete, branch)
	}

	if len(branchesToDelete) == 0 {
		log.Info("✅ No stale ccagent branches found")
		log.Info("📋 Completed successfully - no stale branches to cleanup")
		return nil
	}

	log.Info("🧹 Found %d stale ccagent branches to delete", len(branchesToDelete))

	// Delete each stale branch
	deletedCount := 0
	for _, branch := range branchesToDelete {
		if err := g.gitClient.DeleteLocalBranch(branch); err != nil {
			log.Error("❌ Failed to delete stale branch %s: %v", branch, err)
			// Continue with other branches even if one fails
			continue
		}
		deletedCount++
		log.Info("🗑️ Deleted stale branch: %s", branch)
	}

	log.Info("✅ Successfully deleted %d out of %d stale ccagent branches", deletedCount, len(branchesToDelete))
	log.Info("📋 Completed successfully - cleaned up stale branches")
	return nil
}

func (g *GitUseCase) updatePRTitleAndDescriptionIfNeeded(branchName, slackThreadLink string) error {
	log.Info("📋 Starting to update PR title and description if needed for branch: %s", branchName)

	// Get current PR title and description
	currentTitle, err := g.gitClient.GetPRTitle(branchName)
	if err != nil {
		log.Error("❌ Failed to get current PR title: %v", err)
		return fmt.Errorf("failed to get current PR title: %w", err)
	}

	currentDescription, err := g.gitClient.GetPRDescription(branchName)
	if err != nil {
		log.Error("❌ Failed to get current PR description: %v", err)
		return fmt.Errorf("failed to get current PR description: %w", err)
	}

	// Generate updated PR title using Claude
	updatedTitle, err := g.generateUpdatedPRTitleWithClaude(branchName, currentTitle)
	if err != nil {
		log.Error("❌ Failed to generate updated PR title with Claude: %v", err)
		return fmt.Errorf("failed to generate updated PR title with Claude: %w", err)
	}

	// Generate updated PR description using Claude
	updatedDescription, err := g.generateUpdatedPRDescriptionWithClaude(branchName, currentDescription, slackThreadLink)
	if err != nil {
		log.Error("❌ Failed to generate updated PR description with Claude: %v", err)
		return fmt.Errorf("failed to generate updated PR description with Claude: %w", err)
	}

	// Update title if it has changed
	if strings.TrimSpace(updatedTitle) != strings.TrimSpace(currentTitle) {
		log.Info("🔄 PR title has changed, updating...")
		if err := g.gitClient.UpdatePRTitle(branchName, updatedTitle); err != nil {
			log.Error("❌ Failed to update PR title: %v", err)
			return fmt.Errorf("failed to update PR title: %w", err)
		}
		log.Info("✅ Successfully updated PR title")
	} else {
		log.Info("ℹ️ PR title remains the same - no update needed")
	}

	// Update description if it has changed
	if strings.TrimSpace(updatedDescription) != strings.TrimSpace(currentDescription) {
		log.Info("🔄 PR description has changed, updating...")
		if err := g.gitClient.UpdatePRDescription(branchName, updatedDescription); err != nil {
			log.Error("❌ Failed to update PR description: %v", err)
			return fmt.Errorf("failed to update PR description: %w", err)
		}
		log.Info("✅ Successfully updated PR description")
	} else {
		log.Info("ℹ️ PR description remains the same - no update needed")
	}

	log.Info("📋 Completed successfully - updated PR title and description if needed")
	return nil
}

func (g *GitUseCase) generateUpdatedPRTitleWithClaude(branchName, currentTitle string) (string, error) {
	log.Info("🤖 Asking Claude to generate updated PR title")

	// Get default branch to compare against
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get recent commits since the branch was created
	commitMessages, err := g.gitClient.GetBranchCommitMessages(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit messages: %w", err)
	}

	// Get diff summary
	diffSummary, err := g.gitClient.GetBranchDiffSummary(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff summary: %w", err)
	}

	// Build commit info for context
	commitInfo := "No commits found"
	if len(commitMessages) > 0 {
		commitInfo = fmt.Sprintf("All commits on this branch:\n%s", strings.Join(commitMessages, "\n"))
	}

	prompt := fmt.Sprintf(`I have an existing pull request with this title:
CURRENT TITLE: "%s"

The branch "%s" now has these commits and changes:

%s

Files changed:
%s

INSTRUCTIONS:
- Review the current title and the latest changes made to this branch
- ONLY update the title if the current title has become obsolete or doesn't accurately reflect the work
- If the current title still accurately captures the main purpose, return it unchanged
- If updating, make it additive - build upon the existing title rather than replacing it entirely
- Maximum 40 characters (STRICT LIMIT)
- Start with action verb (Add, Fix, Update, Improve, etc.)
- Be concise and specific
- Don't mention "Claude", "agent", or implementation details

Examples of when to update:
- Current: "Fix error handling" → New commits add user auth → Updated: "Fix error handling and add user auth"
- Current: "Add basic feature" → New commits improve performance → Updated: "Add feature with performance improvements"

Examples of when NOT to update:
- Current: "Fix authentication issues" → New commits fix more auth bugs → Keep: "Fix authentication issues"
- Current: "Add user dashboard" → New commits fix small UI bugs → Keep: "Add user dashboard"

CRITICAL: Your response must contain ONLY the PR title text. Do not include:
- Any explanations or reasoning about your decision
- Quotes around the title
- "The title should be:" or similar phrases
- Commentary about whether you updated it or not
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the title text (updated or unchanged), nothing else.`, currentTitle, branchName, commitInfo, diffSummary)

	result, err := g.claudeService.StartNewConversation(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate updated PR title: %w", err)
	}

	return strings.TrimSpace(result.Output), nil
}

func (g *GitUseCase) generateUpdatedPRDescriptionWithClaude(
	branchName, currentDescription, slackThreadLink string,
) (string, error) {
	log.Info("🤖 Asking Claude to generate updated PR description")

	// Get default branch to compare against
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get recent commits since the branch was created
	commitMessages, err := g.gitClient.GetBranchCommitMessages(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit messages: %w", err)
	}

	// Get diff summary
	diffSummary, err := g.gitClient.GetBranchDiffSummary(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff summary: %w", err)
	}

	// Build commit info for context
	commitInfo := "No commits found"
	if len(commitMessages) > 0 {
		commitInfo = strings.Join(commitMessages, "\n- ")
		commitInfo = "- " + commitInfo // Add bullet to first item
	}

	// Remove existing footer from current description for analysis
	currentDescriptionClean := g.removeFooterFromDescription(currentDescription)

	prompt := fmt.Sprintf(`I have an existing pull request with this description:

CURRENT DESCRIPTION:
%s

The branch "%s" now has these commits and changes:

All commits on this branch:
%s

Files changed:
%s

INSTRUCTIONS:
- Review the current description and the latest changes made to this branch
- ONLY update the description if significant new functionality has been added that warrants description updates
- If the current description still accurately captures the work, return it unchanged (without footer)
- If updating, make it additive - enhance the existing description rather than replacing it
- Keep the same structure: ## Summary and ## Why sections
- Focus on WHAT changed at a high level and WHY the change was necessary
- Use proper markdown formatting
- Keep it professional but brief
- Do NOT mention implementation details

Examples of when to update:
- Current description only mentions "Fix auth bug" → New commits add complete user management → Update to include both
- Current description is "Add dashboard" → New commits add charts and filters → Update to "Add dashboard with charts and filtering"

Examples of when NOT to update:
- Current description covers "User authentication system" → New commits just fix small auth bugs → Keep current
- Current description mentions "Performance improvements" → New commits make minor tweaks → Keep current

IMPORTANT: 
- Do NOT include any "Generated with Claude Control" or similar footer text. I will add that separately.
- Return only the description content in markdown format, nothing else.
- If no update is needed, return the current description exactly as provided (minus any footer).

CRITICAL: Your response must contain ONLY the PR description in markdown format. Do not include:
- Any explanations or reasoning about your decision
- "Here is the updated description:" or similar phrases
- Commentary about whether you updated it or not
- Any text before or after the description
- Any analysis of the changes
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the PR description in markdown format, nothing else.`, currentDescriptionClean, branchName, commitInfo, diffSummary)

	result, err := g.claudeService.StartNewConversation(prompt)
	if err != nil {
		return "", fmt.Errorf("claude failed to generate updated PR description: %w", err)
	}

	// Append footer with Slack thread link
	cleanBody := strings.TrimSpace(result.Output)
	finalBody := cleanBody + fmt.Sprintf(
		"\n\n---\nGenerated by [Claude Control](https://claudecontrol.com) from this [slack thread](%s)",
		slackThreadLink,
	)

	return finalBody, nil
}

func (g *GitUseCase) removeFooterFromDescription(description string) string {
	// Remove the Claude Control footer to get clean description for analysis
	footerPattern := `---\s*\n.*Generated by \[Claude Control\]\(https://claudecontrol\.com\) from this \[slack thread\]\([^)]+\)`

	// Use regex to remove the footer section
	re := regexp.MustCompile(footerPattern)
	cleanDescription := re.ReplaceAllString(description, "")

	// Clean up any trailing whitespace
	return strings.TrimSpace(cleanDescription)
}
