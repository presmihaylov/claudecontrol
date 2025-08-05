package usecases

import (
	"ccagent/models"
	"ccagent/services"
)

// MockGitClient implements GitClientInterface for testing
type MockGitClient struct {
	// Git repository validation
	IsGitRepositoryFunc       func() error
	IsGitRepositoryRootFunc   func() error
	HasRemoteRepositoryFunc   func() error
	IsGitHubCLIAvailableFunc  func() error
	ValidateRemoteAccessFunc  func() error

	// Branch operations
	ResetHardFunc               func() error
	CleanUntrackedFunc          func() error
	GetDefaultBranchFunc        func() (string, error)
	CheckoutBranchFunc          func(branchName string) error
	PullLatestFunc              func() error
	CreateAndCheckoutBranchFunc func(branchName string) error
	GetCurrentBranchFunc        func() (string, error)
	GetLocalBranchesFunc        func() ([]string, error)
	DeleteLocalBranchFunc       func(branchName string) error

	// Commit operations
	HasUncommittedChangesFunc func() (bool, error)
	AddAllFunc                func() error
	CommitFunc                func(message string) error
	GetLatestCommitHashFunc   func() (string, error)
	GetRemoteURLFunc          func() (string, error)
	PushBranchFunc            func(branchName string) error

	// Pull request operations
	ExtractPRIDFromURLFunc       func(prURL string) string
	HasExistingPRFunc            func(branchName string) (bool, error)
	GetPRURLFunc                 func(branchName string) (string, error)
	CreatePullRequestFunc        func(title, body, baseBranch string) (string, error)
	GetBranchCommitMessagesFunc  func(branchName, baseBranch string) ([]string, error)
	GetBranchDiffSummaryFunc     func(branchName, baseBranch string) (string, error)
	GetBranchDiffContentFunc     func(branchName, baseBranch string) (string, error)
	GetPRDescriptionFunc         func(branchName string) (string, error)
	UpdatePRDescriptionFunc      func(branchName, newDescription string) error
	GetPRStateFunc               func(branchName string) (string, error)
	GetPRStateByIDFunc           func(prID string) (string, error)
}

// Git repository validation methods
func (m *MockGitClient) IsGitRepository() error {
	if m.IsGitRepositoryFunc != nil {
		return m.IsGitRepositoryFunc()
	}
	return nil
}

func (m *MockGitClient) IsGitRepositoryRoot() error {
	if m.IsGitRepositoryRootFunc != nil {
		return m.IsGitRepositoryRootFunc()
	}
	return nil
}

func (m *MockGitClient) HasRemoteRepository() error {
	if m.HasRemoteRepositoryFunc != nil {
		return m.HasRemoteRepositoryFunc()
	}
	return nil
}

func (m *MockGitClient) IsGitHubCLIAvailable() error {
	if m.IsGitHubCLIAvailableFunc != nil {
		return m.IsGitHubCLIAvailableFunc()
	}
	return nil
}

func (m *MockGitClient) ValidateRemoteAccess() error {
	if m.ValidateRemoteAccessFunc != nil {
		return m.ValidateRemoteAccessFunc()
	}
	return nil
}

// Branch operations
func (m *MockGitClient) ResetHard() error {
	if m.ResetHardFunc != nil {
		return m.ResetHardFunc()
	}
	return nil
}

func (m *MockGitClient) CleanUntracked() error {
	if m.CleanUntrackedFunc != nil {
		return m.CleanUntrackedFunc()
	}
	return nil
}

func (m *MockGitClient) GetDefaultBranch() (string, error) {
	if m.GetDefaultBranchFunc != nil {
		return m.GetDefaultBranchFunc()
	}
	return "main", nil
}

func (m *MockGitClient) CheckoutBranch(branchName string) error {
	if m.CheckoutBranchFunc != nil {
		return m.CheckoutBranchFunc(branchName)
	}
	return nil
}

func (m *MockGitClient) PullLatest() error {
	if m.PullLatestFunc != nil {
		return m.PullLatestFunc()
	}
	return nil
}

func (m *MockGitClient) CreateAndCheckoutBranch(branchName string) error {
	if m.CreateAndCheckoutBranchFunc != nil {
		return m.CreateAndCheckoutBranchFunc(branchName)
	}
	return nil
}

func (m *MockGitClient) GetCurrentBranch() (string, error) {
	if m.GetCurrentBranchFunc != nil {
		return m.GetCurrentBranchFunc()
	}
	return "main", nil
}

func (m *MockGitClient) GetLocalBranches() ([]string, error) {
	if m.GetLocalBranchesFunc != nil {
		return m.GetLocalBranchesFunc()
	}
	return []string{"main"}, nil
}

func (m *MockGitClient) DeleteLocalBranch(branchName string) error {
	if m.DeleteLocalBranchFunc != nil {
		return m.DeleteLocalBranchFunc(branchName)
	}
	return nil
}

// Commit operations
func (m *MockGitClient) HasUncommittedChanges() (bool, error) {
	if m.HasUncommittedChangesFunc != nil {
		return m.HasUncommittedChangesFunc()
	}
	return false, nil
}

func (m *MockGitClient) AddAll() error {
	if m.AddAllFunc != nil {
		return m.AddAllFunc()
	}
	return nil
}

func (m *MockGitClient) Commit(message string) error {
	if m.CommitFunc != nil {
		return m.CommitFunc(message)
	}
	return nil
}

func (m *MockGitClient) GetLatestCommitHash() (string, error) {
	if m.GetLatestCommitHashFunc != nil {
		return m.GetLatestCommitHashFunc()
	}
	return "abc123", nil
}

func (m *MockGitClient) GetRemoteURL() (string, error) {
	if m.GetRemoteURLFunc != nil {
		return m.GetRemoteURLFunc()
	}
	return "https://github.com/user/repo", nil
}

func (m *MockGitClient) PushBranch(branchName string) error {
	if m.PushBranchFunc != nil {
		return m.PushBranchFunc(branchName)
	}
	return nil
}

// Pull request operations
func (m *MockGitClient) ExtractPRIDFromURL(prURL string) string {
	if m.ExtractPRIDFromURLFunc != nil {
		return m.ExtractPRIDFromURLFunc(prURL)
	}
	return "123"
}

func (m *MockGitClient) HasExistingPR(branchName string) (bool, error) {
	if m.HasExistingPRFunc != nil {
		return m.HasExistingPRFunc(branchName)
	}
	return false, nil
}

func (m *MockGitClient) GetPRURL(branchName string) (string, error) {
	if m.GetPRURLFunc != nil {
		return m.GetPRURLFunc(branchName)
	}
	return "https://github.com/user/repo/pull/123", nil
}

func (m *MockGitClient) CreatePullRequest(title, body, baseBranch string) (string, error) {
	if m.CreatePullRequestFunc != nil {
		return m.CreatePullRequestFunc(title, body, baseBranch)
	}
	return "https://github.com/user/repo/pull/123", nil
}

func (m *MockGitClient) GetBranchCommitMessages(branchName, baseBranch string) ([]string, error) {
	if m.GetBranchCommitMessagesFunc != nil {
		return m.GetBranchCommitMessagesFunc(branchName, baseBranch)
	}
	return []string{"Test commit"}, nil
}

func (m *MockGitClient) GetBranchDiffSummary(branchName, baseBranch string) (string, error) {
	if m.GetBranchDiffSummaryFunc != nil {
		return m.GetBranchDiffSummaryFunc(branchName, baseBranch)
	}
	return "M\tfile.go", nil
}

func (m *MockGitClient) GetBranchDiffContent(branchName, baseBranch string) (string, error) {
	if m.GetBranchDiffContentFunc != nil {
		return m.GetBranchDiffContentFunc(branchName, baseBranch)
	}
	return "diff --git a/file.go b/file.go\n+added line", nil
}

func (m *MockGitClient) GetPRDescription(branchName string) (string, error) {
	if m.GetPRDescriptionFunc != nil {
		return m.GetPRDescriptionFunc(branchName)
	}
	return "Test PR description", nil
}

func (m *MockGitClient) UpdatePRDescription(branchName, newDescription string) error {
	if m.UpdatePRDescriptionFunc != nil {
		return m.UpdatePRDescriptionFunc(branchName, newDescription)
	}
	return nil
}

func (m *MockGitClient) GetPRState(branchName string) (string, error) {
	if m.GetPRStateFunc != nil {
		return m.GetPRStateFunc(branchName)
	}
	return "open", nil
}

func (m *MockGitClient) GetPRStateByID(prID string) (string, error) {
	if m.GetPRStateByIDFunc != nil {
		return m.GetPRStateByIDFunc(prID)
	}
	return "open", nil
}

// MockClaudeService implements ClaudeServiceInterface for testing
type MockClaudeService struct {
	StartNewConversationFunc func(prompt string) (*services.ClaudeResult, error)
}

func (m *MockClaudeService) StartNewConversation(prompt string) (*services.ClaudeResult, error) {
	if m.StartNewConversationFunc != nil {
		return m.StartNewConversationFunc(prompt)
	}
	return &services.ClaudeResult{
		Output:    "Generated response",
		SessionID: "test-session-123",
	}, nil
}

// MockAppState implements AppStateInterface for testing
type MockAppState struct {
	GetAllJobsFunc func() map[string]*models.JobData
}

func (m *MockAppState) GetAllJobs() map[string]*models.JobData {
	if m.GetAllJobsFunc != nil {
		return m.GetAllJobsFunc()
	}
	return make(map[string]*models.JobData)
}