package anthropic_integrations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/clients/anthropic"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/testutils"
)

func setupAnthropicIntegrationsTest(t *testing.T) (*AnthropicIntegrationsService, *db.PostgresUsersRepository, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)

	repo := db.NewPostgresAnthropicIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)

	mockClient := anthropic.NewMockAnthropicClient()
	service := NewAnthropicIntegrationsService(repo, mockClient)

	cleanup := func() {
		dbConn.Close()
	}

	return service, usersRepo, cleanup
}

func TestAnthropicIntegrationsService_CreateAnthropicIntegration(t *testing.T) {
	service, usersRepo, cleanup := setupAnthropicIntegrationsTest(t)
	defer cleanup()

	t.Run("successful integration creation with API key", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		apiKey := "test-api-key-123"
		integration, err := service.CreateAnthropicIntegration(
			context.Background(),
			testUser.OrgID,
			&apiKey,
			nil, // no OAuth token
			nil, // no code verifier
		)

		require.NoError(t, err)
		assert.NotNil(t, integration)
		assert.NotEqual(t, "", integration.ID)
		assert.True(t, core.IsValidULID(integration.ID))
		assert.Equal(t, testUser.OrgID, integration.OrgID)
		assert.Equal(t, apiKey, *integration.AnthropicAPIKey)
		assert.Nil(t, integration.ClaudeCodeOAuthToken)
		assert.Nil(t, integration.ClaudeCodeOAuthRefreshToken)
		assert.Nil(t, integration.ClaudeCodeOAuthTokenExpiresAt)
		assert.NotZero(t, integration.CreatedAt)
		assert.NotZero(t, integration.UpdatedAt)
	})

	t.Run("successful integration creation with OAuth token", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Setup mock client with token exchange response
		testTokens := anthropic.CreateTestTokens()
		mockClient := anthropic.NewMockAnthropicClient().WithTokenExchangeResponse(testTokens)
		testService := NewAnthropicIntegrationsService(
			service.anthropicRepo,
			mockClient,
		)

		oauthToken := "test-oauth-code-123#test-state-456"
		codeVerifier := "test-code-verifier-789"
		integration, err := testService.CreateAnthropicIntegration(
			context.Background(),
			testUser.OrgID,
			nil, // no API key
			&oauthToken,
			&codeVerifier,
		)

		require.NoError(t, err)
		assert.NotNil(t, integration)
		assert.NotEqual(t, "", integration.ID)
		assert.True(t, core.IsValidULID(integration.ID))
		assert.Equal(t, testUser.OrgID, integration.OrgID)
		assert.Nil(t, integration.AnthropicAPIKey)
		assert.Equal(t, testTokens.AccessToken, *integration.ClaudeCodeOAuthToken)
		assert.Equal(t, testTokens.RefreshToken, *integration.ClaudeCodeOAuthRefreshToken)
		assert.NotNil(t, integration.ClaudeCodeOAuthTokenExpiresAt)
		assert.NotZero(t, integration.CreatedAt)
		assert.NotZero(t, integration.UpdatedAt)

		// Verify mock was called
		mockClient.AssertExpectations(t)
	})
}

func TestAnthropicIntegrationsService_ListAnthropicIntegrations(t *testing.T) {
	service, usersRepo, cleanup := setupAnthropicIntegrationsTest(t)
	defer cleanup()

	t.Run("successful retrieval of multiple integrations", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create first integration with API key
		apiKey1 := "test-api-key-123"
		integration1, err := service.CreateAnthropicIntegration(
			context.Background(),
			testUser.OrgID,
			&apiKey1,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Create second integration with OAuth
		testTokens := anthropic.CreateTestTokens()
		mockClient := anthropic.NewMockAnthropicClient().WithTokenExchangeResponse(testTokens)
		testService := NewAnthropicIntegrationsService(
			service.anthropicRepo,
			mockClient,
		)

		oauthToken := "test-oauth-code-456#test-state-789"
		codeVerifier := "test-code-verifier-abc"
		integration2, err := testService.CreateAnthropicIntegration(
			context.Background(),
			testUser.OrgID,
			nil,
			&oauthToken,
			&codeVerifier,
		)
		require.NoError(t, err)

		// Create integration for different user to ensure isolation
		otherUser := testutils.CreateTestUser(t, usersRepo)
		apiKey3 := "test-api-key-other"
		integration3, err := service.CreateAnthropicIntegration(
			context.Background(),
			otherUser.OrgID,
			&apiKey3,
			nil,
			nil,
		)
		require.NoError(t, err)

		// List integrations for the first user
		integrations, err := service.ListAnthropicIntegrations(context.Background(), testUser.OrgID)

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
		assert.False(t, foundIDs[integration3.ID]) // Should not include other user's integration
	})

	t.Run("empty result for user with no integrations", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		integrations, err := service.ListAnthropicIntegrations(context.Background(), testUser.OrgID)

		require.NoError(t, err)
		assert.Empty(t, integrations)
	})

	t.Run("organization isolation verified", func(t *testing.T) {
		// Create two users in different orgs
		user1 := testutils.CreateTestUser(t, usersRepo)
		user2 := testutils.CreateTestUser(t, usersRepo)

		// Create integration for user1
		apiKey1 := "test-api-key-user1"
		integration1, err := service.CreateAnthropicIntegration(
			context.Background(),
			user1.OrgID,
			&apiKey1,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Create integration for user2
		apiKey2 := "test-api-key-user2"
		integration2, err := service.CreateAnthropicIntegration(
			context.Background(),
			user2.OrgID,
			&apiKey2,
			nil,
			nil,
		)
		require.NoError(t, err)

		// List integrations for user1 - should only see their own
		integrations1, err := service.ListAnthropicIntegrations(context.Background(), user1.OrgID)
		require.NoError(t, err)
		assert.Len(t, integrations1, 1)
		assert.Equal(t, integration1.ID, integrations1[0].ID)

		// List integrations for user2 - should only see their own
		integrations2, err := service.ListAnthropicIntegrations(context.Background(), user2.OrgID)
		require.NoError(t, err)
		assert.Len(t, integrations2, 1)
		assert.Equal(t, integration2.ID, integrations2[0].ID)
	})
}

func TestAnthropicIntegrationsService_GetAnthropicIntegrationByID(t *testing.T) {
	service, usersRepo, cleanup := setupAnthropicIntegrationsTest(t)
	defer cleanup()

	t.Run("successful retrieval of existing integration", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration
		apiKey := "test-api-key-123"
		createdIntegration, err := service.CreateAnthropicIntegration(
			context.Background(),
			testUser.OrgID,
			&apiKey,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Get the integration by ID
		maybeIntegration, err := service.GetAnthropicIntegrationByID(
			context.Background(),
			testUser.OrgID,
			createdIntegration.ID,
		)

		require.NoError(t, err)
		assert.True(t, maybeIntegration.IsPresent())

		integration := maybeIntegration.MustGet()
		assert.Equal(t, createdIntegration.ID, integration.ID)
		assert.Equal(t, testUser.OrgID, integration.OrgID)
		assert.Equal(t, apiKey, *integration.AnthropicAPIKey)
	})

	t.Run("returns None for non-existent integration", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		nonExistentID := core.NewID("ai")
		maybeIntegration, err := service.GetAnthropicIntegrationByID(
			context.Background(),
			testUser.OrgID,
			nonExistentID,
		)

		require.NoError(t, err)
		assert.False(t, maybeIntegration.IsPresent())
	})

	t.Run("organization scoping prevents access to other org's integration", func(t *testing.T) {
		user1 := testutils.CreateTestUser(t, usersRepo)
		user2 := testutils.CreateTestUser(t, usersRepo)

		// Create integration for user1
		apiKey := "test-api-key-123"
		integration, err := service.CreateAnthropicIntegration(
			context.Background(),
			user1.OrgID,
			&apiKey,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Try to access with user2's orgID - should not find it
		maybeIntegration, err := service.GetAnthropicIntegrationByID(
			context.Background(),
			user2.OrgID,
			integration.ID,
		)

		require.NoError(t, err)
		assert.False(t, maybeIntegration.IsPresent())

		// Verify user1 can still access their own integration
		maybeIntegration, err = service.GetAnthropicIntegrationByID(
			context.Background(),
			user1.OrgID,
			integration.ID,
		)
		require.NoError(t, err)
		assert.True(t, maybeIntegration.IsPresent())
	})
}

func TestAnthropicIntegrationsService_DeleteAnthropicIntegration(t *testing.T) {
	service, usersRepo, cleanup := setupAnthropicIntegrationsTest(t)
	defer cleanup()

	t.Run("successful deletion of user's integration", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration
		apiKey := "test-api-key-123"
		integration, err := service.CreateAnthropicIntegration(
			context.Background(),
			testUser.OrgID,
			&apiKey,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Delete the integration
		err = service.DeleteAnthropicIntegration(context.Background(), testUser.OrgID, integration.ID)
		require.NoError(t, err)

		// Verify integration is deleted
		integrations, err := service.ListAnthropicIntegrations(context.Background(), testUser.OrgID)
		require.NoError(t, err)
		assert.Empty(t, integrations)

		// Verify GetByID also returns None
		maybeIntegration, err := service.GetAnthropicIntegrationByID(
			context.Background(),
			testUser.OrgID,
			integration.ID,
		)
		require.NoError(t, err)
		assert.False(t, maybeIntegration.IsPresent())
	})

	t.Run("organization isolation prevents deletion of other org's integration", func(t *testing.T) {
		user1 := testutils.CreateTestUser(t, usersRepo)
		user2 := testutils.CreateTestUser(t, usersRepo)

		// Create integration for user1
		apiKey := "test-api-key-123"
		integration, err := service.CreateAnthropicIntegration(
			context.Background(),
			user1.OrgID,
			&apiKey,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Try to delete using user2's orgID - should return error since integration doesn't exist for user2's org
		err = service.DeleteAnthropicIntegration(context.Background(), user2.OrgID, integration.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "anthropic integration not found")

		// Verify integration still exists for user1
		integrations, err := service.ListAnthropicIntegrations(context.Background(), user1.OrgID)
		require.NoError(t, err)
		assert.Len(t, integrations, 1)
		assert.Equal(t, integration.ID, integrations[0].ID)
	})
}

func TestAnthropicIntegrationsService_RefreshTokens(t *testing.T) {
	service, usersRepo, cleanup := setupAnthropicIntegrationsTest(t)
	defer cleanup()

	t.Run("successful token refresh", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create OAuth integration first
		testTokens := anthropic.CreateTestTokens()
		mockClient := anthropic.NewMockAnthropicClient().WithTokenExchangeResponse(testTokens)
		testService := NewAnthropicIntegrationsService(
			service.anthropicRepo,
			mockClient,
		)

		oauthToken := "test-oauth-code-123#test-state-456"
		codeVerifier := "test-code-verifier-789"
		integration, err := testService.CreateAnthropicIntegration(
			context.Background(),
			testUser.OrgID,
			nil,
			&oauthToken,
			&codeVerifier,
		)
		require.NoError(t, err)

		// Setup new mock for refresh operation
		refreshedTokens := anthropic.CreateRefreshedTestTokens()
		refreshMockClient := anthropic.NewMockAnthropicClient().WithRefreshTokenResponse(refreshedTokens)
		refreshService := NewAnthropicIntegrationsService(
			service.anthropicRepo,
			refreshMockClient,
		)

		// Refresh the tokens
		updatedIntegration, err := refreshService.RefreshTokens(
			context.Background(),
			testUser.OrgID,
			integration.ID,
		)

		require.NoError(t, err)
		assert.NotNil(t, updatedIntegration)
		assert.Equal(t, integration.ID, updatedIntegration.ID)
		assert.Equal(t, testUser.OrgID, updatedIntegration.OrgID)
		assert.Equal(t, refreshedTokens.AccessToken, *updatedIntegration.ClaudeCodeOAuthToken)
		assert.Equal(t, refreshedTokens.RefreshToken, *updatedIntegration.ClaudeCodeOAuthRefreshToken)
		assert.NotNil(t, updatedIntegration.ClaudeCodeOAuthTokenExpiresAt)

		// Verify mock was called
		refreshMockClient.AssertExpectations(t)
	})

	t.Run("integration updated in database after refresh", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create OAuth integration first
		testTokens := anthropic.CreateTestTokens()
		mockClient := anthropic.NewMockAnthropicClient().WithTokenExchangeResponse(testTokens)
		testService := NewAnthropicIntegrationsService(
			service.anthropicRepo,
			mockClient,
		)

		oauthToken := "test-oauth-code-123#test-state-456"
		codeVerifier := "test-code-verifier-789"
		integration, err := testService.CreateAnthropicIntegration(
			context.Background(),
			testUser.OrgID,
			nil,
			&oauthToken,
			&codeVerifier,
		)
		require.NoError(t, err)

		// Setup new mock for refresh operation
		refreshedTokens := anthropic.CreateRefreshedTestTokens()
		refreshMockClient := anthropic.NewMockAnthropicClient().WithRefreshTokenResponse(refreshedTokens)
		refreshService := NewAnthropicIntegrationsService(
			service.anthropicRepo,
			refreshMockClient,
		)

		// Refresh the tokens
		_, err = refreshService.RefreshTokens(
			context.Background(),
			testUser.OrgID,
			integration.ID,
		)
		require.NoError(t, err)

		// Verify the integration was updated in the database
		maybeUpdatedIntegration, err := service.GetAnthropicIntegrationByID(
			context.Background(),
			testUser.OrgID,
			integration.ID,
		)
		require.NoError(t, err)
		assert.True(t, maybeUpdatedIntegration.IsPresent())

		updatedIntegration := maybeUpdatedIntegration.MustGet()
		assert.Equal(t, refreshedTokens.AccessToken, *updatedIntegration.ClaudeCodeOAuthToken)
		assert.Equal(t, refreshedTokens.RefreshToken, *updatedIntegration.ClaudeCodeOAuthRefreshToken)
		assert.NotNil(t, updatedIntegration.ClaudeCodeOAuthTokenExpiresAt)
	})
}
