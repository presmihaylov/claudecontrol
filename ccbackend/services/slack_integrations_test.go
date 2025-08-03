package services

import (
	"context"
	"fmt"
	"testing"

	"ccbackend/clients"
	"ccbackend/db"
	"ccbackend/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSlackIntegrationsTest(t *testing.T) (*SlackIntegrationsService, *db.PostgresUsersRepository, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)

	repo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)

	mockClient := clients.NewMockSlackClient()
	service := NewSlackIntegrationsService(repo, mockClient, "test-client-id", "test-client-secret")

	cleanup := func() {
		dbConn.Close()
	}

	return service, usersRepo, cleanup
}

func TestSlackIntegrationsService_CreateSlackIntegration(t *testing.T) {
	service, usersRepo, cleanup := setupSlackIntegrationsTest(t)
	defer cleanup()

	t.Run("successful integration creation", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Use unique team ID to avoid constraint violations
		teamID := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID,
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})

		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", testUser.ID)

		require.NoError(t, err)
		assert.NotNil(t, integration)
		assert.NotEqual(t, uuid.Nil, integration.ID)
		assert.Equal(t, teamID, integration.SlackTeamID)
		assert.Equal(t, "Test Team", integration.SlackTeamName)
		assert.Equal(t, "xoxb-test-token-123", integration.SlackAuthToken)
		assert.Equal(t, testUser.ID, integration.UserID)
		assert.NotZero(t, integration.CreatedAt)
		assert.NotZero(t, integration.UpdatedAt)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.slack_integrations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, integration.ID)
		}()
	})

	t.Run("empty auth code returns error", func(t *testing.T) {
		mockClient := clients.NewMockSlackClient()
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		// Use a real user ID even though this test should fail before DB operations
		testUser := testutils.CreateTestUser(t, usersRepo)

		integration, err := testService.CreateSlackIntegration("", "http://localhost:3000/callback", testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "slack auth code cannot be empty")
	})

	t.Run("nil user ID returns error", func(t *testing.T) {
		mockClient := clients.NewMockSlackClient()
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", uuid.Nil)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "user ID cannot be nil")
	})

	t.Run("slack OAuth error is propagated", func(t *testing.T) {
		mockClient := clients.NewMockSlackClient().WithOAuthV2Error(fmt.Errorf("invalid authorization code"))
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		testUser := testutils.CreateTestUser(t, usersRepo)

		integration, err := testService.CreateSlackIntegration("invalid-code", "http://localhost:3000/callback", testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "failed to exchange OAuth code with Slack")
		assert.Contains(t, err.Error(), "invalid authorization code")
	})

	t.Run("missing team ID in OAuth response returns error", func(t *testing.T) {
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      "", // Empty team ID
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		testUser := testutils.CreateTestUser(t, usersRepo)

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "team ID not found in Slack OAuth response")
	})

	t.Run("missing team name in OAuth response returns error", func(t *testing.T) {
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      "T123456789",
			TeamName:    "", // Empty team name
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		testUser := testutils.CreateTestUser(t, usersRepo)

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "team name not found in Slack OAuth response")
	})

	t.Run("missing access token in OAuth response returns error", func(t *testing.T) {
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      "T123456789",
			TeamName:    "Test Team",
			AccessToken: "", // Empty access token
		})
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		testUser := testutils.CreateTestUser(t, usersRepo)

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "bot access token not found in Slack OAuth response")
	})

	t.Run("database error is propagated", func(t *testing.T) {
		cfg, _ := testutils.LoadTestConfig()
		dbConn, _ := db.NewConnection(cfg.DatabaseURL)
		defer dbConn.Close()

		// Use invalid schema to trigger database error
		invalidRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, "nonexistent_schema")
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      "T123456789",
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(invalidRepo, mockClient, "test-client-id", "test-client-secret")

		testUser := testutils.CreateTestUser(t, usersRepo)

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "failed to create slack integration in database")
	})
}

func TestSlackIntegrationsService_GetSlackIntegrationsByUserID(t *testing.T) {
	service, usersRepo, cleanup := setupSlackIntegrationsTest(t)
	defer cleanup()

	t.Run("successful retrieval of user integrations", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create multiple integrations for the user with unique team IDs
		teamID1 := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID1,
			TeamName:    "Test Team 1",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		integration1, err := testService.CreateSlackIntegration("test-auth-code-1", "http://localhost:3000/callback", testUser.ID)
		require.NoError(t, err)

		teamID2 := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient2 := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID2,
			TeamName:    "Test Team 2",
			AccessToken: "xoxb-test-token-456",
		})
		testService2 := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient2, "test-client-id", "test-client-secret")

		integration2, err := testService2.CreateSlackIntegration("test-auth-code-2", "http://localhost:3000/callback", testUser.ID)
		require.NoError(t, err)

		// Create integration for different user to ensure isolation
		otherUser := testutils.CreateTestUser(t, usersRepo)
		teamID3 := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient3 := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID3,
			TeamName:    "Test Team 3",
			AccessToken: "xoxb-test-token-789",
		})
		testService3 := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient3, "test-client-id", "test-client-secret")
		integration3, err := testService3.CreateSlackIntegration("test-auth-code-3", "http://localhost:3000/callback", otherUser.ID)
		require.NoError(t, err)

		// Get integrations for the first user
		integrations, err := service.GetSlackIntegrationsByUserID(testUser.ID)

		require.NoError(t, err)
		assert.Len(t, integrations, 2)

		// Check that we got the right integrations (should be ordered by created_at DESC)
		foundIDs := make(map[uuid.UUID]bool)
		for _, integration := range integrations {
			foundIDs[integration.ID] = true
			assert.Equal(t, testUser.ID, integration.UserID)
		}
		assert.True(t, foundIDs[integration1.ID])
		assert.True(t, foundIDs[integration2.ID])
		assert.False(t, foundIDs[integration3.ID]) // Should not include other user's integration

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.slack_integrations WHERE id IN ($1, $2, $3)", cfg.DatabaseSchema)
			dbConn.Exec(query, integration1.ID, integration2.ID, integration3.ID)
		}()
	})

	t.Run("empty result for user with no integrations", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		integrations, err := service.GetSlackIntegrationsByUserID(testUser.ID)

		require.NoError(t, err)
		assert.Empty(t, integrations)
	})

	t.Run("nil user ID returns error", func(t *testing.T) {
		integrations, err := service.GetSlackIntegrationsByUserID(uuid.Nil)

		assert.Error(t, err)
		assert.Nil(t, integrations)
		assert.Contains(t, err.Error(), "user ID cannot be nil")
	})
}

func TestSlackIntegrationsService_DeleteSlackIntegration(t *testing.T) {
	service, usersRepo, cleanup := setupSlackIntegrationsTest(t)
	defer cleanup()

	t.Run("successful deletion of user's integration", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration with unique team ID
		teamID := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID,
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", testUser.ID)
		require.NoError(t, err)

		// Create context with user
		ctx := testutils.CreateTestContext(testUser)

		// Delete the integration
		err = service.DeleteSlackIntegration(ctx, integration.ID)
		require.NoError(t, err)

		// Verify integration is deleted
		integrations, err := service.GetSlackIntegrationsByUserID(testUser.ID)
		require.NoError(t, err)
		assert.Empty(t, integrations)
	})

	t.Run("cannot delete other user's integration", func(t *testing.T) {
		// Create two users
		user1 := testutils.CreateTestUser(t, usersRepo)
		user2 := testutils.CreateTestUser(t, usersRepo)

		// Create integration for user1 with unique team ID
		teamID := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID,
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", user1.ID)
		require.NoError(t, err)

		// Try to delete using user2's context
		ctx := testutils.CreateTestContext(user2)

		err = service.DeleteSlackIntegration(ctx, integration.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found or does not belong to user")

		// Verify integration still exists
		integrations, err := service.GetSlackIntegrationsByUserID(user1.ID)
		require.NoError(t, err)
		assert.Len(t, integrations, 1)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.slack_integrations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, integration.ID)
		}()
	})

	t.Run("nil integration ID returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)
		ctx := testutils.CreateTestContext(testUser)

		err := service.DeleteSlackIntegration(ctx, uuid.Nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "integration ID cannot be nil")
	})

	t.Run("missing user in context returns error", func(t *testing.T) {
		ctx := context.Background() // No user in context

		err := service.DeleteSlackIntegration(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found in context")
	})

	t.Run("non-existent integration returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)
		ctx := testutils.CreateTestContext(testUser)

		err := service.DeleteSlackIntegration(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSlackIntegrationsService_GenerateCCAgentSecretKey(t *testing.T) {
	service, usersRepo, cleanup := setupSlackIntegrationsTest(t)
	defer cleanup()

	t.Run("successful secret key generation", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration with unique team ID
		teamID := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID,
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", testUser.ID)
		require.NoError(t, err)

		// Create context with user
		ctx := testutils.CreateTestContext(testUser)

		// Generate secret key
		secretKey, err := service.GenerateCCAgentSecretKey(ctx, integration.ID)

		require.NoError(t, err)
		assert.NotEmpty(t, secretKey)
		assert.Greater(t, len(secretKey), 40) // Base64 encoded 32 bytes should be longer than 40 chars

		// Verify the integration was updated by fetching it again
		integrations, err := service.GetSlackIntegrationsByUserID(testUser.ID)
		require.NoError(t, err)
		require.Len(t, integrations, 1)

		updatedIntegration := integrations[0]
		assert.NotNil(t, updatedIntegration.CCAgentSecretKey)
		assert.Equal(t, secretKey, *updatedIntegration.CCAgentSecretKey)
		assert.NotNil(t, updatedIntegration.CCAgentSecretKeyGeneratedAt)
		assert.True(t, updatedIntegration.CCAgentSecretKeyGeneratedAt.After(integration.CreatedAt))

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.slack_integrations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, integration.ID)
		}()
	})

	t.Run("regenerating secret key updates existing key", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration with unique team ID
		teamID := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID,
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", testUser.ID)
		require.NoError(t, err)

		// Create context with user
		ctx := testutils.CreateTestContext(testUser)

		// Generate first secret key
		firstSecretKey, err := service.GenerateCCAgentSecretKey(ctx, integration.ID)
		require.NoError(t, err)

		// Get the first timestamp
		integrations, err := service.GetSlackIntegrationsByUserID(testUser.ID)
		require.NoError(t, err)
		require.Len(t, integrations, 1)
		firstTimestamp := *integrations[0].CCAgentSecretKeyGeneratedAt

		// Generate second secret key
		secondSecretKey, err := service.GenerateCCAgentSecretKey(ctx, integration.ID)
		require.NoError(t, err)

		// Keys should be different
		assert.NotEqual(t, firstSecretKey, secondSecretKey)

		// Verify the integration was updated
		integrations, err = service.GetSlackIntegrationsByUserID(testUser.ID)
		require.NoError(t, err)
		require.Len(t, integrations, 1)

		updatedIntegration := integrations[0]
		assert.NotNil(t, updatedIntegration.CCAgentSecretKey)
		assert.Equal(t, secondSecretKey, *updatedIntegration.CCAgentSecretKey)
		assert.NotNil(t, updatedIntegration.CCAgentSecretKeyGeneratedAt)
		assert.True(t, updatedIntegration.CCAgentSecretKeyGeneratedAt.After(firstTimestamp))

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.slack_integrations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, integration.ID)
		}()
	})

	t.Run("cannot generate secret key for other user's integration", func(t *testing.T) {
		// Create two users
		user1 := testutils.CreateTestUser(t, usersRepo)
		user2 := testutils.CreateTestUser(t, usersRepo)

		// Create integration for user1 with unique team ID
		teamID := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID,
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient, "test-client-id", "test-client-secret")

		integration, err := testService.CreateSlackIntegration("test-auth-code", "http://localhost:3000/callback", user1.ID)
		require.NoError(t, err)

		// Try to generate secret key using user2's context
		ctx := testutils.CreateTestContext(user2)

		secretKey, err := service.GenerateCCAgentSecretKey(ctx, integration.ID)
		assert.Error(t, err)
		assert.Empty(t, secretKey)
		assert.Contains(t, err.Error(), "not found or does not belong to user")

		// Verify integration still has no secret key
		integrations, err := service.GetSlackIntegrationsByUserID(user1.ID)
		require.NoError(t, err)
		require.Len(t, integrations, 1)
		assert.Nil(t, integrations[0].CCAgentSecretKey)
		assert.Nil(t, integrations[0].CCAgentSecretKeyGeneratedAt)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.slack_integrations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, integration.ID)
		}()
	})

	t.Run("nil integration ID returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)
		ctx := testutils.CreateTestContext(testUser)

		secretKey, err := service.GenerateCCAgentSecretKey(ctx, uuid.Nil)
		assert.Error(t, err)
		assert.Empty(t, secretKey)
		assert.Contains(t, err.Error(), "integration ID cannot be nil")
	})

	t.Run("missing user in context returns error", func(t *testing.T) {
		ctx := context.Background() // No user in context

		secretKey, err := service.GenerateCCAgentSecretKey(ctx, uuid.New())
		assert.Error(t, err)
		assert.Empty(t, secretKey)
		assert.Contains(t, err.Error(), "user not found in context")
	})

	t.Run("non-existent integration returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)
		ctx := testutils.CreateTestContext(testUser)

		secretKey, err := service.GenerateCCAgentSecretKey(ctx, uuid.New())
		assert.Error(t, err)
		assert.Empty(t, secretKey)
		assert.Contains(t, err.Error(), "not found or does not belong to user")
	})

	t.Run("generated keys are unique", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create two integrations with unique team IDs
		teamID1 := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient1 := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID1,
			TeamName:    "Test Team 1",
			AccessToken: "xoxb-test-token-123",
		})
		testService1 := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient1, "test-client-id", "test-client-secret")

		integration1, err := testService1.CreateSlackIntegration("test-auth-code-1", "http://localhost:3000/callback", testUser.ID)
		require.NoError(t, err)

		teamID2 := fmt.Sprintf("T%d", uuid.New().ID())
		mockClient2 := clients.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID2,
			TeamName:    "Test Team 2",
			AccessToken: "xoxb-test-token-456",
		})
		testService2 := NewSlackIntegrationsService(service.slackIntegrationsRepo, mockClient2, "test-client-id", "test-client-secret")

		integration2, err := testService2.CreateSlackIntegration("test-auth-code-2", "http://localhost:3000/callback", testUser.ID)
		require.NoError(t, err)

		// Create context with user
		ctx := testutils.CreateTestContext(testUser)

		// Generate secret keys for both integrations
		secretKey1, err := service.GenerateCCAgentSecretKey(ctx, integration1.ID)
		require.NoError(t, err)

		secretKey2, err := service.GenerateCCAgentSecretKey(ctx, integration2.ID)
		require.NoError(t, err)

		// Keys should be different
		assert.NotEqual(t, secretKey1, secretKey2)
		assert.NotEmpty(t, secretKey1)
		assert.NotEmpty(t, secretKey2)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.slack_integrations WHERE id IN ($1, $2)", cfg.DatabaseSchema)
			dbConn.Exec(query, integration1.ID, integration2.ID)
		}()
	})
}
