package github_integrations

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/clients/github"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/testutils"
)

func setupGitHubIntegrationsTest(t *testing.T) (*GitHubIntegrationsService, *db.PostgresUsersRepository, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)

	repo := db.NewPostgresGitHubIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)

	mockClient := &github.MockGitHubClient{}
	service := NewGitHubIntegrationsService(repo, mockClient)

	cleanup := func() {
		dbConn.Close()
	}

	return service, usersRepo, cleanup
}

func TestGitHubIntegrationsService_CreateGitHubIntegration(t *testing.T) {
	service, usersRepo, cleanup := setupGitHubIntegrationsTest(t)
	defer cleanup()

	t.Run("successful integration creation", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Setup mock client expectations
		mockClient := &github.MockGitHubClient{}
		mockClient.On("ExchangeCodeForAccessToken", context.Background(), "test-auth-code").
			Return("ghs_test_access_token_123", nil)

		testService := NewGitHubIntegrationsService(service.githubRepo, mockClient)

		integration, err := testService.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"test-auth-code",
			"12345678",
		)

		require.NoError(t, err)
		assert.NotNil(t, integration)
		assert.NotEqual(t, "", integration.ID)
		assert.True(t, core.IsValidULID(integration.ID))
		assert.Equal(t, testUser.OrgID, integration.OrgID)
		assert.Equal(t, "12345678", integration.GitHubInstallationID)
		assert.Equal(t, "ghs_test_access_token_123", integration.GitHubAccessToken)
		assert.NotZero(t, integration.CreatedAt)
		assert.NotZero(t, integration.UpdatedAt)

		// Verify mock expectations
		mockClient.AssertExpectations(t)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.github_integrations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, integration.ID)
		}()
	})

	t.Run("empty auth code returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		integration, err := service.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"", // empty auth code
			"12345678",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "auth code cannot be empty")
	})

	t.Run("empty installation ID returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		integration, err := service.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"test-auth-code",
			"", // empty installation ID
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "installation ID cannot be empty")
	})

	t.Run("invalid organization ID returns error", func(t *testing.T) {
		integration, err := service.CreateGitHubIntegration(
			context.Background(),
			"invalid-org-id", // Invalid orgID
			"test-auth-code",
			"12345678",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})

	t.Run("GitHub not configured returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create service with nil client to simulate not configured
		unconfiguredService := NewGitHubIntegrationsService(service.githubRepo, nil)

		integration, err := unconfiguredService.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"test-auth-code",
			"12345678",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Equal(t, ErrGitHubNotConfigured, err)
	})

	t.Run("OAuth exchange error is propagated", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Setup mock client to return error
		mockClient := &github.MockGitHubClient{}
		mockClient.On("ExchangeCodeForAccessToken", context.Background(), "invalid-code").
			Return("", fmt.Errorf("invalid authorization code"))

		testService := NewGitHubIntegrationsService(service.githubRepo, mockClient)

		integration, err := testService.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"invalid-code",
			"12345678",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "failed to verify GitHub installation")
		assert.Contains(t, err.Error(), "invalid authorization code")

		// Verify mock expectations
		mockClient.AssertExpectations(t)
	})
}

func TestGitHubIntegrationsService_ListGitHubIntegrations(t *testing.T) {
	service, usersRepo, cleanup := setupGitHubIntegrationsTest(t)
	defer cleanup()

	t.Run("successful retrieval of organization integrations", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create multiple integrations for the organization
		mockClient1 := &github.MockGitHubClient{}
		mockClient1.On("ExchangeCodeForAccessToken", context.Background(), "test-auth-code-1").
			Return("ghs_token_1", nil)

		testService1 := NewGitHubIntegrationsService(service.githubRepo, mockClient1)
		integration1, err := testService1.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"test-auth-code-1",
			"12345678",
		)
		require.NoError(t, err)

		mockClient2 := &github.MockGitHubClient{}
		mockClient2.On("ExchangeCodeForAccessToken", context.Background(), "test-auth-code-2").
			Return("ghs_token_2", nil)

		testService2 := NewGitHubIntegrationsService(service.githubRepo, mockClient2)
		integration2, err := testService2.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"test-auth-code-2",
			"87654321",
		)
		require.NoError(t, err)

		// Create integration for different user to ensure isolation
		otherUser := testutils.CreateTestUser(t, usersRepo)
		mockClient3 := &github.MockGitHubClient{}
		mockClient3.On("ExchangeCodeForAccessToken", context.Background(), "test-auth-code-3").
			Return("ghs_token_3", nil)

		testService3 := NewGitHubIntegrationsService(service.githubRepo, mockClient3)
		integration3, err := testService3.CreateGitHubIntegration(
			context.Background(),
			otherUser.OrgID,
			"test-auth-code-3",
			"99999999",
		)
		require.NoError(t, err)

		// List integrations for the first user's organization
		integrations, err := service.ListGitHubIntegrations(context.Background(), testUser.OrgID)

		require.NoError(t, err)
		assert.Len(t, integrations, 2)

		// Check that we got the right integrations
		foundIDs := make(map[string]bool)
		for _, integration := range integrations {
			foundIDs[integration.ID] = true
			assert.Equal(t, testUser.OrgID, integration.OrgID)
		}
		assert.True(t, foundIDs[integration1.ID])
		assert.True(t, foundIDs[integration2.ID])
		assert.False(t, foundIDs[integration3.ID]) // Should not include other org's integration

		// Verify mock expectations
		mockClient1.AssertExpectations(t)
		mockClient2.AssertExpectations(t)
		mockClient3.AssertExpectations(t)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.github_integrations WHERE id IN ($1, $2, $3)", cfg.DatabaseSchema)
			dbConn.Exec(query, integration1.ID, integration2.ID, integration3.ID)
		}()
	})

	t.Run("empty result for organization with no integrations", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		integrations, err := service.ListGitHubIntegrations(context.Background(), testUser.OrgID)

		require.NoError(t, err)
		assert.Empty(t, integrations)
	})

	t.Run("invalid organization ID returns error", func(t *testing.T) {
		integrations, err := service.ListGitHubIntegrations(context.Background(), "invalid-org-id")

		assert.Error(t, err)
		assert.Nil(t, integrations)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})

	t.Run("GitHub not configured returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create service with nil client to simulate not configured
		unconfiguredService := NewGitHubIntegrationsService(service.githubRepo, nil)

		integrations, err := unconfiguredService.ListGitHubIntegrations(context.Background(), testUser.OrgID)

		assert.Error(t, err)
		assert.Nil(t, integrations)
		assert.Equal(t, ErrGitHubNotConfigured, err)
	})
}

func TestGitHubIntegrationsService_GetGitHubIntegrationByID(t *testing.T) {
	service, usersRepo, cleanup := setupGitHubIntegrationsTest(t)
	defer cleanup()

	t.Run("successful retrieval of existing integration", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration
		mockClient := &github.MockGitHubClient{}
		mockClient.On("ExchangeCodeForAccessToken", context.Background(), "test-auth-code").
			Return("ghs_token_123", nil)

		testService := NewGitHubIntegrationsService(service.githubRepo, mockClient)
		createdIntegration, err := testService.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"test-auth-code",
			"12345678",
		)
		require.NoError(t, err)

		// Get the integration by ID
		maybeIntegration, err := service.GetGitHubIntegrationByID(
			context.Background(),
			testUser.OrgID,
			createdIntegration.ID,
		)

		require.NoError(t, err)
		assert.True(t, maybeIntegration.IsPresent())

		integration, exists := maybeIntegration.Get()
		assert.True(t, exists)
		assert.Equal(t, createdIntegration.ID, integration.ID)
		assert.Equal(t, createdIntegration.OrgID, integration.OrgID)
		assert.Equal(t, createdIntegration.GitHubInstallationID, integration.GitHubInstallationID)
		assert.Equal(t, createdIntegration.GitHubAccessToken, integration.GitHubAccessToken)

		// Verify mock expectations
		mockClient.AssertExpectations(t)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.github_integrations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, createdIntegration.ID)
		}()
	})

	t.Run("returns None for non-existent integration", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		maybeIntegration, err := service.GetGitHubIntegrationByID(
			context.Background(),
			testUser.OrgID,
			core.NewID("ghi"), // Non-existent integration ID
		)

		require.NoError(t, err)
		assert.False(t, maybeIntegration.IsPresent())
	})

	t.Run("invalid organization ID returns error", func(t *testing.T) {
		maybeIntegration, err := service.GetGitHubIntegrationByID(
			context.Background(),
			"invalid-org-id",
			core.NewID("ghi"),
		)

		assert.Error(t, err)
		assert.False(t, maybeIntegration.IsPresent())
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})

	t.Run("invalid integration ID returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		maybeIntegration, err := service.GetGitHubIntegrationByID(
			context.Background(),
			testUser.OrgID,
			"invalid-integration-id",
		)

		assert.Error(t, err)
		assert.False(t, maybeIntegration.IsPresent())
		assert.Contains(t, err.Error(), "integration ID must be a valid ULID")
	})

	t.Run("GitHub not configured returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create service with nil client to simulate not configured
		unconfiguredService := NewGitHubIntegrationsService(service.githubRepo, nil)

		maybeIntegration, err := unconfiguredService.GetGitHubIntegrationByID(
			context.Background(),
			testUser.OrgID,
			core.NewID("ghi"),
		)

		assert.Error(t, err)
		assert.False(t, maybeIntegration.IsPresent())
		assert.Equal(t, ErrGitHubNotConfigured, err)
	})
}

func TestGitHubIntegrationsService_DeleteGitHubIntegration(t *testing.T) {
	service, usersRepo, cleanup := setupGitHubIntegrationsTest(t)
	defer cleanup()

	t.Run("successful deletion of integration", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration
		mockClient := &github.MockGitHubClient{}
		mockClient.On("ExchangeCodeForAccessToken", context.Background(), "test-auth-code").
			Return("ghs_token_123", nil)
		mockClient.On("UninstallApp", context.Background(), "12345678").
			Return(nil)

		testService := NewGitHubIntegrationsService(service.githubRepo, mockClient)
		integration, err := testService.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"test-auth-code",
			"12345678",
		)
		require.NoError(t, err)

		// Delete the integration
		err = testService.DeleteGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			integration.ID,
		)
		require.NoError(t, err)

		// Verify integration is deleted
		integrations, err := service.ListGitHubIntegrations(context.Background(), testUser.OrgID)
		require.NoError(t, err)
		assert.Empty(t, integrations)

		// Verify mock expectations
		mockClient.AssertExpectations(t)
	})

	t.Run("deletion continues even if uninstall fails", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration
		mockClient := &github.MockGitHubClient{}
		mockClient.On("ExchangeCodeForAccessToken", context.Background(), "test-auth-code").
			Return("ghs_token_123", nil)
		mockClient.On("UninstallApp", context.Background(), "12345678").
			Return(fmt.Errorf("app already uninstalled"))

		testService := NewGitHubIntegrationsService(service.githubRepo, mockClient)
		integration, err := testService.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"test-auth-code",
			"12345678",
		)
		require.NoError(t, err)

		// Delete the integration - should succeed despite uninstall error
		err = testService.DeleteGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			integration.ID,
		)
		require.NoError(t, err)

		// Verify integration is still deleted from database
		integrations, err := service.ListGitHubIntegrations(context.Background(), testUser.OrgID)
		require.NoError(t, err)
		assert.Empty(t, integrations)

		// Verify mock expectations
		mockClient.AssertExpectations(t)
	})

	t.Run("invalid organization ID returns error", func(t *testing.T) {
		err := service.DeleteGitHubIntegration(
			context.Background(),
			"invalid-org-id",
			core.NewID("ghi"),
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})

	t.Run("invalid integration ID returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		err := service.DeleteGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"invalid-integration-id",
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "integration ID must be a valid ULID")
	})

	t.Run("non-existent integration returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		err := service.DeleteGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			core.NewID("ghi"), // Non-existent integration
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub integration not found")
	})

	t.Run("GitHub not configured returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create service with nil client to simulate not configured
		unconfiguredService := NewGitHubIntegrationsService(service.githubRepo, nil)

		err := unconfiguredService.DeleteGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			core.NewID("ghi"),
		)

		assert.Error(t, err)
		assert.Equal(t, ErrGitHubNotConfigured, err)
	})
}

func TestGitHubIntegrationsService_ListAvailableRepositories(t *testing.T) {
	service, usersRepo, cleanup := setupGitHubIntegrationsTest(t)
	defer cleanup()

	t.Run("successful retrieval of available repositories", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration first
		mockClient := &github.MockGitHubClient{}
		mockClient.On("ExchangeCodeForAccessToken", context.Background(), "test-auth-code").
			Return("ghs_token_123", nil)

		testRepositories := []models.GitHubRepository{
			{
				ID:       12345,
				Name:     "test-repo-1",
				FullName: "testorg/test-repo-1",
				Private:  false,
				HTMLURL:  "https://github.com/testorg/test-repo-1",
			},
			{
				ID:       67890,
				Name:     "test-repo-2",
				FullName: "testorg/test-repo-2",
				Private:  true,
				HTMLURL:  "https://github.com/testorg/test-repo-2",
			},
		}

		mockClient.On("ListInstalledRepositories", context.Background(), "12345678").
			Return(testRepositories, nil)

		testService := NewGitHubIntegrationsService(service.githubRepo, mockClient)

		// Create integration
		_, err := testService.CreateGitHubIntegration(
			context.Background(),
			testUser.OrgID,
			"test-auth-code",
			"12345678",
		)
		require.NoError(t, err)

		// List available repositories
		repositories, err := testService.ListAvailableRepositories(context.Background(), testUser.OrgID)

		require.NoError(t, err)
		assert.Len(t, repositories, 2)
		assert.Equal(t, testRepositories, repositories)

		// Verify mock expectations
		mockClient.AssertExpectations(t)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.github_integrations WHERE organization_id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, testUser.OrgID)
		}()
	})

	t.Run("empty result for organization with no integrations", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		repositories, err := service.ListAvailableRepositories(context.Background(), testUser.OrgID)

		require.NoError(t, err)
		assert.Empty(t, repositories)
	})

	t.Run("invalid organization ID returns error", func(t *testing.T) {
		repositories, err := service.ListAvailableRepositories(context.Background(), "invalid-org-id")

		assert.Error(t, err)
		assert.Nil(t, repositories)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})

	t.Run("GitHub not configured returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create service with nil client to simulate not configured
		unconfiguredService := NewGitHubIntegrationsService(service.githubRepo, nil)

		repositories, err := unconfiguredService.ListAvailableRepositories(context.Background(), testUser.OrgID)

		assert.Error(t, err)
		assert.Nil(t, repositories)
		assert.Equal(t, ErrGitHubNotConfigured, err)
	})
}
