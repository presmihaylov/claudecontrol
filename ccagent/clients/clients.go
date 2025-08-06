package clients

// ClaudeClient defines the interface for Claude operations
type ClaudeClient interface {
	ContinueSession(sessionID, prompt string) (string, error)
	StartNewSession(prompt string) (string, error)
	StartNewSessionWithSystemPrompt(prompt, systemPrompt string) (string, error)
}

// GitClient defines the interface for Git operations
type GitClient interface {
	CheckoutBranch(branchName string) error
	PullLatest() error
	ResetHard() error
	CleanUntracked() error
	AddAll() error
	Commit(message string) error
	PushBranch(branchName string) error
	CreatePullRequest(title, body, baseBranch string) (string, error)
	GetPRURL(branchName string) (string, error)
	GetCurrentBranch() (string, error)
	GetDefaultBranch() (string, error)
	CreateAndCheckoutBranch(branchName string) error
	IsGitRepository() error
	IsGitRepositoryRoot() error
	HasRemoteRepository() error
	IsGitHubCLIAvailable() error
	HasUncommittedChanges() (bool, error)
	HasExistingPR(branchName string) (bool, error)
	GetLatestCommitHash() (string, error)
	GetRemoteURL() (string, error)
	GetPRDescription(branchName string) (string, error)
	UpdatePRDescription(branchName, newDescription string) error
	GetPRState(branchName string) (string, error)
	ExtractPRIDFromURL(prURL string) string
	GetPRStateByID(prID string) (string, error)
	GetBranchCommitMessages(branchName, baseBranch string) ([]string, error)
	GetBranchDiffSummary(branchName, baseBranch string) (string, error)
	GetBranchDiffContent(branchName, baseBranch string) (string, error)
	GetLocalBranches() ([]string, error)
	DeleteLocalBranch(branchName string) error
	ValidateRemoteAccess() error
}
