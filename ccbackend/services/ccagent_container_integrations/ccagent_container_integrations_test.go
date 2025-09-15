package ccagentcontainerintegrations

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/clients/ssh"
	"ccbackend/config"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services/anthropic_integrations"
	"ccbackend/services/github_integrations"
	"ccbackend/services/organizations"
	"ccbackend/testutils"
)

func setupCCAgentContainerIntegrationsTest(t *testing.T) (*CCAgentContainerIntegrationsService, *db.PostgresOrganizationsRepository, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)

	repo := db.NewPostgresCCAgentContainerIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	orgsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)

	// Create mock services
	mockGithubService := &github_integrations.MockGitHubIntegrationsService{}
	mockAnthropicService := &anthropic_integrations.MockAnthropicIntegrationsService{}
	mockOrganizationsService := &organizations.MockOrganizationsService{}
	mockSSHClient := &ssh.MockSSHClient{}

	// Create app config with test SSH settings
	appConfig := &config.AppConfig{
		SSHConfig: config.SSHConfig{
			DefaultHost: "test-ssh-host.example.com",
		},
	}

	service := NewCCAgentContainerIntegrationsService(
		repo,
		appConfig,
		mockGithubService,
		mockAnthropicService,
		mockOrganizationsService,
		mockSSHClient,
	)

	cleanup := func() {
		dbConn.Close()
	}

	return service, orgsRepo, cleanup
}

func TestCCAgentContainerIntegrationsService_CreateCCAgentContainerIntegration(t *testing.T) {
	service, orgsRepo, cleanup := setupCCAgentContainerIntegrationsTest(t)
	defer cleanup()

	t.Run("successful integration creation", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Test with valid parameters
		integration, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			3,
			"https://github.com/example/repo",
		)

		require.NoError(t, err)
		assert.NotNil(t, integration)
		assert.Equal(t, 3, integration.InstancesCount)
		assert.Equal(t, "github.com/example/repo", integration.RepoURL) // Should be sanitized
		assert.Equal(t, "test-ssh-host.example.com", integration.SSHHost)
		assert.Equal(t, models.OrgID(testOrg.ID), integration.OrgID)
		assert.True(t, core.IsValidULID(integration.ID))
		assert.Contains(t, integration.ID, "cci_")

		// Clean up
		defer func() {
			err := service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration.ID)
			assert.NoError(t, err)
		}()
	})

	t.Run("URL sanitization", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		testCases := []struct {
			inputURL    string
			expectedURL string
		}{
			{"https://github.com/example/repo", "github.com/example/repo"},
			{"http://github.com/example/repo", "github.com/example/repo"},
			{"github.com/example/repo", "github.com/example/repo"},
			{"https://gitlab.com/group/project", "gitlab.com/group/project"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("URL_%s", tc.inputURL), func(t *testing.T) {
				integration, err := service.CreateCCAgentContainerIntegration(
					context.Background(),
					models.OrgID(testOrg.ID),
					1,
					tc.inputURL,
				)

				require.NoError(t, err)
				assert.Equal(t, tc.expectedURL, integration.RepoURL)

				// Clean up
				defer func() {
					err := service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration.ID)
					assert.NoError(t, err)
				}()
			})
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		testCases := []struct {
			name           string
			instancesCount int
			repoURL        string
			expectedError  string
		}{
			{
				name:           "instances count too low",
				instancesCount: 0,
				repoURL:        "github.com/example/repo",
				expectedError:  "instances_count must be between 1 and 10",
			},
			{
				name:           "instances count too high",
				instancesCount: 11,
				repoURL:        "github.com/example/repo",
				expectedError:  "instances_count must be between 1 and 10",
			},
			{
				name:           "empty repo URL",
				instancesCount: 1,
				repoURL:        "",
				expectedError:  "repo_url cannot be empty",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := service.CreateCCAgentContainerIntegration(
					context.Background(),
					models.OrgID(testOrg.ID),
					tc.instancesCount,
					tc.repoURL,
				)

				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			})
		}
	})
}

func TestCCAgentContainerIntegrationsService_ListCCAgentContainerIntegrations(t *testing.T) {
	service, orgsRepo, cleanup := setupCCAgentContainerIntegrationsTest(t)
	defer cleanup()

	t.Run("list empty integrations", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		integrations, err := service.ListCCAgentContainerIntegrations(
			context.Background(),
			models.OrgID(testOrg.ID),
		)

		require.NoError(t, err)
		assert.Empty(t, integrations)
	})

	t.Run("list multiple integrations", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Create test integrations
		integration1, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			2,
			"github.com/example/repo1",
		)
		require.NoError(t, err)

		integration2, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			5,
			"github.com/example/repo2",
		)
		require.NoError(t, err)

		// List integrations
		integrations, err := service.ListCCAgentContainerIntegrations(
			context.Background(),
			models.OrgID(testOrg.ID),
		)

		require.NoError(t, err)
		assert.Len(t, integrations, 2)

		// Find our integrations in the results
		var found1, found2 bool
		for _, integration := range integrations {
			if integration.ID == integration1.ID {
				assert.Equal(t, 2, integration.InstancesCount)
				assert.Equal(t, "github.com/example/repo1", integration.RepoURL)
				found1 = true
			}
			if integration.ID == integration2.ID {
				assert.Equal(t, 5, integration.InstancesCount)
				assert.Equal(t, "github.com/example/repo2", integration.RepoURL)
				found2 = true
			}
		}
		assert.True(t, found1, "Integration 1 not found in results")
		assert.True(t, found2, "Integration 2 not found in results")

		// Clean up
		defer func() {
			service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration1.ID)
			service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration2.ID)
		}()
	})
}

func TestCCAgentContainerIntegrationsService_GetCCAgentContainerIntegrationByID(t *testing.T) {
	service, orgsRepo, cleanup := setupCCAgentContainerIntegrationsTest(t)
	defer cleanup()

	t.Run("get existing integration", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Create test integration
		createdIntegration, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			4,
			"github.com/example/test-repo",
		)
		require.NoError(t, err)

		// Get integration by ID
		integrationOpt, err := service.GetCCAgentContainerIntegrationByID(
			context.Background(),
			models.OrgID(testOrg.ID),
			createdIntegration.ID,
		)

		require.NoError(t, err)
		assert.True(t, integrationOpt.IsPresent())

		integration := integrationOpt.MustGet()
		assert.Equal(t, createdIntegration.ID, integration.ID)
		assert.Equal(t, 4, integration.InstancesCount)
		assert.Equal(t, "github.com/example/test-repo", integration.RepoURL)
		assert.Equal(t, models.OrgID(testOrg.ID), integration.OrgID)

		// Clean up
		defer func() {
			service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration.ID)
		}()
	})

	t.Run("get non-existent integration", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		nonExistentID := core.NewID("cci")

		integrationOpt, err := service.GetCCAgentContainerIntegrationByID(
			context.Background(),
			models.OrgID(testOrg.ID),
			nonExistentID,
		)

		require.NoError(t, err)
		assert.False(t, integrationOpt.IsPresent())
	})

	t.Run("invalid integration ID", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		_, err := service.GetCCAgentContainerIntegrationByID(
			context.Background(),
			models.OrgID(testOrg.ID),
			"invalid-id",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid integration ID")
	})
}

func TestCCAgentContainerIntegrationsService_DeleteCCAgentContainerIntegration(t *testing.T) {
	service, orgsRepo, cleanup := setupCCAgentContainerIntegrationsTest(t)
	defer cleanup()

	t.Run("successful deletion", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Create test integration
		integration, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			1,
			"github.com/example/to-delete",
		)
		require.NoError(t, err)

		// Delete integration
		err = service.DeleteCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			integration.ID,
		)
		require.NoError(t, err)

		// Verify it's deleted
		integrationOpt, err := service.GetCCAgentContainerIntegrationByID(
			context.Background(),
			models.OrgID(testOrg.ID),
			integration.ID,
		)
		require.NoError(t, err)
		assert.False(t, integrationOpt.IsPresent())
	})

	t.Run("invalid integration ID", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		err := service.DeleteCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			"invalid-id",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid integration ID")
	})
}

func TestCCAgentContainerIntegrationsService_RedeployCCAgentContainer(t *testing.T) {
	service, orgsRepo, cleanup := setupCCAgentContainerIntegrationsTest(t)
	defer cleanup()

	t.Run("successful redeploy with API key", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Create test integration
		integration, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			2,
			"github.com/example/redeploy-repo",
		)
		require.NoError(t, err)

		// Setup mock expectations
		mockOrganizationsService := service.organizationsService.(*organizations.MockOrganizationsService)
		mockGithubService := service.githubIntegrationsService.(*github_integrations.MockGitHubIntegrationsService)
		mockAnthropicService := service.anthropicIntegrationsService.(*anthropic_integrations.MockAnthropicIntegrationsService)
		mockSSHClient := service.sshClient.(*ssh.MockSSHClient)

		// Mock organization with CCAgent secret key
		testOrgModel := &models.Organization{
			ID:                     testOrg.ID,
			CCAgentSystemSecretKey: "test-secret-key-123",
		}
		mockOrganizationsService.On("GetOrganizationByID", context.Background(), testOrg.ID).
			Return(mo.Some(testOrgModel), nil)

		// Mock GitHub integration
		githubIntegration := models.GitHubIntegration{
			GitHubInstallationID: "12345",
		}
		mockGithubService.On("ListGitHubIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.GitHubIntegration{githubIntegration}, nil)

		// Mock Anthropic integration with API key
		apiKey := "sk-ant-api-key-123"
		anthropicIntegration := models.AnthropicIntegration{
			AnthropicAPIKey: &apiKey,
		}
		mockAnthropicService.On("ListAnthropicIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.AnthropicIntegration{anthropicIntegration}, nil)

		// Mock SSH execution
		expectedCommand := fmt.Sprintf("/root/scripts/redeployccagent.sh -n 'ccagent-%s' -k 'test-secret-key-123' -r 'github.com/example/redeploy-repo' -i '12345' -a 'sk-ant-api-key-123'", integration.ID)
		mockSSHClient.On("ExecuteCommand", "test-ssh-host.example.com", expectedCommand).
			Return(nil)

		// Execute redeploy
		err = service.RedeployCCAgentContainer(
			context.Background(),
			models.OrgID(testOrg.ID),
			integration.ID,
			false,
		)

		require.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
		mockGithubService.AssertExpectations(t)
		mockAnthropicService.AssertExpectations(t)
		mockSSHClient.AssertExpectations(t)

		// Clean up
		defer func() {
			service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration.ID)
		}()
	})

	t.Run("successful redeploy with OAuth token", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Create test integration
		integration, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			1,
			"github.com/example/oauth-repo",
		)
		require.NoError(t, err)

		// Setup mock expectations
		mockOrganizationsService := service.organizationsService.(*organizations.MockOrganizationsService)
		mockGithubService := service.githubIntegrationsService.(*github_integrations.MockGitHubIntegrationsService)
		mockAnthropicService := service.anthropicIntegrationsService.(*anthropic_integrations.MockAnthropicIntegrationsService)
		mockSSHClient := service.sshClient.(*ssh.MockSSHClient)

		// Mock organization
		testOrgModel := &models.Organization{
			ID:                     testOrg.ID,
			CCAgentSystemSecretKey: "test-oauth-secret-456",
		}
		mockOrganizationsService.On("GetOrganizationByID", context.Background(), testOrg.ID).
			Return(mo.Some(testOrgModel), nil)

		// Mock GitHub integration
		githubIntegration := models.GitHubIntegration{
			GitHubInstallationID: "67890",
		}
		mockGithubService.On("ListGitHubIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.GitHubIntegration{githubIntegration}, nil)

		// Mock Anthropic integration with OAuth token
		oauthToken := "oauth-token-xyz-789"
		anthropicIntegration := models.AnthropicIntegration{
			ClaudeCodeOAuthToken: &oauthToken,
		}
		mockAnthropicService.On("ListAnthropicIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.AnthropicIntegration{anthropicIntegration}, nil)

		// Mock SSH execution
		expectedCommand := fmt.Sprintf("/root/scripts/redeployccagent.sh -n 'ccagent-%s' -k 'test-oauth-secret-456' -r 'github.com/example/oauth-repo' -i '67890' -o 'oauth-token-xyz-789'", integration.ID)
		mockSSHClient.On("ExecuteCommand", "test-ssh-host.example.com", expectedCommand).
			Return(nil)

		// Execute redeploy
		err = service.RedeployCCAgentContainer(
			context.Background(),
			models.OrgID(testOrg.ID),
			integration.ID,
			false,
		)

		require.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
		mockGithubService.AssertExpectations(t)
		mockAnthropicService.AssertExpectations(t)
		mockSSHClient.AssertExpectations(t)

		// Clean up
		defer func() {
			service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration.ID)
		}()
	})

	t.Run("config-only redeploy", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Create test integration
		integration, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			1,
			"github.com/example/config-only",
		)
		require.NoError(t, err)

		// Setup mock expectations
		mockOrganizationsService := service.organizationsService.(*organizations.MockOrganizationsService)
		mockGithubService := service.githubIntegrationsService.(*github_integrations.MockGitHubIntegrationsService)
		mockAnthropicService := service.anthropicIntegrationsService.(*anthropic_integrations.MockAnthropicIntegrationsService)
		mockSSHClient := service.sshClient.(*ssh.MockSSHClient)

		// Mock organization
		testOrgModel := &models.Organization{
			ID:                     testOrg.ID,
			CCAgentSystemSecretKey: "test-config-secret-789",
		}
		mockOrganizationsService.On("GetOrganizationByID", context.Background(), testOrg.ID).
			Return(mo.Some(testOrgModel), nil)

		// Mock GitHub integration
		githubIntegration := models.GitHubIntegration{
			GitHubInstallationID: "11111",
		}
		mockGithubService.On("ListGitHubIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.GitHubIntegration{githubIntegration}, nil)

		// Mock Anthropic integration
		apiKey := "sk-ant-config-key"
		anthropicIntegration := models.AnthropicIntegration{
			AnthropicAPIKey: &apiKey,
		}
		mockAnthropicService.On("ListAnthropicIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.AnthropicIntegration{anthropicIntegration}, nil)

		// Mock SSH execution with --config-only flag
		expectedCommand := fmt.Sprintf("/root/scripts/redeployccagent.sh -n 'ccagent-%s' -k 'test-config-secret-789' -r 'github.com/example/config-only' -i '11111' -a 'sk-ant-config-key' --config-only", integration.ID)
		mockSSHClient.On("ExecuteCommand", "test-ssh-host.example.com", expectedCommand).
			Return(nil)

		// Execute config-only redeploy
		err = service.RedeployCCAgentContainer(
			context.Background(),
			models.OrgID(testOrg.ID),
			integration.ID,
			true,
		)

		require.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
		mockGithubService.AssertExpectations(t)
		mockAnthropicService.AssertExpectations(t)
		mockSSHClient.AssertExpectations(t)

		// Clean up
		defer func() {
			service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration.ID)
		}()
	})

	t.Run("invalid integration ID", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		err := service.RedeployCCAgentContainer(
			context.Background(),
			models.OrgID(testOrg.ID),
			"invalid-id",
			false,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid integration ID")
	})

	t.Run("integration not found", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		nonExistentID := core.NewID("cci")

		err := service.RedeployCCAgentContainer(
			context.Background(),
			models.OrgID(testOrg.ID),
			nonExistentID,
			false,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "CCAgent container integration not found")
	})

	t.Run("no GitHub integration", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Create test integration
		integration, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			1,
			"github.com/example/no-github",
		)
		require.NoError(t, err)

		// Setup mock expectations
		mockOrganizationsService := service.organizationsService.(*organizations.MockOrganizationsService)
		mockGithubService := service.githubIntegrationsService.(*github_integrations.MockGitHubIntegrationsService)

		// Mock organization
		testOrgModel := &models.Organization{
			ID:                     testOrg.ID,
			CCAgentSystemSecretKey: "test-secret",
		}
		mockOrganizationsService.On("GetOrganizationByID", context.Background(), testOrg.ID).
			Return(mo.Some(testOrgModel), nil)

		// Mock empty GitHub integrations
		mockGithubService.On("ListGitHubIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.GitHubIntegration{}, nil)

		err = service.RedeployCCAgentContainer(
			context.Background(),
			models.OrgID(testOrg.ID),
			integration.ID,
			false,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no GitHub integration found for organization")

		// Clean up
		defer func() {
			service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration.ID)
		}()
	})

	t.Run("no Anthropic integration", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Create test integration
		integration, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			1,
			"github.com/example/no-anthropic",
		)
		require.NoError(t, err)

		// Setup mock expectations
		mockOrganizationsService := service.organizationsService.(*organizations.MockOrganizationsService)
		mockGithubService := service.githubIntegrationsService.(*github_integrations.MockGitHubIntegrationsService)
		mockAnthropicService := service.anthropicIntegrationsService.(*anthropic_integrations.MockAnthropicIntegrationsService)

		// Mock organization
		testOrgModel := &models.Organization{
			ID:                     testOrg.ID,
			CCAgentSystemSecretKey: "test-secret",
		}
		mockOrganizationsService.On("GetOrganizationByID", context.Background(), testOrg.ID).
			Return(mo.Some(testOrgModel), nil)

		// Mock GitHub integration
		githubIntegration := models.GitHubIntegration{
			GitHubInstallationID: "12345",
		}
		mockGithubService.On("ListGitHubIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.GitHubIntegration{githubIntegration}, nil)

		// Mock empty Anthropic integrations
		mockAnthropicService.On("ListAnthropicIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.AnthropicIntegration{}, nil)

		err = service.RedeployCCAgentContainer(
			context.Background(),
			models.OrgID(testOrg.ID),
			integration.ID,
			false,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no Anthropic integration found for organization")

		// Clean up
		defer func() {
			service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration.ID)
		}()
	})

	t.Run("Anthropic integration with no authentication", func(t *testing.T) {
		testOrg := testutils.CreateTestOrganization(t, orgsRepo)

		// Create test integration
		integration, err := service.CreateCCAgentContainerIntegration(
			context.Background(),
			models.OrgID(testOrg.ID),
			1,
			"github.com/example/no-auth",
		)
		require.NoError(t, err)

		// Setup mock expectations
		mockOrganizationsService := service.organizationsService.(*organizations.MockOrganizationsService)
		mockGithubService := service.githubIntegrationsService.(*github_integrations.MockGitHubIntegrationsService)
		mockAnthropicService := service.anthropicIntegrationsService.(*anthropic_integrations.MockAnthropicIntegrationsService)

		// Mock organization
		testOrgModel := &models.Organization{
			ID:                     testOrg.ID,
			CCAgentSystemSecretKey: "test-secret",
		}
		mockOrganizationsService.On("GetOrganizationByID", context.Background(), testOrg.ID).
			Return(mo.Some(testOrgModel), nil)

		// Mock GitHub integration
		githubIntegration := models.GitHubIntegration{
			GitHubInstallationID: "12345",
		}
		mockGithubService.On("ListGitHubIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.GitHubIntegration{githubIntegration}, nil)

		// Mock Anthropic integration with no API key or OAuth token
		anthropicIntegration := models.AnthropicIntegration{
			AnthropicAPIKey:      nil,
			ClaudeCodeOAuthToken: nil,
		}
		mockAnthropicService.On("ListAnthropicIntegrations", context.Background(), models.OrgID(testOrg.ID)).
			Return([]models.AnthropicIntegration{anthropicIntegration}, nil)

		err = service.RedeployCCAgentContainer(
			context.Background(),
			models.OrgID(testOrg.ID),
			integration.ID,
			false,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "anthropic integration does not have API key or OAuth token configured")

		// Clean up
		defer func() {
			service.DeleteCCAgentContainerIntegration(context.Background(), models.OrgID(testOrg.ID), integration.ID)
		}()
	})
}
