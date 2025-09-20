package connectedchannels

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services"
)

type ConnectedChannelsService struct {
	connectedChannelsRepo *db.PostgresConnectedChannelsRepository
	agentsService         services.AgentsService
}

func NewConnectedChannelsService(
	repo *db.PostgresConnectedChannelsRepository,
	agentsService services.AgentsService,
) *ConnectedChannelsService {
	return &ConnectedChannelsService{
		connectedChannelsRepo: repo,
		agentsService:         agentsService,
	}
}

// Slack-specific methods

func (s *ConnectedChannelsService) UpsertSlackConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	teamID string,
	channelID string,
) (*models.SlackConnectedChannel, error) {
	log.Printf("📋 Starting to upsert Slack connected channel: %s (team: %s) for org: %s", channelID, teamID, orgID)

	if teamID == "" {
		return nil, fmt.Errorf("team ID cannot be empty")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channel ID cannot be empty")
	}

	// Check if channel already exists
	existingChannel, err := s.connectedChannelsRepo.GetSlackConnectedChannel(ctx, orgID, teamID, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing Slack channel: %w", err)
	}

	var defaultRepoURL *string
	if !existingChannel.IsPresent() {
		// New channel - assign default repo URL from first available agent
		defaultRepoURL, err = s.getFirstAvailableRepoURL(ctx, orgID)
		if err != nil {
			log.Printf("❌ Failed to get default repo URL for new Slack channel: %v", err)
			return nil, fmt.Errorf("failed to get default repo URL for new Slack channel: %w", err)
		}
		log.Printf("📋 New Slack channel detected, assigned default repo URL: %v", defaultRepoURL)
	} else {
		// Existing channel - preserve current default_repo_url
		existing := existingChannel.MustGet()
		defaultRepoURL = existing.DefaultRepoURL
		log.Printf("📋 Existing Slack channel detected, preserving default repo URL: %v", defaultRepoURL)
	}

	// Create database model for upsert
	dbChannel := &db.DatabaseConnectedChannel{
		ID:               core.NewID("cc"),
		OrgID:            orgID,
		SlackTeamID:      &teamID,
		SlackChannelID:   &channelID,
		DiscordGuildID:   nil,
		DiscordChannelID: nil,
		DefaultRepoURL:   defaultRepoURL,
	}

	if err := s.connectedChannelsRepo.UpsertSlackConnectedChannel(ctx, dbChannel); err != nil {
		return nil, fmt.Errorf("failed to upsert Slack connected channel: %w", err)
	}

	// Convert to domain model
	slackChannel, err := dbChannel.ToSlackConnectedChannel()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to Slack domain model: %w", err)
	}

	log.Printf("📋 Completed successfully - upserted Slack connected channel with ID: %s", slackChannel.ID)
	return slackChannel, nil
}

func (s *ConnectedChannelsService) GetSlackConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	teamID string,
	channelID string,
) (mo.Option[*models.SlackConnectedChannel], error) {
	log.Printf("📋 Starting to get Slack connected channel: %s (team: %s) for org: %s", channelID, teamID, orgID)

	if teamID == "" {
		return mo.None[*models.SlackConnectedChannel](), fmt.Errorf("team ID cannot be empty")
	}
	if channelID == "" {
		return mo.None[*models.SlackConnectedChannel](), fmt.Errorf("channel ID cannot be empty")
	}

	dbChannel, err := s.connectedChannelsRepo.GetSlackConnectedChannel(ctx, orgID, teamID, channelID)
	if err != nil {
		return mo.None[*models.SlackConnectedChannel](), fmt.Errorf("failed to get Slack connected channel: %w", err)
	}

	if !dbChannel.IsPresent() {
		log.Printf("📋 Completed successfully - no Slack connected channel found for channel: %s", channelID)
		return mo.None[*models.SlackConnectedChannel](), nil
	}

	// Convert to domain model
	slackChannel, err := dbChannel.MustGet().ToSlackConnectedChannel()
	if err != nil {
		return mo.None[*models.SlackConnectedChannel](), fmt.Errorf("failed to convert to Slack domain model: %w", err)
	}

	log.Printf("📋 Completed successfully - found Slack connected channel with ID: %s", slackChannel.ID)
	return mo.Some(slackChannel), nil
}

// Discord-specific methods

func (s *ConnectedChannelsService) UpsertDiscordConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	guildID string,
	channelID string,
) (*models.DiscordConnectedChannel, error) {
	log.Printf("📋 Starting to upsert Discord connected channel: %s (guild: %s) for org: %s", channelID, guildID, orgID)

	if guildID == "" {
		return nil, fmt.Errorf("guild ID cannot be empty")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channel ID cannot be empty")
	}

	// Check if channel already exists
	existingChannel, err := s.connectedChannelsRepo.GetDiscordConnectedChannel(ctx, orgID, guildID, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing Discord channel: %w", err)
	}

	var defaultRepoURL *string
	if !existingChannel.IsPresent() {
		// New channel - assign default repo URL from first available agent
		defaultRepoURL, err = s.getFirstAvailableRepoURL(ctx, orgID)
		if err != nil {
			log.Printf("❌ Failed to get default repo URL for new Discord channel: %v", err)
			return nil, fmt.Errorf("failed to get default repo URL for new Discord channel: %w", err)
		}
		log.Printf("📋 New Discord channel detected, assigned default repo URL: %v", defaultRepoURL)
	} else {
		// Existing channel - preserve current default_repo_url
		existing := existingChannel.MustGet()
		defaultRepoURL = existing.DefaultRepoURL
		log.Printf("📋 Existing Discord channel detected, preserving default repo URL: %v", defaultRepoURL)
	}

	// Create database model for upsert
	dbChannel := &db.DatabaseConnectedChannel{
		ID:               core.NewID("cc"),
		OrgID:            orgID,
		SlackTeamID:      nil,
		SlackChannelID:   nil,
		DiscordGuildID:   &guildID,
		DiscordChannelID: &channelID,
		DefaultRepoURL:   defaultRepoURL,
	}

	if err := s.connectedChannelsRepo.UpsertDiscordConnectedChannel(ctx, dbChannel); err != nil {
		return nil, fmt.Errorf("failed to upsert Discord connected channel: %w", err)
	}

	// Convert to domain model
	discordChannel, err := dbChannel.ToDiscordConnectedChannel()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to Discord domain model: %w", err)
	}

	log.Printf("📋 Completed successfully - upserted Discord connected channel with ID: %s", discordChannel.ID)
	return discordChannel, nil
}

func (s *ConnectedChannelsService) GetDiscordConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	guildID string,
	channelID string,
) (mo.Option[*models.DiscordConnectedChannel], error) {
	log.Printf("📋 Starting to get Discord connected channel: %s (guild: %s) for org: %s", channelID, guildID, orgID)

	if guildID == "" {
		return mo.None[*models.DiscordConnectedChannel](), fmt.Errorf("guild ID cannot be empty")
	}
	if channelID == "" {
		return mo.None[*models.DiscordConnectedChannel](), fmt.Errorf("channel ID cannot be empty")
	}

	dbChannel, err := s.connectedChannelsRepo.GetDiscordConnectedChannel(ctx, orgID, guildID, channelID)
	if err != nil {
		return mo.None[*models.DiscordConnectedChannel](), fmt.Errorf("failed to get Discord connected channel: %w", err)
	}

	if !dbChannel.IsPresent() {
		log.Printf("📋 Completed successfully - no Discord connected channel found for channel: %s", channelID)
		return mo.None[*models.DiscordConnectedChannel](), nil
	}

	// Convert to domain model
	discordChannel, err := dbChannel.MustGet().ToDiscordConnectedChannel()
	if err != nil {
		return mo.None[*models.DiscordConnectedChannel](), fmt.Errorf("failed to convert to Discord domain model: %w", err)
	}

	log.Printf("📋 Completed successfully - found Discord connected channel with ID: %s", discordChannel.ID)
	return mo.Some(discordChannel), nil
}

// Common methods

func (s *ConnectedChannelsService) GetConnectedChannelByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[models.ConnectedChannel], error) {
	log.Printf("📋 Starting to get connected channel by ID: %s for org: %s", id, orgID)

	if !core.IsValidULID(id) {
		return mo.None[models.ConnectedChannel](), fmt.Errorf("ID must be a valid ULID")
	}

	dbChannel, err := s.connectedChannelsRepo.GetConnectedChannelByID(ctx, id, orgID)
	if err != nil {
		return mo.None[models.ConnectedChannel](), fmt.Errorf("failed to get connected channel: %w", err)
	}

	if !dbChannel.IsPresent() {
		log.Printf("📋 Completed successfully - connected channel not found: %s", id)
		return mo.None[models.ConnectedChannel](), nil
	}

	// Convert to domain interface
	channel, err := dbChannel.MustGet().ToConnectedChannel()
	if err != nil {
		return mo.None[models.ConnectedChannel](), fmt.Errorf("failed to convert to domain model: %w", err)
	}

	log.Printf("📋 Completed successfully - found connected channel: %s", id)
	return mo.Some(channel), nil
}

func (s *ConnectedChannelsService) GetConnectedChannelsByOrganization(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.ConnectedChannel, error) {
	log.Printf("📋 Starting to get connected channels for org: %s", orgID)

	dbChannels, err := s.connectedChannelsRepo.GetConnectedChannelsByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connected channels: %w", err)
	}

	// Convert to domain models
	channels := make([]models.ConnectedChannel, 0, len(dbChannels))
	for _, dbChannel := range dbChannels {
		channel, err := dbChannel.ToConnectedChannel()
		if err != nil {
			log.Printf("❌ Failed to convert database channel to domain model: %v", err)
			return nil, fmt.Errorf("failed to convert database channel to domain model (channel ID: %s): %w", dbChannel.ID, err)
		}
		channels = append(channels, channel)
	}

	log.Printf("📋 Completed successfully - found %d connected channels for org: %s", len(channels), orgID)
	return channels, nil
}

func (s *ConnectedChannelsService) DeleteConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) error {
	log.Printf("📋 Starting to delete connected channel: %s for org: %s", id, orgID)

	if !core.IsValidULID(id) {
		return fmt.Errorf("ID must be a valid ULID")
	}

	deleted, err := s.connectedChannelsRepo.DeleteConnectedChannel(ctx, id, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete connected channel: %w", err)
	}

	if !deleted {
		return fmt.Errorf("connected channel not found")
	}

	log.Printf("📋 Completed successfully - deleted connected channel: %s", id)
	return nil
}

func (s *ConnectedChannelsService) UpdateConnectedChannelDefaultRepoURL(
	ctx context.Context,
	orgID models.OrgID,
	id string,
	defaultRepoURL *string,
) error {
	log.Printf("📋 Starting to update default repo URL for connected channel: %s for org: %s", id, orgID)

	if !core.IsValidULID(id) {
		return fmt.Errorf("ID must be a valid ULID")
	}

	updated, err := s.connectedChannelsRepo.UpdateConnectedChannelDefaultRepoURL(ctx, id, orgID, defaultRepoURL)
	if err != nil {
		return fmt.Errorf("failed to update connected channel default repo URL: %w", err)
	}

	if !updated {
		return fmt.Errorf("connected channel not found")
	}

	log.Printf("📋 Completed successfully - updated default repo URL for connected channel: %s", id)
	return nil
}

// getFirstAvailableRepoURL gets the repository URL from the first available active agent
func (s *ConnectedChannelsService) getFirstAvailableRepoURL(ctx context.Context, orgID models.OrgID) (*string, error) {
	log.Printf("📋 Starting to get first available repo URL for org: %s", orgID)

	agents, err := s.agentsService.GetConnectedActiveAgents(ctx, orgID, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get active agents: %w", err)
	}

	if len(agents) == 0 {
		log.Printf("📋 No active agents found for org: %s", orgID)
		return nil, nil
	}

	// Get the first agent's repository URL
	firstAgent := agents[0]
	if firstAgent.RepoURL == "" {
		log.Printf("📋 First agent has empty repo URL for org: %s", orgID)
		return nil, nil
	}

	log.Printf("📋 Completed successfully - found repo URL from first agent: %s", firstAgent.RepoURL)
	return &firstAgent.RepoURL, nil
}