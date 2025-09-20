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

	// Get default repo URL from first available agent
	// Note: Since we don't have a get function for Discord channels, we always assign a repo URL
	defaultRepoURL, err := s.getFirstAvailableRepoURL(ctx, orgID)
	if err != nil {
		log.Printf("❌ Failed to get default repo URL for Discord channel: %v", err)
		return nil, fmt.Errorf("failed to get default repo URL for Discord channel: %w", err)
	}
	log.Printf("📋 Assigned default repo URL for Discord channel: %v", defaultRepoURL)

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