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

	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)
	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, cfg.DatabaseSchema)

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
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, testUser.OrgID)

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, testUser.OrgID, channel.OrgID)
		assert.Equal(t, teamID, channel.TeamID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.NotNil(t, channel.DefaultRepoURL)
		assert.Equal(t, testAgent.RepoURL, *channel.DefaultRepoURL)

		mockAgentsService.AssertExpectations(t)
	})

	t.Run("Update existing Slack channel preserves default repo URL", func(t *testing.T) {
		teamID := "T0987654321"
		channelID := "C0987654321"
		originalRepoURL := "https://github.com/original/repo.git"

		// Create channel with original repo URL
		originalChannel := &db.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            testUser.OrgID,
			SlackTeamID:      &teamID,
			SlackChannelID:   &channelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &originalRepoURL,
		}
		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), originalChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), originalChannel.ID, testUser.OrgID)

		// Upsert again should preserve the original repo URL (no agents service call expected)
		updatedChannel, err := service.UpsertSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)

		assert.Equal(t, originalChannel.ID, updatedChannel.ID)
		assert.Equal(t, originalRepoURL, *updatedChannel.DefaultRepoURL)
		// Should not call GetConnectedActiveAgents for existing channel
		mockAgentsService.AssertNotCalled(t, "GetConnectedActiveAgents")
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
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, testUser.OrgID)

		assert.Nil(t, channel.DefaultRepoURL)
		mockAgentsService.AssertExpectations(t)
	})
}

func TestConnectedChannelsService_UpsertDiscordConnectedChannel(t *testing.T) {
	service, testUser, mockAgentsService, cleanup := setupTestService(t)
	defer cleanup()

	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)
	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, cfg.DatabaseSchema)

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
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, testUser.OrgID)

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, testUser.OrgID, channel.OrgID)
		assert.Equal(t, guildID, channel.GuildID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.NotNil(t, channel.DefaultRepoURL)
		assert.Equal(t, testAgent.RepoURL, *channel.DefaultRepoURL)

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



