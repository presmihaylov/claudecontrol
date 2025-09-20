package connectedchannels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services/agents"
	"ccbackend/services/connectedchannels"
	"ccbackend/testutils"
)

func setupTestService(t *testing.T) (*connectedchannels.ConnectedChannelsService, *models.User, *agents.MockAgentsService, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	// Create repositories
	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)

	// Create test user (this also creates the organization)
	testUser := testutils.CreateTestUser(t, usersRepo)

	// Create mock agents service for controlled testing
	mockAgentsService := &agents.MockAgentsService{}
	connectedChannelsService := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

	cleanup := func() {
		// Clean up test data
		dbConn.Close()
	}

	return connectedChannelsService, testUser, mockAgentsService, cleanup
}

func TestConnectedChannelsService_UpsertSlackConnectedChannel(t *testing.T) {
	service, testUser, mockAgentsService, cleanup := setupTestService(t)
	defer cleanup()


	t.Run("Create new Slack channel with default repo URL", func(t *testing.T) {
		// Create test agent
		testAgent := &models.ActiveAgent{
			ID:             core.NewID("ag"),
			WSConnectionID: "test-conn-1",
			OrgID:          testUser.OrgID,
			CCAgentID:      "test-agent-1",
			RepoURL:        "https://github.com/test/repo.git",
		}

		// Mock agents service to return our test agent
		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), testUser.OrgID, []string{}).
			Return([]*models.ActiveAgent{testAgent}, nil).Once()

		teamID := "T1234567890"
		channelID := "C1234567890"

		channel, err := service.UpsertSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, testUser.OrgID, channel.OrgID)
		assert.Equal(t, teamID, channel.TeamID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.NotNil(t, channel.DefaultRepoURL)
		assert.Equal(t, testAgent.RepoURL, *channel.DefaultRepoURL)

		// Verify the channel can be retrieved
		retrievedChannel, err := service.GetSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)
		require.True(t, retrievedChannel.IsPresent())
		assert.Equal(t, channel.ID, retrievedChannel.MustGet().ID)

		mockAgentsService.AssertExpectations(t)
	})

	t.Run("Upsert same Slack channel twice", func(t *testing.T) {
		// Create test agent
		testAgent := &models.ActiveAgent{
			ID:             core.NewID("ag"),
			WSConnectionID: "test-conn-2",
			OrgID:          testUser.OrgID,
			CCAgentID:      "test-agent-2",
			RepoURL:        "https://github.com/duplicate/repo.git",
		}

		// Mock agents service to return our test agent
		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), testUser.OrgID, []string{}).
			Return([]*models.ActiveAgent{testAgent}, nil).Once()

		teamID := "T0987654321"
		channelID := "C0987654321"

		// First upsert
		_, err := service.UpsertSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)

		// Second upsert of the same channel
		secondChannel, err := service.UpsertSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)

		// Both should have the same basic properties
		assert.Equal(t, testUser.OrgID, secondChannel.OrgID)
		assert.Equal(t, teamID, secondChannel.TeamID)
		assert.Equal(t, channelID, secondChannel.ChannelID)

		// Verify the channel can be retrieved
		retrievedChannel, err := service.GetSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)
		require.True(t, retrievedChannel.IsPresent())
		assert.Equal(t, teamID, retrievedChannel.MustGet().TeamID)
		assert.Equal(t, channelID, retrievedChannel.MustGet().ChannelID)

		mockAgentsService.AssertExpectations(t)
	})

	t.Run("Empty team ID returns error", func(t *testing.T) {
		_, err := service.UpsertSlackConnectedChannel(context.Background(), testUser.OrgID, "", "C1234567890")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team ID cannot be empty")
	})

	t.Run("Empty channel ID returns error", func(t *testing.T) {
		_, err := service.UpsertSlackConnectedChannel(context.Background(), testUser.OrgID, "T1234567890", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel ID cannot be empty")
	})

	t.Run("No agents available returns empty repo URL", func(t *testing.T) {
		// Mock returns empty agents
		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), testUser.OrgID, []string{}).
			Return([]*models.ActiveAgent{}, nil).Once()

		teamID := "T2345678901"
		channelID := "C2345678901"

		channel, err := service.UpsertSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)

		assert.Nil(t, channel.DefaultRepoURL)
		mockAgentsService.AssertExpectations(t)

		// Verify the channel can be retrieved
		retrievedChannel, err := service.GetSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)
		require.True(t, retrievedChannel.IsPresent())
		assert.Nil(t, retrievedChannel.MustGet().DefaultRepoURL)
	})

	t.Run("Channel with null repo URL gets assigned repo URL when agents become available", func(t *testing.T) {
		teamID := "T3456789012"
		channelID := "C3456789012"

		// First call - no agents available
		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), testUser.OrgID, []string{}).
			Return([]*models.ActiveAgent{}, nil).Once()

		firstChannel, err := service.UpsertSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)
		assert.Nil(t, firstChannel.DefaultRepoURL)

		// Second call - agent becomes available
		testAgent := &models.ActiveAgent{
			ID:             core.NewID("ag"),
			WSConnectionID: "test-conn-3",
			OrgID:          testUser.OrgID,
			CCAgentID:      "test-agent-3",
			RepoURL:        "https://github.com/newly-available/repo.git",
		}

		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), testUser.OrgID, []string{}).
			Return([]*models.ActiveAgent{testAgent}, nil).Once()

		secondChannel, err := service.UpsertSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)

		// Should now have the repo URL assigned
		assert.NotNil(t, secondChannel.DefaultRepoURL)
		assert.Equal(t, testAgent.RepoURL, *secondChannel.DefaultRepoURL)

		// Verify via get function
		retrievedChannel, err := service.GetSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)
		require.True(t, retrievedChannel.IsPresent())
		assert.Equal(t, testAgent.RepoURL, *retrievedChannel.MustGet().DefaultRepoURL)

		mockAgentsService.AssertExpectations(t)
	})
}

func TestConnectedChannelsService_UpsertDiscordConnectedChannel(t *testing.T) {
	service, testUser, mockAgentsService, cleanup := setupTestService(t)
	defer cleanup()


	t.Run("Create new Discord channel with default repo URL", func(t *testing.T) {
		// Create test agent
		testAgent := &models.ActiveAgent{
			ID:             core.NewID("ag"),
			WSConnectionID: "test-conn-1",
			OrgID:          testUser.OrgID,
			CCAgentID:      "test-agent-1",
			RepoURL:        "https://github.com/test/repo.git",
		}

		// Mock agents service to return our test agent
		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), testUser.OrgID, []string{}).
			Return([]*models.ActiveAgent{testAgent}, nil).Once()

		guildID := "987654321098765432"
		channelID := "123456789012345678"

		channel, err := service.UpsertDiscordConnectedChannel(context.Background(), testUser.OrgID, guildID, channelID)
		require.NoError(t, err)

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, testUser.OrgID, channel.OrgID)
		assert.Equal(t, guildID, channel.GuildID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.NotNil(t, channel.DefaultRepoURL)
		assert.Equal(t, testAgent.RepoURL, *channel.DefaultRepoURL)

		// Note: No get function for Discord channels since we only kept Slack get for testing

		mockAgentsService.AssertExpectations(t)
	})

	t.Run("Empty guild ID returns error", func(t *testing.T) {
		_, err := service.UpsertDiscordConnectedChannel(context.Background(), testUser.OrgID, "", "123456789012345678")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "guild ID cannot be empty")
	})

	t.Run("Empty channel ID returns error", func(t *testing.T) {
		_, err := service.UpsertDiscordConnectedChannel(context.Background(), testUser.OrgID, "987654321098765432", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel ID cannot be empty")
	})
}



