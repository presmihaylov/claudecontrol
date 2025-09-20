package connectedchannels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services/connectedchannels"
	"ccbackend/testutils"
)

func TestConnectedChannelsService_UpsertSlackConnectedChannel(t *testing.T) {
	dbConn, err := testutils.SetupTestDB()
	require.NoError(t, err)
	defer dbConn.Close()

	// Setup repositories and service
	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, testutils.TestSchema)
	agentsRepo := db.NewPostgresAgentsRepository(dbConn, testutils.TestSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, testutils.TestSchema)

	// Create test organization and agent
	testOrg := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), testOrg)
	require.NoError(t, err)

	testAgent := &models.ActiveAgent{
		ID:             core.NewID("ag"),
		WSConnectionID: "test-conn-1",
		OrgID:          models.OrgID(testOrg.ID),
		CCAgentID:      "test-agent-1",
		RepoURL:        "https://github.com/test/repo.git",
	}
	err = agentsRepo.UpsertActiveAgent(context.Background(), testAgent)
	require.NoError(t, err)

	// Create mock agents service that returns our test agent
	mockAgentsService := &testutils.MockAgentsService{}
	mockAgentsService.On("GetConnectedActiveAgents", context.Background(), models.OrgID(testOrg.ID), []string{}).
		Return([]*models.ActiveAgent{testAgent}, nil)

	defer func() {
		// Cleanup
		agentsRepo.DeleteActiveAgent(context.Background(), testAgent.ID, models.OrgID(testOrg.ID))
	}()

	service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

	t.Run("Create new Slack channel with default repo URL", func(t *testing.T) {
		teamID := "T1234567890"
		channelID := "C1234567890"

		channel, err := service.UpsertSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), teamID, channelID)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, models.OrgID(testOrg.ID))

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, models.OrgID(testOrg.ID), channel.OrgID)
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
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      &teamID,
			SlackChannelID:   &channelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &originalRepoURL,
		}
		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), originalChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), originalChannel.ID, models.OrgID(testOrg.ID))

		// Upsert again should preserve the original repo URL
		updatedChannel, err := service.UpsertSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), teamID, channelID)
		require.NoError(t, err)

		assert.Equal(t, originalChannel.ID, updatedChannel.ID)
		assert.Equal(t, originalRepoURL, *updatedChannel.DefaultRepoURL)
		// Should not call GetConnectedActiveAgents for existing channel
		mockAgentsService.AssertNotCalled(t, "GetConnectedActiveAgents")
	})

	t.Run("Empty team ID returns error", func(t *testing.T) {
		_, err := service.UpsertSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "", "C1234567890")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team ID cannot be empty")
	})

	t.Run("Empty channel ID returns error", func(t *testing.T) {
		_, err := service.UpsertSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "T1234567890", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel ID cannot be empty")
	})
}

func TestConnectedChannelsService_UpsertDiscordConnectedChannel(t *testing.T) {
	dbConn, err := testutils.SetupTestDB()
	require.NoError(t, err)
	defer dbConn.Close()

	// Setup repositories and service
	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, testutils.TestSchema)
	agentsRepo := db.NewPostgresAgentsRepository(dbConn, testutils.TestSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, testutils.TestSchema)

	// Create test organization and agent
	testOrg := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), testOrg)
	require.NoError(t, err)

	testAgent := &models.ActiveAgent{
		ID:             core.NewID("ag"),
		WSConnectionID: "test-conn-1",
		OrgID:          models.OrgID(testOrg.ID),
		CCAgentID:      "test-agent-1",
		RepoURL:        "https://github.com/test/repo.git",
	}
	err = agentsRepo.UpsertActiveAgent(context.Background(), testAgent)
	require.NoError(t, err)

	// Create mock agents service that returns our test agent
	mockAgentsService := &testutils.MockAgentsService{}
	mockAgentsService.On("GetConnectedActiveAgents", context.Background(), models.OrgID(testOrg.ID), []string{}).
		Return([]*models.ActiveAgent{testAgent}, nil)

	defer func() {
		// Cleanup
		agentsRepo.DeleteActiveAgent(context.Background(), testAgent.ID, models.OrgID(testOrg.ID))
	}()

	service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

	t.Run("Create new Discord channel with default repo URL", func(t *testing.T) {
		guildID := "987654321098765432"
		channelID := "123456789012345678"

		channel, err := service.UpsertDiscordConnectedChannel(context.Background(), models.OrgID(testOrg.ID), guildID, channelID)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, models.OrgID(testOrg.ID))

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, models.OrgID(testOrg.ID), channel.OrgID)
		assert.Equal(t, guildID, channel.GuildID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.NotNil(t, channel.DefaultRepoURL)
		assert.Equal(t, testAgent.RepoURL, *channel.DefaultRepoURL)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("Update existing Discord channel preserves default repo URL", func(t *testing.T) {
		guildID := "111111111111111111"
		channelID := "222222222222222222"
		originalRepoURL := "https://github.com/original/repo.git"

		// Create channel with original repo URL
		originalChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      nil,
			SlackChannelID:   nil,
			DiscordGuildID:   &guildID,
			DiscordChannelID: &channelID,
			DefaultRepoURL:   &originalRepoURL,
		}
		err := connectedChannelsRepo.UpsertDiscordConnectedChannel(context.Background(), originalChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), originalChannel.ID, models.OrgID(testOrg.ID))

		// Upsert again should preserve the original repo URL
		updatedChannel, err := service.UpsertDiscordConnectedChannel(context.Background(), models.OrgID(testOrg.ID), guildID, channelID)
		require.NoError(t, err)

		assert.Equal(t, originalChannel.ID, updatedChannel.ID)
		assert.Equal(t, originalRepoURL, *updatedChannel.DefaultRepoURL)
		// Should not call GetConnectedActiveAgents for existing channel
		mockAgentsService.AssertNotCalled(t, "GetConnectedActiveAgents")
	})

	t.Run("Empty guild ID returns error", func(t *testing.T) {
		_, err := service.UpsertDiscordConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "", "123456789012345678")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "guild ID cannot be empty")
	})

	t.Run("Empty channel ID returns error", func(t *testing.T) {
		_, err := service.UpsertDiscordConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "987654321098765432", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel ID cannot be empty")
	})
}

func TestConnectedChannelsService_GetSlackConnectedChannel(t *testing.T) {
	dbConn, err := testutils.SetupTestDB()
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, testutils.TestSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, testutils.TestSchema)

	// Create test organization
	testOrg := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), testOrg)
	require.NoError(t, err)

	mockAgentsService := &testutils.MockAgentsService{}
	service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

	t.Run("Get existing Slack channel", func(t *testing.T) {
		teamID := "T1234567890"
		channelID := "C1234567890"
		repoURL := "https://github.com/test/repo.git"

		// Create test channel
		testChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      &teamID,
			SlackChannelID:   &channelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &repoURL,
		}
		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), testChannel.ID, models.OrgID(testOrg.ID))

		// Get the channel
		maybeChannel, err := service.GetSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), teamID, channelID)
		require.NoError(t, err)
		require.True(t, maybeChannel.IsPresent())

		channel := maybeChannel.MustGet()
		assert.Equal(t, testChannel.ID, channel.ID)
		assert.Equal(t, teamID, channel.TeamID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Equal(t, repoURL, *channel.DefaultRepoURL)
	})

	t.Run("Get non-existent Slack channel", func(t *testing.T) {
		maybeChannel, err := service.GetSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "T9999999999", "C9999999999")
		require.NoError(t, err)
		assert.False(t, maybeChannel.IsPresent())
	})

	t.Run("Empty team ID returns error", func(t *testing.T) {
		_, err := service.GetSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "", "C1234567890")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team ID cannot be empty")
	})

	t.Run("Empty channel ID returns error", func(t *testing.T) {
		_, err := service.GetSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "T1234567890", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel ID cannot be empty")
	})
}

func TestConnectedChannelsService_GetDiscordConnectedChannel(t *testing.T) {
	dbConn, err := testutils.SetupTestDB()
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, testutils.TestSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, testutils.TestSchema)

	// Create test organization
	testOrg := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), testOrg)
	require.NoError(t, err)

	mockAgentsService := &testutils.MockAgentsService{}
	service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

	t.Run("Get existing Discord channel", func(t *testing.T) {
		guildID := "987654321098765432"
		channelID := "123456789012345678"
		repoURL := "https://github.com/test/repo.git"

		// Create test channel
		testChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      nil,
			SlackChannelID:   nil,
			DiscordGuildID:   &guildID,
			DiscordChannelID: &channelID,
			DefaultRepoURL:   &repoURL,
		}
		err := connectedChannelsRepo.UpsertDiscordConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), testChannel.ID, models.OrgID(testOrg.ID))

		// Get the channel
		maybeChannel, err := service.GetDiscordConnectedChannel(context.Background(), models.OrgID(testOrg.ID), guildID, channelID)
		require.NoError(t, err)
		require.True(t, maybeChannel.IsPresent())

		channel := maybeChannel.MustGet()
		assert.Equal(t, testChannel.ID, channel.ID)
		assert.Equal(t, guildID, channel.GuildID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Equal(t, repoURL, *channel.DefaultRepoURL)
	})

	t.Run("Get non-existent Discord channel", func(t *testing.T) {
		maybeChannel, err := service.GetDiscordConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "999999999999999999", "888888888888888888")
		require.NoError(t, err)
		assert.False(t, maybeChannel.IsPresent())
	})

	t.Run("Empty guild ID returns error", func(t *testing.T) {
		_, err := service.GetDiscordConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "", "123456789012345678")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "guild ID cannot be empty")
	})

	t.Run("Empty channel ID returns error", func(t *testing.T) {
		_, err := service.GetDiscordConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "987654321098765432", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel ID cannot be empty")
	})
}

func TestConnectedChannelsService_GetConnectedChannelsByOrganization(t *testing.T) {
	dbConn, err := testutils.SetupTestDB()
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, testutils.TestSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, testutils.TestSchema)

	// Create test organization
	testOrg := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), testOrg)
	require.NoError(t, err)

	mockAgentsService := &testutils.MockAgentsService{}
	service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

	t.Run("Get channels for organization", func(t *testing.T) {
		repoURL := "https://github.com/test/repo.git"

		// Create test channels
		slackTeamID := "T1234567890"
		slackChannelID := "C1234567890"
		slackChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            models.OrgID(testOrg.ID),
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
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      nil,
			SlackChannelID:   nil,
			DiscordGuildID:   &discordGuildID,
			DiscordChannelID: &discordChannelID,
			DefaultRepoURL:   &repoURL,
		}

		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), slackChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), slackChannel.ID, models.OrgID(testOrg.ID))

		err = connectedChannelsRepo.UpsertDiscordConnectedChannel(context.Background(), discordChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), discordChannel.ID, models.OrgID(testOrg.ID))

		// Get channels
		channels, err := service.GetConnectedChannelsByOrganization(context.Background(), models.OrgID(testOrg.ID))
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

	t.Run("Get channels for organization with no channels", func(t *testing.T) {
		// Create another organization
		emptyOrg := &models.Organization{
			ID: core.NewID("org"),
		}
		err := organizationsRepo.CreateOrganization(context.Background(), emptyOrg)
		require.NoError(t, err)

		channels, err := service.GetConnectedChannelsByOrganization(context.Background(), models.OrgID(emptyOrg.ID))
		require.NoError(t, err)
		assert.Len(t, channels, 0)
	})
}

func TestConnectedChannelsService_DeleteConnectedChannel(t *testing.T) {
	dbConn, err := testutils.SetupTestDB()
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, testutils.TestSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, testutils.TestSchema)

	// Create test organization
	testOrg := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), testOrg)
	require.NoError(t, err)

	mockAgentsService := &testutils.MockAgentsService{}
	service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

	t.Run("Delete existing channel", func(t *testing.T) {
		repoURL := "https://github.com/test/repo.git"
		teamID := "T1234567890"
		channelID := "C1234567890"
		testChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      &teamID,
			SlackChannelID:   &channelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &repoURL,
		}
		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)

		// Delete the channel
		err = service.DeleteConnectedChannel(context.Background(), models.OrgID(testOrg.ID), testChannel.ID)
		require.NoError(t, err)

		// Verify it's deleted
		maybeChannel, err := connectedChannelsRepo.GetConnectedChannelByID(context.Background(), testChannel.ID, models.OrgID(testOrg.ID))
		require.NoError(t, err)
		assert.False(t, maybeChannel.IsPresent())
	})

	t.Run("Delete non-existent channel returns error", func(t *testing.T) {
		err := service.DeleteConnectedChannel(context.Background(), models.OrgID(testOrg.ID), core.NewID("cc"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connected channel not found")
	})

	t.Run("Invalid ID returns error", func(t *testing.T) {
		err := service.DeleteConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "invalid-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ID must be a valid ULID")
	})
}

func TestConnectedChannelsService_GetConnectedChannelByID(t *testing.T) {
	dbConn, err := testutils.SetupTestDB()
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, testutils.TestSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, testutils.TestSchema)

	// Create test organization
	testOrg := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), testOrg)
	require.NoError(t, err)

	mockAgentsService := &testutils.MockAgentsService{}
	service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

	t.Run("Get existing Slack channel by ID", func(t *testing.T) {
		teamID := "T1234567890"
		channelID := "C1234567890"
		repoURL := "https://github.com/test/repo.git"

		// Create test channel
		testChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      &teamID,
			SlackChannelID:   &channelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &repoURL,
		}
		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), testChannel.ID, models.OrgID(testOrg.ID))

		// Get the channel by ID
		maybeChannel, err := service.GetConnectedChannelByID(context.Background(), models.OrgID(testOrg.ID), testChannel.ID)
		require.NoError(t, err)
		require.True(t, maybeChannel.IsPresent())

		channel := maybeChannel.MustGet()
		slackChannel, ok := channel.(*models.SlackConnectedChannel)
		require.True(t, ok, "Channel should be a SlackConnectedChannel")
		assert.Equal(t, testChannel.ID, slackChannel.ID)
		assert.Equal(t, teamID, slackChannel.TeamID)
		assert.Equal(t, channelID, slackChannel.ChannelID)
		assert.Equal(t, repoURL, *slackChannel.DefaultRepoURL)
	})

	t.Run("Get existing Discord channel by ID", func(t *testing.T) {
		guildID := "987654321098765432"
		channelID := "123456789012345678"
		repoURL := "https://github.com/test/repo.git"

		// Create test channel
		testChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      nil,
			SlackChannelID:   nil,
			DiscordGuildID:   &guildID,
			DiscordChannelID: &channelID,
			DefaultRepoURL:   &repoURL,
		}
		err := connectedChannelsRepo.UpsertDiscordConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), testChannel.ID, models.OrgID(testOrg.ID))

		// Get the channel by ID
		maybeChannel, err := service.GetConnectedChannelByID(context.Background(), models.OrgID(testOrg.ID), testChannel.ID)
		require.NoError(t, err)
		require.True(t, maybeChannel.IsPresent())

		channel := maybeChannel.MustGet()
		discordChannel, ok := channel.(*models.DiscordConnectedChannel)
		require.True(t, ok, "Channel should be a DiscordConnectedChannel")
		assert.Equal(t, testChannel.ID, discordChannel.ID)
		assert.Equal(t, guildID, discordChannel.GuildID)
		assert.Equal(t, channelID, discordChannel.ChannelID)
		assert.Equal(t, repoURL, *discordChannel.DefaultRepoURL)
	})

	t.Run("Get non-existent channel", func(t *testing.T) {
		maybeChannel, err := service.GetConnectedChannelByID(context.Background(), models.OrgID(testOrg.ID), core.NewID("cc"))
		require.NoError(t, err)
		assert.False(t, maybeChannel.IsPresent())
	})

	t.Run("Invalid ID returns error", func(t *testing.T) {
		_, err := service.GetConnectedChannelByID(context.Background(), models.OrgID(testOrg.ID), "invalid-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ID must be a valid ULID")
	})
}

func TestConnectedChannelsService_UpdateConnectedChannelDefaultRepoURL(t *testing.T) {
	dbConn, err := testutils.SetupTestDB()
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, testutils.TestSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, testutils.TestSchema)

	// Create test organization
	testOrg := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), testOrg)
	require.NoError(t, err)

	mockAgentsService := &testutils.MockAgentsService{}
	service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

	t.Run("Update existing channel repo URL", func(t *testing.T) {
		teamID := "T1234567890"
		channelID := "C1234567890"
		originalRepoURL := "https://github.com/original/repo.git"
		newRepoURL := "https://github.com/updated/repo.git"

		// Create test channel
		testChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      &teamID,
			SlackChannelID:   &channelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &originalRepoURL,
		}
		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), testChannel.ID, models.OrgID(testOrg.ID))

		// Update the repo URL
		err = service.UpdateConnectedChannelDefaultRepoURL(context.Background(), models.OrgID(testOrg.ID), testChannel.ID, &newRepoURL)
		require.NoError(t, err)

		// Verify the update
		maybeChannel, err := service.GetSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), teamID, channelID)
		require.NoError(t, err)
		require.True(t, maybeChannel.IsPresent())

		channel := maybeChannel.MustGet()
		assert.Equal(t, newRepoURL, *channel.DefaultRepoURL)
	})

	t.Run("Clear repo URL with nil", func(t *testing.T) {
		teamID := "T0987654321"
		channelID := "C0987654321"
		originalRepoURL := "https://github.com/original/repo.git"

		// Create test channel
		testChannel := &models.DatabaseConnectedChannel{
			ID:               core.NewID("cc"),
			OrgID:            models.OrgID(testOrg.ID),
			SlackTeamID:      &teamID,
			SlackChannelID:   &channelID,
			DiscordGuildID:   nil,
			DiscordChannelID: nil,
			DefaultRepoURL:   &originalRepoURL,
		}
		err := connectedChannelsRepo.UpsertSlackConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), testChannel.ID, models.OrgID(testOrg.ID))

		// Clear the repo URL
		err = service.UpdateConnectedChannelDefaultRepoURL(context.Background(), models.OrgID(testOrg.ID), testChannel.ID, nil)
		require.NoError(t, err)

		// Verify the update
		maybeChannel, err := service.GetSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), teamID, channelID)
		require.NoError(t, err)
		require.True(t, maybeChannel.IsPresent())

		channel := maybeChannel.MustGet()
		assert.Nil(t, channel.DefaultRepoURL)
	})

	t.Run("Update non-existent channel returns error", func(t *testing.T) {
		newRepoURL := "https://github.com/test/repo.git"
		err := service.UpdateConnectedChannelDefaultRepoURL(context.Background(), models.OrgID(testOrg.ID), core.NewID("cc"), &newRepoURL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connected channel not found")
	})

	t.Run("Invalid ID returns error", func(t *testing.T) {
		newRepoURL := "https://github.com/test/repo.git"
		err := service.UpdateConnectedChannelDefaultRepoURL(context.Background(), models.OrgID(testOrg.ID), "invalid-id", &newRepoURL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ID must be a valid ULID")
	})
}

func TestConnectedChannelsService_RepoURLAssignment(t *testing.T) {
	dbConn, err := testutils.SetupTestDB()
	require.NoError(t, err)
	defer dbConn.Close()

	connectedChannelsRepo := db.NewPostgresConnectedChannelsRepository(dbConn, testutils.TestSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, testutils.TestSchema)

	// Create test organization
	testOrg := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), testOrg)
	require.NoError(t, err)

	t.Run("New Slack channel with no agents available", func(t *testing.T) {
		// Create mock that returns no agents
		mockAgentsService := &testutils.MockAgentsService{}
		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), models.OrgID(testOrg.ID), []string{}).
			Return([]*models.ActiveAgent{}, nil)

		service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

		teamID := "T1111111111"
		channelID := "C1111111111"

		channel, err := service.UpsertSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), teamID, channelID)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, models.OrgID(testOrg.ID))

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, models.OrgID(testOrg.ID), channel.OrgID)
		assert.Equal(t, teamID, channel.TeamID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Nil(t, channel.DefaultRepoURL)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("New Discord channel with no agents available", func(t *testing.T) {
		// Create mock that returns no agents
		mockAgentsService := &testutils.MockAgentsService{}
		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), models.OrgID(testOrg.ID), []string{}).
			Return([]*models.ActiveAgent{}, nil)

		service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

		guildID := "987654321098765432"
		channelID := "111111111111111111"

		channel, err := service.UpsertDiscordConnectedChannel(context.Background(), models.OrgID(testOrg.ID), guildID, channelID)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, models.OrgID(testOrg.ID))

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, models.OrgID(testOrg.ID), channel.OrgID)
		assert.Equal(t, guildID, channel.GuildID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Nil(t, channel.DefaultRepoURL)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("New channel with agent having empty repo URL", func(t *testing.T) {
		// Create agent with empty repo URL
		testAgent := &models.ActiveAgent{
			ID:             core.NewID("ag"),
			WSConnectionID: "test-conn-1",
			OrgID:          models.OrgID(testOrg.ID),
			CCAgentID:      "test-agent-1",
			RepoURL:        "", // Empty repo URL
		}

		mockAgentsService := &testutils.MockAgentsService{}
		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), models.OrgID(testOrg.ID), []string{}).
			Return([]*models.ActiveAgent{testAgent}, nil)

		service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

		teamID := "T2222222222"
		channelID := "C2222222222"

		channel, err := service.UpsertSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), teamID, channelID)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, models.OrgID(testOrg.ID))

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, models.OrgID(testOrg.ID), channel.OrgID)
		assert.Equal(t, teamID, channel.TeamID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Nil(t, channel.DefaultRepoURL)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("Error getting agents is handled gracefully", func(t *testing.T) {
		// Create mock that returns error
		mockAgentsService := &testutils.MockAgentsService{}
		mockAgentsService.On("GetConnectedActiveAgents", context.Background(), models.OrgID(testOrg.ID), []string{}).
			Return([]*models.ActiveAgent{}, errors.New("failed to get agents"))

		service := connectedchannels.NewConnectedChannelsService(connectedChannelsRepo, mockAgentsService)

		teamID := "T3333333333"
		channelID := "C3333333333"

		channel, err := service.UpsertSlackConnectedChannel(context.Background(), models.OrgID(testOrg.ID), teamID, channelID)
		require.NoError(t, err) // Should succeed despite error getting agents
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, models.OrgID(testOrg.ID))

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, models.OrgID(testOrg.ID), channel.OrgID)
		assert.Equal(t, teamID, channel.TeamID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Nil(t, channel.DefaultRepoURL) // Should be nil when agent service fails
		mockAgentsService.AssertExpectations(t)
	})
}