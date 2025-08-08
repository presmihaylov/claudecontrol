package slackintegrations

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/clients"
	"ccbackend/clients/slack"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/testutils"
)

func setupSlackIntegrationsTest(t *testing.T) (*SlackIntegrationsService, *db.PostgresUsersRepository, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)

	repo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)

	mockClient := slack.NewMockSlackClient()
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
		teamID := fmt.Sprintf("T%s", core.NewID("t"))
		mockClient := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID,
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})

		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		ctx := testutils.CreateTestContextWithUser(testUser)
		integration, err := testService.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"test-auth-code",
			"http://localhost:3000/callback",
		)

		require.NoError(t, err)
		assert.NotNil(t, integration)
		assert.NotEqual(t, "", integration.ID)
		assert.Equal(t, teamID, integration.SlackTeamID)
		assert.Equal(t, "Test Team", integration.SlackTeamName)
		assert.Equal(t, "xoxb-test-token-123", integration.SlackAuthToken)
		assert.Equal(t, testUser.OrganizationID, integration.OrganizationID)
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
		mockClient := slack.NewMockSlackClient()
		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		// Use a real user ID even though this test should fail before DB operations
		testUser := testutils.CreateTestUser(t, usersRepo)

		ctx := testutils.CreateTestContextWithUser(testUser)
		integration, err := testService.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"",
			"http://localhost:3000/callback",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "slack auth code cannot be empty")
	})

	t.Run("invalid organization ID returns error", func(t *testing.T) {
		mockClient := slack.NewMockSlackClient()
		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		integration, err := testService.CreateSlackIntegration(
			context.Background(),
			"invalid-org-id", // Invalid organizationID
			"test-auth-code",
			"http://localhost:3000/callback",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})

	t.Run("slack OAuth error is propagated", func(t *testing.T) {
		mockClient := slack.NewMockSlackClient().WithOAuthV2Error(fmt.Errorf("invalid authorization code"))
		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		testUser := testutils.CreateTestUser(t, usersRepo)

		ctx := testutils.CreateTestContextWithUser(testUser)
		integration, err := testService.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"invalid-code",
			"http://localhost:3000/callback",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "failed to exchange OAuth code with Slack")
		assert.Contains(t, err.Error(), "invalid authorization code")
	})

	t.Run("missing team ID in OAuth response returns error", func(t *testing.T) {
		mockClient := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      "", // Empty team ID
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		testUser := testutils.CreateTestUser(t, usersRepo)

		ctx := testutils.CreateTestContextWithUser(testUser)
		integration, err := testService.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"test-auth-code",
			"http://localhost:3000/callback",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "team ID not found in Slack OAuth response")
	})

	t.Run("missing team name in OAuth response returns error", func(t *testing.T) {
		mockClient := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      "T123456789",
			TeamName:    "", // Empty team name
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		testUser := testutils.CreateTestUser(t, usersRepo)

		ctx := testutils.CreateTestContextWithUser(testUser)
		integration, err := testService.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"test-auth-code",
			"http://localhost:3000/callback",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "team name not found in Slack OAuth response")
	})

	t.Run("missing access token in OAuth response returns error", func(t *testing.T) {
		mockClient := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      "T123456789",
			TeamName:    "Test Team",
			AccessToken: "", // Empty access token
		})
		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		testUser := testutils.CreateTestUser(t, usersRepo)

		ctx := testutils.CreateTestContextWithUser(testUser)
		integration, err := testService.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"test-auth-code",
			"http://localhost:3000/callback",
		)

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
		mockClient := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      "T123456789",
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(invalidRepo, mockClient, "test-client-id", "test-client-secret")

		testUser := testutils.CreateTestUser(t, usersRepo)

		ctx := testutils.CreateTestContextWithUser(testUser)
		integration, err := testService.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"test-auth-code",
			"http://localhost:3000/callback",
		)

		assert.Error(t, err)
		assert.Nil(t, integration)
		assert.Contains(t, err.Error(), "failed to create slack integration in database")
	})
}

func TestSlackIntegrationsService_GetSlackIntegrationsByOrganizationID(t *testing.T) {
	service, usersRepo, cleanup := setupSlackIntegrationsTest(t)
	defer cleanup()

	t.Run("successful retrieval of user integrations", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create multiple integrations for the user with unique team IDs
		teamID1 := fmt.Sprintf("T%s", core.NewID("t"))
		mockClient := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID1,
			TeamName:    "Test Team 1",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		ctx := testutils.CreateTestContextWithUser(testUser)
		integration1, err := testService.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"test-auth-code-1",
			"http://localhost:3000/callback",
		)
		require.NoError(t, err)

		teamID2 := fmt.Sprintf("T%s", core.NewID("t"))
		mockClient2 := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID2,
			TeamName:    "Test Team 2",
			AccessToken: "xoxb-test-token-456",
		})
		testService2 := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient2,
			"test-client-id",
			"test-client-secret",
		)

		ctx = testutils.CreateTestContextWithUser(testUser)
		integration2, err := testService2.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"test-auth-code-2",
			"http://localhost:3000/callback",
		)
		require.NoError(t, err)

		// Create integration for different user to ensure isolation
		otherUser := testutils.CreateTestUser(t, usersRepo)
		teamID3 := fmt.Sprintf("T%s", core.NewID("t"))
		mockClient3 := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID3,
			TeamName:    "Test Team 3",
			AccessToken: "xoxb-test-token-789",
		})
		testService3 := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient3,
			"test-client-id",
			"test-client-secret",
		)
		ctx3 := testutils.CreateTestContextWithUser(otherUser)
		integration3, err := testService3.CreateSlackIntegration(
			ctx3,
			otherUser.OrganizationID,
			"test-auth-code-3",
			"http://localhost:3000/callback",
		)
		require.NoError(t, err)

		// Get integrations for the first user
		ctx = testutils.CreateTestContextWithUser(testUser)
		integrations, err := service.GetSlackIntegrationsByOrganizationID(ctx, testUser.OrganizationID)

		require.NoError(t, err)
		assert.Len(t, integrations, 2)

		// Check that we got the right integrations (should be ordered by created_at DESC)
		foundIDs := make(map[string]bool)
		for _, integration := range integrations {
			foundIDs[integration.ID] = true
			assert.Equal(t, testUser.OrganizationID, integration.OrganizationID)
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

		ctx := testutils.CreateTestContextWithUser(testUser)
		integrations, err := service.GetSlackIntegrationsByOrganizationID(ctx, testUser.OrganizationID)

		require.NoError(t, err)
		assert.Empty(t, integrations)
	})

	t.Run("invalid organization ID returns error", func(t *testing.T) {
		integrations, err := service.GetSlackIntegrationsByOrganizationID(context.Background(), "invalid-org-id")

		assert.Error(t, err)
		assert.Nil(t, integrations)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})
}

func TestSlackIntegrationsService_DeleteSlackIntegration(t *testing.T) {
	service, usersRepo, cleanup := setupSlackIntegrationsTest(t)
	defer cleanup()

	t.Run("successful deletion of user's integration", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)

		// Create an integration with unique team ID
		teamID := fmt.Sprintf("T%s", core.NewID("t"))
		mockClient := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID,
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		ctx := testutils.CreateTestContextWithUser(testUser)
		integration, err := testService.CreateSlackIntegration(
			ctx,
			testUser.OrganizationID,
			"test-auth-code",
			"http://localhost:3000/callback",
		)
		require.NoError(t, err)

		// Create context with user
		ctx = testutils.CreateTestContextWithUser(testUser)

		// Delete the integration
		err = service.DeleteSlackIntegration(ctx, testUser.OrganizationID, integration.ID)
		require.NoError(t, err)

		// Verify integration is deleted
		ctx = testutils.CreateTestContextWithUser(testUser)
		integrations, err := service.GetSlackIntegrationsByOrganizationID(ctx, testUser.OrganizationID)
		require.NoError(t, err)
		assert.Empty(t, integrations)
	})

	t.Run("cannot delete other user's integration", func(t *testing.T) {
		// Create two users
		user1 := testutils.CreateTestUser(t, usersRepo)
		user2 := testutils.CreateTestUser(t, usersRepo)

		// Create integration for user1 with unique team ID
		teamID := fmt.Sprintf("T%s", core.NewID("t"))
		mockClient := slack.NewMockSlackClient().WithOAuthV2Response(&clients.OAuthV2Response{
			TeamID:      teamID,
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		})
		testService := NewSlackIntegrationsService(
			service.slackIntegrationsRepo,
			mockClient,
			"test-client-id",
			"test-client-secret",
		)

		ctx := testutils.CreateTestContextWithUser(user1)
		integration, err := testService.CreateSlackIntegration(
			ctx,
			user1.OrganizationID,
			"test-auth-code",
			"http://localhost:3000/callback",
		)
		require.NoError(t, err)

		// Try to delete using user2's context
		ctx = testutils.CreateTestContextWithUser(user2)

		err = service.DeleteSlackIntegration(ctx, user2.OrganizationID, integration.ID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, core.ErrNotFound))

		// Verify integration still exists
		ctx = testutils.CreateTestContextWithUser(user1)
		integrations, err := service.GetSlackIntegrationsByOrganizationID(ctx, user1.OrganizationID)
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
		ctx := testutils.CreateTestContextWithUser(testUser)

		err := service.DeleteSlackIntegration(ctx, testUser.OrganizationID, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "integration ID must be a valid ULID")
	})

	t.Run("invalid organization ID returns error", func(t *testing.T) {
		ctx := context.Background() // No user in context

		err := service.DeleteSlackIntegration(ctx, "invalid-org-id", core.NewID("t"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})

	t.Run("non-existent integration returns error", func(t *testing.T) {
		testUser := testutils.CreateTestUser(t, usersRepo)
		ctx := testutils.CreateTestContextWithUser(testUser)

		err := service.DeleteSlackIntegration(ctx, testUser.OrganizationID, core.NewID("t"))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, core.ErrNotFound))
	})
}
