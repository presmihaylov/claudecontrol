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
		originalChannel := &models.DatabaseConnectedChannel{
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

func TestConnectedChannelsService_GetSlackConnectedChannel(t *testing.T) {
	service, testUser, _, cleanup := setupTestService(t)
	defer cleanup()

	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)
	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, cfg.DatabaseSchema)

	t.Run("Get existing Slack channel", func(t *testing.T) {
		teamID := "T1234567890"
		channelID := "C1234567890"
		repoURL := "https://github.com/test/repo.git"

		// Create test channel
		testChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            testUser.OrgID,
			SlackTeamID:      &teamID,
			SlackChannelID:   &channelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &repoURL,
		}
		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), testChannel.ID, testUser.OrgID)

		// Get the channel
		maybeChannel, err := service.GetSlackConnectedChannel(context.Background(), testUser.OrgID, teamID, channelID)
		require.NoError(t, err)
		require.True(t, maybeChannel.IsPresent())

		channel := maybeChannel.MustGet()
		assert.Equal(t, testChannel.ID, channel.ID)
		assert.Equal(t, teamID, channel.TeamID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Equal(t, repoURL, *channel.DefaultRepoURL)
	})

	t.Run("Get non-existent Slack channel", func(t *testing.T) {
		maybeChannel, err := service.GetSlackConnectedChannel(context.Background(), testUser.OrgID, "T9999999999", "C9999999999")
		require.NoError(t, err)
		assert.False(t, maybeChannel.IsPresent())
	})
}

func TestConnectedChannelsService_GetConnectedChannelsByOrganization(t *testing.T) {
	service, testUser, _, cleanup := setupTestService(t)
	defer cleanup()

	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)
	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, cfg.DatabaseSchema)

	t.Run("Get channels for organization", func(t *testing.T) {
		repoURL := "https://github.com/test/repo.git"

		// Create test channels
		slackTeamID := "T1234567890"
		slackChannelID := "C1234567890"
		slackChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            testUser.OrgID,
			SlackTeamID:      &slackTeamID,
			SlackChannelID:   &slackChannelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &repoURL,
		}

		discordGuildID := "987654321098765432"
		discordChannelID := "123456789012345678"
		discordChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            testUser.OrgID,
			SlackTeamID:      nil,
			SlackChannelID:   nil,
			DiscordGuildID:   &discordGuildID,
			DiscordChannelID: &discordChannelID,
			DefaultRepoURL:   &repoURL,
		}

		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), slackChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), slackChannel.ID, testUser.OrgID)

		err = connectedChannelsRepo.UpsertDiscordConnectedChannel(context.Background(), discordChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), discordChannel.ID, testUser.OrgID)

		// Get channels
		channels, err := service.GetConnectedChannelsByOrganization(context.Background(), testUser.OrgID)
		require.NoError(t, err)
		assert.Len(t, channels, 2)

		// Verify channels are returned
		foundSlack := false
		foundDiscord := false
		for _, ch := range channels {
			switch ch := ch.(type) {
			case *models.SlackConnectedChannel:
				if ch.ChannelID == slackChannelID && ch.TeamID == slackTeamID {
					foundSlack = true
				}
			case *models.DiscordConnectedChannel:
				if ch.ChannelID == discordChannelID && ch.GuildID == discordGuildID {
					foundDiscord = true
				}
			}
		}
		assert.True(t, foundSlack, "Slack channel should be found")
		assert.True(t, foundDiscord, "Discord channel should be found")
	})
}

func TestConnectedChannelsService_DeleteConnectedChannel(t *testing.T) {
	service, testUser, _, cleanup := setupTestService(t)
	defer cleanup()

	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)
	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, cfg.DatabaseSchema)

	t.Run("Delete existing channel", func(t *testing.T) {
		repoURL := "https://github.com/test/repo.git"
		teamID := "T1234567890"
		channelID := "C1234567890"
		testChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            testUser.OrgID,
			SlackTeamID:      &teamID,
			SlackChannelID:   &channelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &repoURL,
		}
		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)

		// Delete the channel
		err = service.DeleteConnectedChannel(context.Background(), testUser.OrgID, testChannel.ID)
		require.NoError(t, err)

		// Verify it's deleted
		maybeChannel, err := connectedChannelsRepo.GetConnectedChannelByID(context.Background(), testChannel.ID, testUser.OrgID)
		require.NoError(t, err)
		assert.False(t, maybeChannel.IsPresent())
	})

	t.Run("Delete non-existent channel returns error", func(t *testing.T) {
		err := service.DeleteConnectedChannel(context.Background(), testUser.OrgID, core.NewID("cc"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connected channel not found")
	})
}