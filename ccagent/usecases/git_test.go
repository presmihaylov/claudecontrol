package usecases

import (
	"errors"
	"testing"

	"ccagent/models"
	"ccagent/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitUseCase_ValidateGitEnvironment(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockGitClient)
		expectedError  string
		shouldSucceed  bool
	}{
		{
			name: "Success - all validations pass",
			setupMock: func(mock *MockGitClient) {
				mock.IsGitRepositoryFunc = func() error { return nil }
				mock.IsGitRepositoryRootFunc = func() error { return nil }
				mock.HasRemoteRepositoryFunc = func() error { return nil }
				mock.IsGitHubCLIAvailableFunc = func() error { return nil }
				mock.ValidateRemoteAccessFunc = func() error { return nil }
			},
			shouldSucceed: true,
		},
		{
			name: "Failure - not a git repository",
			setupMock: func(mock *MockGitClient) {
				mock.IsGitRepositoryFunc = func() error { return errors.New("not a git repository") }
			},
			expectedError: "ccagent must be run from within a Git repository",
		},
		{
			name: "Failure - not at repository root",
			setupMock: func(mock *MockGitClient) {
				mock.IsGitRepositoryFunc = func() error { return nil }
				mock.IsGitRepositoryRootFunc = func() error { return errors.New("not at root") }
			},
			expectedError: "ccagent must be run from the Git repository root",
		},
		{
			name: "Failure - no remote repository",
			setupMock: func(mock *MockGitClient) {
				mock.IsGitRepositoryFunc = func() error { return nil }
				mock.IsGitRepositoryRootFunc = func() error { return nil }
				mock.HasRemoteRepositoryFunc = func() error { return errors.New("no remote") }
			},
			expectedError: "git repository must have a remote configured",
		},
		{
			name: "Failure - GitHub CLI not available",
			setupMock: func(mock *MockGitClient) {
				mock.IsGitRepositoryFunc = func() error { return nil }
				mock.IsGitRepositoryRootFunc = func() error { return nil }
				mock.HasRemoteRepositoryFunc = func() error { return nil }
				mock.IsGitHubCLIAvailableFunc = func() error { return errors.New("gh not found") }
			},
			expectedError: "GitHub CLI (gh) must be installed and configured",
		},
		{
			name: "Failure - remote access validation fails",
			setupMock: func(mock *MockGitClient) {
				mock.IsGitRepositoryFunc = func() error { return nil }
				mock.IsGitRepositoryRootFunc = func() error { return nil }
				mock.HasRemoteRepositoryFunc = func() error { return nil }
				mock.IsGitHubCLIAvailableFunc = func() error { return nil }
				mock.ValidateRemoteAccessFunc = func() error { return errors.New("access denied") }
			},
			expectedError: "remote repository access validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitClient{}
			mockClaude := &MockClaudeService{}
			mockAppState := &MockAppState{}

			tt.setupMock(mockGit)

			useCase := NewGitUseCase(mockGit, mockClaude, mockAppState)
			err := useCase.ValidateGitEnvironment()

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestGitUseCase_SwitchToJobBranch(t *testing.T) {
	tests := []struct {
		name          string
		branchName    string
		setupMock     func(*MockGitClient)
		expectedError string
		shouldSucceed bool
	}{
		{
			name:       "Success - switch to existing branch",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return nil }
				mock.CleanUntrackedFunc = func() error { return nil }
				mock.GetDefaultBranchFunc = func() (string, error) { return "main", nil }
				mock.CheckoutBranchFunc = func(branchName string) error { return nil }
				mock.PullLatestFunc = func() error { return nil }
			},
			shouldSucceed: true,
		},
		{
			name:       "Failure - reset hard fails",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return errors.New("reset failed") }
			},
			expectedError: "failed to reset hard",
		},
		{
			name:       "Failure - clean untracked fails",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return nil }
				mock.CleanUntrackedFunc = func() error { return errors.New("clean failed") }
			},
			expectedError: "failed to clean untracked files",
		},
		{
			name:       "Failure - get default branch fails",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return nil }
				mock.CleanUntrackedFunc = func() error { return nil }
				mock.GetDefaultBranchFunc = func() (string, error) { return "", errors.New("no default branch") }
			},
			expectedError: "failed to get default branch",
		},
		{
			name:       "Failure - checkout default branch fails",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return nil }
				mock.CleanUntrackedFunc = func() error { return nil }
				mock.GetDefaultBranchFunc = func() (string, error) { return "main", nil }
				mock.CheckoutBranchFunc = func(branchName string) error {
					if branchName == "main" {
						return errors.New("checkout failed")
					}
					return nil
				}
			},
			expectedError: "failed to checkout default branch main",
		},
		{
			name:       "Failure - pull latest fails",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return nil }
				mock.CleanUntrackedFunc = func() error { return nil }
				mock.GetDefaultBranchFunc = func() (string, error) { return "main", nil }
				mock.CheckoutBranchFunc = func(branchName string) error {
					if branchName == "main" {
						return nil
					}
					return nil
				}
				mock.PullLatestFunc = func() error { return errors.New("pull failed") }
			},
			expectedError: "failed to pull latest changes",
		},
		{
			name:       "Failure - checkout target branch fails",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return nil }
				mock.CleanUntrackedFunc = func() error { return nil }
				mock.GetDefaultBranchFunc = func() (string, error) { return "main", nil }
				mock.CheckoutBranchFunc = func(branchName string) error {
					if branchName == "main" {
						return nil
					}
					return errors.New("checkout target failed")
				}
				mock.PullLatestFunc = func() error { return nil }
			},
			expectedError: "failed to checkout target branch feature-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitClient{}
			mockClaude := &MockClaudeService{}
			mockAppState := &MockAppState{}

			tt.setupMock(mockGit)

			useCase := NewGitUseCase(mockGit, mockClaude, mockAppState)
			err := useCase.SwitchToJobBranch(tt.branchName)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestGitUseCase_PrepareForNewConversation(t *testing.T) {
	tests := []struct {
		name               string
		conversationHint   string
		setupMock          func(*MockGitClient)
		expectedError      string
		shouldSucceed      bool
		expectedBranchName string
	}{
		{
			name:             "Success - prepare new conversation",
			conversationHint: "test hint",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return nil }
				mock.CleanUntrackedFunc = func() error { return nil }
				mock.GetDefaultBranchFunc = func() (string, error) { return "main", nil }
				mock.CheckoutBranchFunc = func(branchName string) error { return nil }
				mock.PullLatestFunc = func() error { return nil }
				mock.CreateAndCheckoutBranchFunc = func(branchName string) error { return nil }
			},
			shouldSucceed: true,
		},
		{
			name:             "Failure - reset and pull default branch fails",
			conversationHint: "test hint",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return errors.New("reset failed") }
			},
			expectedError: "failed to reset and pull main",
		},
		{
			name:             "Failure - create and checkout branch fails",
			conversationHint: "test hint",
			setupMock: func(mock *MockGitClient) {
				mock.ResetHardFunc = func() error { return nil }
				mock.CleanUntrackedFunc = func() error { return nil }
				mock.GetDefaultBranchFunc = func() (string, error) { return "main", nil }
				mock.CheckoutBranchFunc = func(branchName string) error { return nil }
				mock.PullLatestFunc = func() error { return nil }
				mock.CreateAndCheckoutBranchFunc = func(branchName string) error {
					return errors.New("create branch failed")
				}
			},
			expectedError: "failed to create and checkout new branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitClient{}
			mockClaude := &MockClaudeService{}
			mockAppState := &MockAppState{}

			tt.setupMock(mockGit)

			useCase := NewGitUseCase(mockGit, mockClaude, mockAppState)
			branchName, err := useCase.PrepareForNewConversation(tt.conversationHint)

			if tt.shouldSucceed {
				assert.NoError(t, err)
				assert.NotEmpty(t, branchName)
				assert.Contains(t, branchName, "ccagent/")
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, branchName)
			}
		})
	}
}

func TestGitUseCase_AutoCommitChangesIfNeeded(t *testing.T) {
	tests := []struct {
		name              string
		slackThreadLink   string
		setupMock         func(*MockGitClient, *MockClaudeService)
		expectedError     string
		shouldSucceed     bool
		expectPRCreated   bool
		expectNoChanges   bool
	}{
		{
			name:            "Success - no changes to commit",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mockGit *MockGitClient, mockClaude *MockClaudeService) {
				mockGit.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mockGit.HasUncommittedChangesFunc = func() (bool, error) { return false, nil }
			},
			shouldSucceed:   true,
			expectNoChanges: true,
		},
		{
			name:            "Success - commit changes and create new PR",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mockGit *MockGitClient, mockClaude *MockClaudeService) {
				mockGit.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mockGit.HasUncommittedChangesFunc = func() (bool, error) { return true, nil }
				mockGit.AddAllFunc = func() error { return nil }
				mockGit.CommitFunc = func(message string) error { return nil }
				mockGit.GetLatestCommitHashFunc = func() (string, error) { return "abc123", nil }
				mockGit.GetRemoteURLFunc = func() (string, error) { return "https://github.com/user/repo", nil }
				mockGit.PushBranchFunc = func(branchName string) error { return nil }
				mockGit.HasExistingPRFunc = func(branchName string) (bool, error) { return false, nil }
				mockGit.GetDefaultBranchFunc = func() (string, error) { return "main", nil }
				mockGit.GetBranchCommitMessagesFunc = func(branchName, baseBranch string) ([]string, error) {
					return []string{"Test commit"}, nil
				}
				mockGit.GetBranchDiffSummaryFunc = func(branchName, baseBranch string) (string, error) {
					return "M\tfile.go", nil
				}
				mockGit.GetBranchDiffContentFunc = func(branchName, baseBranch string) (string, error) {
					return "diff --git a/file.go b/file.go\n+added line", nil
				}
				mockGit.CreatePullRequestFunc = func(title, body, baseBranch string) (string, error) {
					return "https://github.com/user/repo/pull/123", nil
				}
				mockGit.ExtractPRIDFromURLFunc = func(prURL string) string { return "123" }

				mockClaude.StartNewConversationFunc = func(prompt string) (*services.ClaudeResult, error) {
					return &services.ClaudeResult{Output: "Fix user authentication"}, nil
				}
			},
			shouldSucceed:   true,
			expectPRCreated: true,
		},
		{
			name:            "Success - commit changes and update existing PR",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mockGit *MockGitClient, mockClaude *MockClaudeService) {
				mockGit.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mockGit.HasUncommittedChangesFunc = func() (bool, error) { return true, nil }
				mockGit.AddAllFunc = func() error { return nil }
				mockGit.CommitFunc = func(message string) error { return nil }
				mockGit.GetLatestCommitHashFunc = func() (string, error) { return "abc123", nil }
				mockGit.GetRemoteURLFunc = func() (string, error) { return "https://github.com/user/repo", nil }
				mockGit.PushBranchFunc = func(branchName string) error { return nil }
				mockGit.HasExistingPRFunc = func(branchName string) (bool, error) { return true, nil }
				mockGit.GetPRURLFunc = func(branchName string) (string, error) {
					return "https://github.com/user/repo/pull/123", nil
				}
				mockGit.ExtractPRIDFromURLFunc = func(prURL string) string { return "123" }

				mockClaude.StartNewConversationFunc = func(prompt string) (*services.ClaudeResult, error) {
					return &services.ClaudeResult{Output: "Fix user authentication"}, nil
				}
			},
			shouldSucceed:   true,
			expectPRCreated: false,
		},
		{
			name:            "Failure - get current branch fails",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mockGit *MockGitClient, mockClaude *MockClaudeService) {
				mockGit.GetCurrentBranchFunc = func() (string, error) {
					return "", errors.New("no current branch")
				}
			},
			expectedError: "failed to get current branch",
		},
		{
			name:            "Failure - check uncommitted changes fails",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mockGit *MockGitClient, mockClaude *MockClaudeService) {
				mockGit.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mockGit.HasUncommittedChangesFunc = func() (bool, error) {
					return false, errors.New("git status failed")
				}
			},
			expectedError: "failed to check for uncommitted changes",
		},
		{
			name:            "Failure - claude fails to generate commit message",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mockGit *MockGitClient, mockClaude *MockClaudeService) {
				mockGit.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mockGit.HasUncommittedChangesFunc = func() (bool, error) { return true, nil }

				mockClaude.StartNewConversationFunc = func(prompt string) (*services.ClaudeResult, error) {
					return nil, errors.New("claude service failed")
				}
			},
			expectedError: "failed to generate commit message with Claude",
		},
		{
			name:            "Failure - add all fails",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mockGit *MockGitClient, mockClaude *MockClaudeService) {
				mockGit.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mockGit.HasUncommittedChangesFunc = func() (bool, error) { return true, nil }
				mockGit.AddAllFunc = func() error { return errors.New("git add failed") }

				mockClaude.StartNewConversationFunc = func(prompt string) (*services.ClaudeResult, error) {
					return &services.ClaudeResult{Output: "Fix user authentication"}, nil
				}
			},
			expectedError: "failed to add all changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitClient{}
			mockClaude := &MockClaudeService{}
			mockAppState := &MockAppState{}

			tt.setupMock(mockGit, mockClaude)

			useCase := NewGitUseCase(mockGit, mockClaude, mockAppState)
			result, err := useCase.AutoCommitChangesIfNeeded(tt.slackThreadLink)

			if tt.shouldSucceed {
				assert.NoError(t, err)
				require.NotNil(t, result)

				if tt.expectNoChanges {
					assert.False(t, result.JustCreatedPR)
					assert.Empty(t, result.PullRequestLink)
					assert.Empty(t, result.CommitHash)
				} else {
					assert.Equal(t, tt.expectPRCreated, result.JustCreatedPR)
					assert.NotEmpty(t, result.PullRequestLink)
					assert.NotEmpty(t, result.CommitHash)
					assert.NotEmpty(t, result.RepositoryURL)
					assert.NotEmpty(t, result.BranchName)
				}
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			}
		})
	}
}

func TestGitUseCase_CheckPRStatus(t *testing.T) {
	tests := []struct {
		name           string
		branchName     string
		setupMock      func(*MockGitClient)
		expectedError  string
		expectedStatus string
		shouldSucceed  bool
	}{
		{
			name:       "Success - PR exists and is open",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.HasExistingPRFunc = func(branchName string) (bool, error) { return true, nil }
				mock.GetPRStateFunc = func(branchName string) (string, error) { return "open", nil }
			},
			shouldSucceed:  true,
			expectedStatus: "open",
		},
		{
			name:       "Success - PR exists and is merged",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.HasExistingPRFunc = func(branchName string) (bool, error) { return true, nil }
				mock.GetPRStateFunc = func(branchName string) (string, error) { return "merged", nil }
			},
			shouldSucceed:  true,
			expectedStatus: "merged",
		},
		{
			name:       "Success - no PR exists",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.HasExistingPRFunc = func(branchName string) (bool, error) { return false, nil }
			},
			shouldSucceed:  true,
			expectedStatus: "no_pr",
		},
		{
			name:       "Failure - check existing PR fails",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.HasExistingPRFunc = func(branchName string) (bool, error) {
					return false, errors.New("github api failed")
				}
			},
			expectedError: "failed to check for existing PR",
		},
		{
			name:       "Failure - get PR state fails",
			branchName: "feature-branch",
			setupMock: func(mock *MockGitClient) {
				mock.HasExistingPRFunc = func(branchName string) (bool, error) { return true, nil }
				mock.GetPRStateFunc = func(branchName string) (string, error) {
					return "", errors.New("github api failed")
				}
			},
			expectedError: "failed to get PR state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitClient{}
			mockClaude := &MockClaudeService{}
			mockAppState := &MockAppState{}

			tt.setupMock(mockGit)

			useCase := NewGitUseCase(mockGit, mockClaude, mockAppState)
			status, err := useCase.CheckPRStatus(tt.branchName)

			if tt.shouldSucceed {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, status)
			}
		})
	}
}

func TestGitUseCase_CheckPRStatusByID(t *testing.T) {
	tests := []struct {
		name           string
		prID           string
		setupMock      func(*MockGitClient)
		expectedError  string
		expectedStatus string
		shouldSucceed  bool
	}{
		{
			name: "Success - PR exists and is open",
			prID: "123",
			setupMock: func(mock *MockGitClient) {
				mock.GetPRStateByIDFunc = func(prID string) (string, error) { return "open", nil }
			},
			shouldSucceed:  true,
			expectedStatus: "open",
		},
		{
			name: "Success - PR exists and is closed",
			prID: "123",
			setupMock: func(mock *MockGitClient) {
				mock.GetPRStateByIDFunc = func(prID string) (string, error) { return "closed", nil }
			},
			shouldSucceed:  true,
			expectedStatus: "closed",
		},
		{
			name: "Failure - get PR state by ID fails",
			prID: "123",
			setupMock: func(mock *MockGitClient) {
				mock.GetPRStateByIDFunc = func(prID string) (string, error) {
					return "", errors.New("PR not found")
				}
			},
			expectedError: "failed to get PR state by ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitClient{}
			mockClaude := &MockClaudeService{}
			mockAppState := &MockAppState{}

			tt.setupMock(mockGit)

			useCase := NewGitUseCase(mockGit, mockClaude, mockAppState)
			status, err := useCase.CheckPRStatusByID(tt.prID)

			if tt.shouldSucceed {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, status)
			}
		})
	}
}

func TestGitUseCase_CleanupStaleBranches(t *testing.T) {
	tests := []struct {
		name              string
		setupMock         func(*MockGitClient, *MockAppState)
		expectedError     string
		shouldSucceed     bool
		expectBranchesDeleted bool
	}{
		{
			name: "Success - no stale branches",
			setupMock: func(mockGit *MockGitClient, mockAppState *MockAppState) {
				mockGit.GetLocalBranchesFunc = func() ([]string, error) {
					return []string{"main", "develop"}, nil
				}
				mockGit.GetCurrentBranchFunc = func() (string, error) { return "main", nil }
				mockGit.GetDefaultBranchFunc = func() (string, error) { return "main", nil }
				mockAppState.GetAllJobsFunc = func() map[string]*models.JobData {
					return make(map[string]*models.JobData)
				}
			},
			shouldSucceed: true,
		},
		{
			name: "Success - cleanup stale ccagent branches",
			setupMock: func(mockGit *MockGitClient, mockAppState *MockAppState) {
				mockGit.GetLocalBranchesFunc = func() ([]string, error) {
					return []string{"main", "ccagent/old-branch-1", "ccagent/old-branch-2", "ccagent/tracked-branch"}, nil
				}
				mockGit.GetCurrentBranchFunc = func() (string, error) { return "main", nil }
				mockGit.GetDefaultBranchFunc = func() (string, error) { return "main", nil }
				mockGit.DeleteLocalBranchFunc = func(branchName string) error { return nil }
				mockAppState.GetAllJobsFunc = func() map[string]*models.JobData {
					return map[string]*models.JobData{
						"job1": {BranchName: "ccagent/tracked-branch"},
					}
				}
			},
			shouldSucceed:         true,
			expectBranchesDeleted: true,
		},
		{
			name: "Failure - get local branches fails",
			setupMock: func(mockGit *MockGitClient, mockAppState *MockAppState) {
				mockGit.GetLocalBranchesFunc = func() ([]string, error) {
					return nil, errors.New("git branch failed")
				}
			},
			expectedError: "failed to get local branches",
		},
		{
			name: "Failure - get current branch fails",
			setupMock: func(mockGit *MockGitClient, mockAppState *MockAppState) {
				mockGit.GetLocalBranchesFunc = func() ([]string, error) {
					return []string{"main", "ccagent/old-branch"}, nil
				}
				mockGit.GetCurrentBranchFunc = func() (string, error) {
					return "", errors.New("no current branch")
				}
			},
			expectedError: "failed to get current branch",
		},
		{
			name: "Failure - get default branch fails",
			setupMock: func(mockGit *MockGitClient, mockAppState *MockAppState) {
				mockGit.GetLocalBranchesFunc = func() ([]string, error) {
					return []string{"main", "ccagent/old-branch"}, nil
				}
				mockGit.GetCurrentBranchFunc = func() (string, error) { return "main", nil }
				mockGit.GetDefaultBranchFunc = func() (string, error) {
					return "", errors.New("no default branch")
				}
			},
			expectedError: "failed to get default branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitClient{}
			mockClaude := &MockClaudeService{}
			mockAppState := &MockAppState{}

			tt.setupMock(mockGit, mockAppState)

			useCase := NewGitUseCase(mockGit, mockClaude, mockAppState)
			err := useCase.CleanupStaleBranches()

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestGitUseCase_ValidateAndRestorePRDescriptionFooter(t *testing.T) {
	tests := []struct {
		name              string
		slackThreadLink   string
		setupMock         func(*MockGitClient)
		expectedError     string
		shouldSucceed     bool
		expectUpdate      bool
	}{
		{
			name:            "Success - no PR exists",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mock *MockGitClient) {
				mock.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mock.HasExistingPRFunc = func(branchName string) (bool, error) { return false, nil }
			},
			shouldSucceed: true,
		},
		{
			name:            "Success - PR has correct footer already",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mock *MockGitClient) {
				mock.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mock.HasExistingPRFunc = func(branchName string) (bool, error) { return true, nil }
				mock.GetPRDescriptionFunc = func(branchName string) (string, error) {
					return "PR body\n\n---\nGenerated with [Claude Control](https://claudecontrol.com) from this [slack thread](https://slack.com/thread/123)", nil
				}
			},
			shouldSucceed: true,
		},
		{
			name:            "Success - PR needs footer restoration",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mock *MockGitClient) {
				mock.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mock.HasExistingPRFunc = func(branchName string) (bool, error) { return true, nil }
				mock.GetPRDescriptionFunc = func(branchName string) (string, error) {
					return "PR body without footer", nil
				}
				mock.UpdatePRDescriptionFunc = func(branchName, newDescription string) error { return nil }
			},
			shouldSucceed: true,
			expectUpdate:  true,
		},
		{
			name:            "Failure - get current branch fails",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mock *MockGitClient) {
				mock.GetCurrentBranchFunc = func() (string, error) {
					return "", errors.New("no current branch")
				}
			},
			expectedError: "failed to get current branch",
		},
		{
			name:            "Failure - check existing PR fails",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mock *MockGitClient) {
				mock.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mock.HasExistingPRFunc = func(branchName string) (bool, error) {
					return false, errors.New("github api failed")
				}
			},
			expectedError: "failed to check for existing PR",
		},
		{
			name:            "Failure - get PR description fails",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mock *MockGitClient) {
				mock.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mock.HasExistingPRFunc = func(branchName string) (bool, error) { return true, nil }
				mock.GetPRDescriptionFunc = func(branchName string) (string, error) {
					return "", errors.New("github api failed")
				}
			},
			expectedError: "failed to get PR description",
		},
		{
			name:            "Failure - update PR description fails",
			slackThreadLink: "https://slack.com/thread/123",
			setupMock: func(mock *MockGitClient) {
				mock.GetCurrentBranchFunc = func() (string, error) { return "feature-branch", nil }
				mock.HasExistingPRFunc = func(branchName string) (bool, error) { return true, nil }
				mock.GetPRDescriptionFunc = func(branchName string) (string, error) {
					return "PR body without footer", nil
				}
				mock.UpdatePRDescriptionFunc = func(branchName, newDescription string) error {
					return errors.New("github api failed")
				}
			},
			expectedError: "failed to update PR description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitClient{}
			mockClaude := &MockClaudeService{}
			mockAppState := &MockAppState{}

			tt.setupMock(mockGit)

			useCase := NewGitUseCase(mockGit, mockClaude, mockAppState)
			err := useCase.ValidateAndRestorePRDescriptionFooter(tt.slackThreadLink)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}