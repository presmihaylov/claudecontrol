package connectedchannels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services/connectedchannels"
	"ccbackend/testutils"
)

func TestConnectedChannelsService_UpsertConnectedChannel(t *testing.T) {
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
		channelID := "C1234567890"
		channelType := models.ChannelTypeSlack

		channel, err := service.UpsertConnectedChannel(context.Background(), models.OrgID(testOrg.ID), channelID, channelType)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, models.OrgID(testOrg.ID))

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, models.OrgID(testOrg.ID), channel.OrgID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Equal(t, channelType, channel.ChannelType)
		assert.NotNil(t, channel.DefaultRepoURL)
		assert.Equal(t, testAgent.RepoURL, *channel.DefaultRepoURL)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("Create new Discord channel with default repo URL", func(t *testing.T) {
		channelID := "987654321098765432"
		channelType := models.ChannelTypeDiscord

		channel, err := service.UpsertConnectedChannel(context.Background(), models.OrgID(testOrg.ID), channelID, channelType)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), channel.ID, models.OrgID(testOrg.ID))

		assert.NotEmpty(t, channel.ID)
		assert.Equal(t, models.OrgID(testOrg.ID), channel.OrgID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Equal(t, channelType, channel.ChannelType)
		assert.NotNil(t, channel.DefaultRepoURL)
		assert.Equal(t, testAgent.RepoURL, *channel.DefaultRepoURL)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("Update existing channel preserves default repo URL", func(t *testing.T) {
		channelID := "C0987654321"
		channelType := models.ChannelTypeSlack
		originalRepoURL := "https://github.com/original/repo.git"

		// Create channel with original repo URL
		originalChannel := &models.ConnectedChannel{
			ID:             core.NewID("cc"),
			OrgID:          models.OrgID(testOrg.ID),
			ChannelID:      channelID,
			ChannelType:    channelType,
			DefaultRepoURL: &originalRepoURL,
		}
		err := connectedChannelsRepo.UpsertConnectedChannel(context.Background(), originalChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), originalChannel.ID, models.OrgID(testOrg.ID))

		// Upsert again should preserve the original repo URL
		updatedChannel, err := service.UpsertConnectedChannel(context.Background(), models.OrgID(testOrg.ID), channelID, channelType)
		require.NoError(t, err)

		assert.Equal(t, originalChannel.ID, updatedChannel.ID)
		assert.Equal(t, originalRepoURL, *updatedChannel.DefaultRepoURL)
		// Should not call GetConnectedActiveAgents for existing channel
		mockAgentsService.AssertNotCalled(t, "GetConnectedActiveAgents")
	})

	t.Run("Invalid channel type returns error", func(t *testing.T) {
		channelID := "C1234567890"
		invalidChannelType := "invalid"

		_, err := service.UpsertConnectedChannel(context.Background(), models.OrgID(testOrg.ID), channelID, invalidChannelType)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel type must be 'slack' or 'discord'")
	})

	t.Run("Empty channel ID returns error", func(t *testing.T) {
		_, err := service.UpsertConnectedChannel(context.Background(), models.OrgID(testOrg.ID), "", models.ChannelTypeSlack)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel ID cannot be empty")
	})
}

func TestConnectedChannelsService_GetConnectedChannelByChannelID(t *testing.T) {
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

	t.Run("Get existing channel", func(t *testing.T) {
		channelID := "C1234567890"
		channelType := models.ChannelTypeSlack
		repoURL := "https://github.com/test/repo.git"

		// Create test channel
		testChannel := &models.ConnectedChannel{
			ID:             core.NewID("cc"),
			OrgID:          models.OrgID(testOrg.ID),
			ChannelID:      channelID,
			ChannelType:    channelType,
			DefaultRepoURL: &repoURL,
		}
		err := connectedChannelsRepo.UpsertConnectedChannel(context.Background(), testChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), testChannel.ID, models.OrgID(testOrg.ID))

		// Get the channel
		maybeChannel, err := service.GetConnectedChannelByChannelID(context.Background(), models.OrgID(testOrg.ID), channelID, channelType)
		require.NoError(t, err)
		require.True(t, maybeChannel.IsPresent())

		channel := maybeChannel.MustGet()
		assert.Equal(t, testChannel.ID, channel.ID)
		assert.Equal(t, channelID, channel.ChannelID)
		assert.Equal(t, channelType, channel.ChannelType)
		assert.Equal(t, repoURL, *channel.DefaultRepoURL)
	})

	t.Run("Get non-existent channel", func(t *testing.T) {
		maybeChannel, err := service.GetConnectedChannelByChannelID(context.Background(), models.OrgID(testOrg.ID), "nonexistent", models.ChannelTypeSlack)
		require.NoError(t, err)
		assert.False(t, maybeChannel.IsPresent())
	})

	t.Run("Invalid channel type returns error", func(t *testing.T) {
		_, err := service.GetConnectedChannelByChannelID(context.Background(), models.OrgID(testOrg.ID), "C1234567890", "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel type must be 'slack' or 'discord'")
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
		slackChannel := &models.ConnectedChannel{
			ID:             core.NewID("cc"),
			OrgID:          models.OrgID(testOrg.ID),
			ChannelID:      "C1234567890",
			ChannelType:    models.ChannelTypeSlack,
			DefaultRepoURL: &repoURL,
		}
		discordChannel := &models.ConnectedChannel{
			ID:             core.NewID("cc"),
			OrgID:          models.OrgID(testOrg.ID),
			ChannelID:      "987654321098765432",
			ChannelType:    models.ChannelTypeDiscord,
			DefaultRepoURL: &repoURL,
		}

		err := connectedChannelsRepo.UpsertConnectedChannel(context.Background(), slackChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), slackChannel.ID, models.OrgID(testOrg.ID))

		err = connectedChannelsRepo.UpsertConnectedChannel(context.Background(), discordChannel)
		require.NoError(t, err)
		defer connectedChannelsRepo.DeleteConnectedChannel(context.Background(), discordChannel.ID, models.OrgID(testOrg.ID))

		// Get channels
		channels, err := service.GetConnectedChannelsByOrganization(context.Background(), models.OrgID(testOrg.ID))
		require.NoError(t, err)
		assert.Len(t, channels, 2)

		// Verify channels are returned
		channelIDs := make([]string, len(channels))
		for i, ch := range channels {
			channelIDs[i] = ch.ChannelID
		}
		assert.Contains(t, channelIDs, "C1234567890")
		assert.Contains(t, channelIDs, "987654321098765432")
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
		testChannel := &models.ConnectedChannel{
			ID:             core.NewID("cc"),
			OrgID:          models.OrgID(testOrg.ID),
			ChannelID:      "C1234567890",
			ChannelType:    models.ChannelTypeSlack,
			DefaultRepoURL: &repoURL,
		}
		err := connectedChannelsRepo.UpsertConnectedChannel(context.Background(), testChannel)
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