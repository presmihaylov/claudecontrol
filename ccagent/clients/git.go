package clients

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ccagent/core/log"
)

type GitClient struct{}

func NewGitClient() *GitClient {
	return &GitClient{}
}

func (g *GitClient) CheckoutBranch(branchName string) error {
	log.Info("📋 Starting to checkout branch: %s", branchName)

	cmd := exec.Command("git", "checkout", branchName)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Git checkout failed for branch %s: %v\nOutput: %s", branchName, err, string(output))
		return fmt.Errorf("git checkout failed: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Successfully checked out branch: %s", branchName)
	log.Info("📋 Completed successfully - checked out branch")
	return nil
}

func (g *GitClient) PullLatest() error {
	log.Info("📋 Starting to pull latest changes")

	cmd := exec.Command("git", "pull")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Git pull failed: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("git pull failed: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Successfully pulled latest changes")
	log.Info("📋 Completed successfully - pulled latest changes")
	return nil
}

func (g *GitClient) ResetHard() error {
	log.Info("📋 Starting to reset hard to HEAD")

	cmd := exec.Command("git", "reset", "--hard", "HEAD")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Git reset hard failed: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("git reset hard failed: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Successfully reset hard to HEAD")
	log.Info("📋 Completed successfully - reset hard")
	return nil
}

func (g *GitClient) CleanUntracked() error {
	log.Info("📋 Starting to clean untracked files")

	cmd := exec.Command("git", "clean", "-fd")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Git clean failed: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("git clean failed: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Successfully cleaned untracked files")
	log.Info("📋 Completed successfully - cleaned untracked files")
	return nil
}

func (g *GitClient) AddAll() error {
	log.Info("📋 Starting to add all changes")

	cmd := exec.Command("git", "add", ".")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Git add failed: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("git add failed: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Successfully added all changes")
	log.Info("📋 Completed successfully - added all changes")
	return nil
}

func (g *GitClient) Commit(message string) error {
	log.Info("📋 Starting to commit with message: %s", message)

	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Git commit failed with message '%s': %v\nOutput: %s", message, err, string(output))
		return fmt.Errorf("git commit failed: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Successfully committed changes")
	log.Info("📋 Completed successfully - committed changes")
	return nil
}

func (g *GitClient) PushBranch(branchName string) error {
	log.Info("📋 Starting to push branch: %s", branchName)

	cmd := exec.Command("git", "push", "-u", "origin", branchName)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Git push failed for branch %s: %v\nOutput: %s", branchName, err, string(output))
		return fmt.Errorf("git push failed: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Successfully pushed branch: %s", branchName)
	log.Info("📋 Completed successfully - pushed branch")
	return nil
}

func (g *GitClient) CreatePullRequest(title, body, baseBranch string) (string, error) {
	log.Info("📋 Starting to create pull request: %s", title)

	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", body, "--base", baseBranch)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ GitHub PR creation failed for title '%s': %v\nOutput: %s", title, err, string(output))
		return "", fmt.Errorf("github pr creation failed: %w\nOutput: %s", err, string(output))
	}

	// The output contains the PR URL
	prURL := strings.TrimSpace(string(output))

	log.Info("✅ Successfully created pull request: %s", title)
	log.Info("📋 Completed successfully - created pull request")
	return prURL, nil
}

func (g *GitClient) GetPRURL(branchName string) (string, error) {
	log.Info("📋 Starting to get PR URL for branch: %s", branchName)

	cmd := exec.Command("gh", "pr", "view", branchName, "--json", "url", "--jq", ".url")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get PR URL for branch %s: %v\nOutput: %s", branchName, err, string(output))
		return "", fmt.Errorf("failed to get PR URL: %w\nOutput: %s", err, string(output))
	}

	prURL := strings.TrimSpace(string(output))

	log.Info("✅ Successfully got PR URL: %s", prURL)
	log.Info("📋 Completed successfully - got PR URL")
	return prURL, nil
}

func (g *GitClient) GetCurrentBranch() (string, error) {
	log.Info("📋 Starting to get current branch")

	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get current branch: %v\nOutput: %s", err, string(output))
		return "", fmt.Errorf("failed to get current branch: %w\nOutput: %s", err, string(output))
	}

	branch := strings.TrimSpace(string(output))
	log.Info("✅ Current branch: %s", branch)
	log.Info("📋 Completed successfully - got current branch")
	return branch, nil
}

func (g *GitClient) GetDefaultBranch() (string, error) {
	log.Info("📋 Starting to determine default branch")

	// Run git remote show origin to get HEAD branch information
	cmd := exec.Command("git", "remote", "show", "origin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("❌ Failed to run git remote show origin: %v\nOutput: %s", err, string(output))
		return "", fmt.Errorf("failed to get remote information: %w\nOutput: %s", err, string(output))
	}

	// Parse the output to find the HEAD branch line
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "HEAD branch:") {
			// Extract branch name after "HEAD branch: "
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) != 2 {
				log.Error("❌ Unexpected format in remote show output: %s", trimmedLine)
				return "", fmt.Errorf("unexpected format in remote show output: %s", trimmedLine)
			}

			branchName := strings.TrimSpace(parts[1])
			log.Info("✅ Default branch from remote: %s", branchName)
			log.Info("📋 Completed successfully - got default branch from remote")
			return branchName, nil
		}
	}

	log.Error("❌ Could not find HEAD branch in remote show output")
	return "", fmt.Errorf("could not determine default branch from remote show output")
}

func (g *GitClient) CreateAndCheckoutBranch(branchName string) error {
	log.Info("📋 Starting to create and checkout branch: %s", branchName)

	cmd := exec.Command("git", "checkout", "-b", branchName)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Git checkout -b failed for branch %s: %v\nOutput: %s", branchName, err, string(output))
		return fmt.Errorf("git checkout -b failed: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Successfully created and checked out branch: %s", branchName)
	log.Info("📋 Completed successfully - created and checked out branch")
	return nil
}

func (g *GitClient) IsGitRepository() error {
	log.Info("📋 Starting to check if current directory is a Git repository")

	cmd := exec.Command("git", "rev-parse", "--git-dir")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Not a Git repository: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("not a git repository: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Current directory is a Git repository")
	log.Info("📋 Completed successfully - validated Git repository")
	return nil
}

func (g *GitClient) IsGitRepositoryRoot() error {
	log.Info("📋 Starting to check if current directory is the Git repository root")

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get Git repository root: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("failed to get git repository root: %w\nOutput: %s", err, string(output))
	}

	gitRoot := strings.TrimSpace(string(output))

	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error("❌ Failed to get current working directory: %v", err)
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	if gitRoot != currentDir {
		log.Error("❌ Not at Git repository root. Current: %s, Git root: %s", currentDir, gitRoot)
		return fmt.Errorf("ccagent must be run from the Git repository root directory. Current: %s, Git root: %s", currentDir, gitRoot)
	}

	log.Info("✅ Current directory is the Git repository root")
	log.Info("📋 Completed successfully - validated Git repository root")
	return nil
}

func (g *GitClient) HasRemoteRepository() error {
	log.Info("📋 Starting to check for remote repository")

	cmd := exec.Command("git", "remote", "-v")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to check remotes: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("failed to check git remotes: %w\nOutput: %s", err, string(output))
	}

	remotes := strings.TrimSpace(string(output))
	if remotes == "" {
		log.Error("❌ No remote repositories configured")
		return fmt.Errorf("no remote repositories configured")
	}

	log.Info("✅ Remote repository found")
	log.Info("📋 Completed successfully - validated remote repository")
	return nil
}

func (g *GitClient) IsGitHubCLIAvailable() error {
	log.Info("📋 Starting to check GitHub CLI availability")

	// Check if gh command exists
	cmd := exec.Command("gh", "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ GitHub CLI not found: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("github cli (gh) not found: %w\nOutput: %s", err, string(output))
	}

	// Check if gh is authenticated
	cmd = exec.Command("gh", "auth", "status")
	output, err = cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ GitHub CLI not authenticated: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("github cli not authenticated (run 'gh auth login'): %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ GitHub CLI is available and authenticated")
	log.Info("📋 Completed successfully - validated GitHub CLI")
	return nil
}

func (g *GitClient) HasUncommittedChanges() (bool, error) {
	log.Info("📋 Starting to check for uncommitted changes")

	// Check for staged and unstaged changes
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to check git status: %v\nOutput: %s", err, string(output))
		return false, fmt.Errorf("failed to check git status: %w\nOutput: %s", err, string(output))
	}

	statusOutput := strings.TrimSpace(string(output))
	hasChanges := statusOutput != ""

	if hasChanges {
		log.Info("✅ Found uncommitted changes")
		log.Info("📄 Git status output: %s", statusOutput)
	} else {
		log.Info("✅ No uncommitted changes found")
	}

	log.Info("📋 Completed successfully - checked for uncommitted changes")
	return hasChanges, nil
}



func (g *GitClient) HasExistingPR(branchName string) (bool, error) {
	log.Info("📋 Starting to check for existing PR for branch: %s", branchName)

	// Use GitHub CLI to list PRs for the current branch
	cmd := exec.Command("gh", "pr", "list", "--head", branchName, "--json", "number")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to check for existing PR for branch %s: %v\nOutput: %s", branchName, err, string(output))
		return false, fmt.Errorf("failed to check for existing PR: %w\nOutput: %s", err, string(output))
	}

	// If output is "[]" (empty JSON array), no PRs exist for this branch
	outputStr := strings.TrimSpace(string(output))
	hasPR := outputStr != "[]" && outputStr != ""

	if hasPR {
		log.Info("✅ Found existing PR for branch: %s", branchName)
	} else {
		log.Info("✅ No existing PR found for branch: %s", branchName)
	}

	log.Info("📋 Completed successfully - checked for existing PR")
	return hasPR, nil
}

func (g *GitClient) GetLatestCommitHash() (string, error) {
	log.Info("📋 Starting to get latest commit hash")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get latest commit hash: %v\nOutput: %s", err, string(output))
		return "", fmt.Errorf("failed to get latest commit hash: %w\nOutput: %s", err, string(output))
	}

	commitHash := strings.TrimSpace(string(output))
	log.Info("✅ Latest commit hash: %s", commitHash)
	log.Info("📋 Completed successfully - got latest commit hash")
	return commitHash, nil
}

func (g *GitClient) GetRemoteURL() (string, error) {
	log.Info("📋 Starting to get remote URL")

	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get remote URL: %v\nOutput: %s", err, string(output))
		return "", fmt.Errorf("failed to get remote URL: %w\nOutput: %s", err, string(output))
	}

	remoteURL := strings.TrimSpace(string(output))

	// Convert SSH URL to HTTPS if needed for GitHub links
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		// Convert git@github.com:owner/repo.git to https://github.com/owner/repo
		remoteURL = strings.Replace(remoteURL, "git@github.com:", "https://github.com/", 1)
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
	} else if strings.HasSuffix(remoteURL, ".git") {
		// Remove .git suffix from HTTPS URLs
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
	}

	log.Info("✅ Remote URL: %s", remoteURL)
	log.Info("📋 Completed successfully - got remote URL")
	return remoteURL, nil
}

func (g *GitClient) GetPRDescription(branchName string) (string, error) {
	log.Info("📋 Starting to get PR description for branch: %s", branchName)

	cmd := exec.Command("gh", "pr", "view", branchName, "--json", "body", "--jq", ".body")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get PR description for branch %s: %v\nOutput: %s", branchName, err, string(output))
		return "", fmt.Errorf("failed to get PR description: %w\nOutput: %s", err, string(output))
	}

	description := strings.TrimSpace(string(output))
	log.Info("✅ Successfully got PR description")
	log.Info("📋 Completed successfully - got PR description")
	return description, nil
}

func (g *GitClient) UpdatePRDescription(branchName, newDescription string) error {
	log.Info("📋 Starting to update PR description for branch: %s", branchName)

	cmd := exec.Command("gh", "pr", "edit", branchName, "--body", newDescription)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to update PR description for branch %s: %v\nOutput: %s", branchName, err, string(output))
		return fmt.Errorf("failed to update PR description: %w\nOutput: %s", err, string(output))
	}

	log.Info("✅ Successfully updated PR description")
	log.Info("📋 Completed successfully - updated PR description")
	return nil
}

func (g *GitClient) GetPRState(branchName string) (string, error) {
	log.Info("📋 Starting to get PR state for branch: %s", branchName)

	cmd := exec.Command("gh", "pr", "view", branchName, "--json", "state", "--jq", ".state")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get PR state for branch %s: %v\nOutput: %s", branchName, err, string(output))
		return "", fmt.Errorf("failed to get PR state: %w\nOutput: %s", err, string(output))
	}

	state := strings.TrimSpace(string(output))
	log.Info("✅ Retrieved PR state: %s", state)
	log.Info("📋 Completed successfully - got PR state")
	return strings.ToLower(state), nil
}

func (g *GitClient) ExtractPRIDFromURL(prURL string) string {
	if prURL == "" {
		return ""
	}

	// Extract PR number from URL like https://github.com/user/repo/pull/1234
	parts := strings.Split(prURL, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}

	return ""
}

func (g *GitClient) GetPRStateByID(prID string) (string, error) {
	log.Info("📋 Starting to get PR state by ID: %s", prID)

	cmd := exec.Command("gh", "pr", "view", prID, "--json", "state", "--jq", ".state")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get PR state for PR ID %s: %v\nOutput: %s", prID, err, string(output))
		return "", fmt.Errorf("failed to get PR state by ID: %w\nOutput: %s", err, string(output))
	}

	state := strings.TrimSpace(string(output))
	log.Info("✅ Retrieved PR state by ID: %s", state)
	log.Info("📋 Completed successfully - got PR state by ID")
	return strings.ToLower(state), nil
}

func (g *GitClient) GetBranchCommitMessages(branchName, baseBranch string) ([]string, error) {
	log.Info("📋 Starting to get commit messages for branch %s vs base %s", branchName, baseBranch)

	// Get commits that are in branchName but not in baseBranch
	cmd := exec.Command("git", "log", "--pretty=format:%s", fmt.Sprintf("%s..%s", baseBranch, branchName))
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get branch commit messages: %v\nOutput: %s", err, string(output))
		return nil, fmt.Errorf("failed to get branch commit messages: %w\nOutput: %s", err, string(output))
	}

	commitMessages := []string{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			commitMessages = append(commitMessages, line)
		}
	}

	log.Info("✅ Found %d commit messages for branch", len(commitMessages))
	log.Info("📋 Completed successfully - got branch commit messages")
	return commitMessages, nil
}

func (g *GitClient) GetBranchDiffSummary(branchName, baseBranch string) (string, error) {
	log.Info("📋 Starting to get diff summary for branch %s vs base %s", branchName, baseBranch)

	// Get a concise diff summary showing files changed
	cmd := exec.Command("git", "diff", "--name-status", fmt.Sprintf("%s..%s", baseBranch, branchName))
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("❌ Failed to get branch diff summary: %v\nOutput: %s", err, string(output))
		return "", fmt.Errorf("failed to get branch diff summary: %w\nOutput: %s", err, string(output))
	}

	diffSummary := strings.TrimSpace(string(output))
	log.Info("✅ Got diff summary with %d lines", len(strings.Split(diffSummary, "\n")))
	log.Info("📋 Completed successfully - got branch diff summary")
	return diffSummary, nil
}
