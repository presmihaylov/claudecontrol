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
		// Existing channel - check if repo URL is null and try to assign it
		existing := existingChannel.MustGet()
		if existing.DefaultRepoURL == nil {
			// Existing channel has no repo URL - try to assign one now
			defaultRepoURL, err = s.getFirstAvailableRepoURL(ctx, orgID)
			if err != nil {
				log.Printf("❌ Failed to get default repo URL for existing Slack channel: %v", err)
				return nil, fmt.Errorf("failed to get default repo URL for existing Slack channel: %w", err)
			}
			log.Printf("📋 Existing Slack channel had no repo URL, assigned default repo URL: %v", defaultRepoURL)
		} else {
			// Existing channel has repo URL - preserve it
			defaultRepoURL = existing.DefaultRepoURL
			log.Printf("📋 Existing Slack channel detected, preserving default repo URL: %v", defaultRepoURL)
		}
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

func (s *ConnectedChannelsService) UpdateSlackChannelDefaultRepo(
	ctx context.Context,
	orgID models.OrgID,
	teamID string,
	channelID string,
	repoURL string,
) (*models.SlackConnectedChannel, error) {
	log.Printf("📋 Starting to update Slack channel default repo: %s (team: %s) for org: %s to repo: %s", channelID, teamID, orgID, repoURL)

	if teamID == "" {
		return nil, fmt.Errorf("team ID cannot be empty")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channel ID cannot be empty")
	}
	if repoURL == "" {
		return nil, fmt.Errorf("repo URL cannot be empty")
	}

	// Create database model for upsert with new repo URL
	dbChannel := &db.DatabaseConnectedChannel{
		ID:               core.NewID("cc"),
		OrgID:            orgID,
		SlackTeamID:      &teamID,
		SlackChannelID:   &channelID,
		DiscordGuildID:   nil,
		DiscordChannelID: nil,
		DefaultRepoURL:   &repoURL,
	}

	if err := s.connectedChannelsRepo.UpsertSlackConnectedChannel(ctx, dbChannel); err != nil {
		return nil, fmt.Errorf("failed to update Slack connected channel default repo: %w", err)
	}

	// Convert to domain model
	slackChannel, err := dbChannel.ToSlackConnectedChannel()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to Slack domain model: %w", err)
	}

	log.Printf("📋 Completed successfully - updated Slack connected channel default repo with ID: %s", slackChannel.ID)
	return slackChannel, nil
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
		// Existing channel - check if repo URL is null and try to assign it
		existing := existingChannel.MustGet()
		if existing.DefaultRepoURL == nil {
			// Existing channel has no repo URL - try to assign one now
			defaultRepoURL, err = s.getFirstAvailableRepoURL(ctx, orgID)
			if err != nil {
				log.Printf("❌ Failed to get default repo URL for existing Discord channel: %v", err)
				return nil, fmt.Errorf("failed to get default repo URL for existing Discord channel: %w", err)
			}
			log.Printf("📋 Existing Discord channel had no repo URL, assigned default repo URL: %v", defaultRepoURL)
		} else {
			// Existing channel has repo URL - preserve it
			defaultRepoURL = existing.DefaultRepoURL
			log.Printf("📋 Existing Discord channel detected, preserving default repo URL: %v", defaultRepoURL)
		}
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

func (s *ConnectedChannelsService) UpdateDiscordChannelDefaultRepo(
	ctx context.Context,
	orgID models.OrgID,
	guildID string,
	channelID string,
	repoURL string,
) (*models.DiscordConnectedChannel, error) {
	log.Printf("📋 Starting to update Discord channel default repo: %s (guild: %s) for org: %s to repo: %s", channelID, guildID, orgID, repoURL)

	if guildID == "" {
		return nil, fmt.Errorf("guild ID cannot be empty")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channel ID cannot be empty")
	}
	if repoURL == "" {
		return nil, fmt.Errorf("repo URL cannot be empty")
	}

	// Create database model for upsert with new repo URL
	dbChannel := &db.DatabaseConnectedChannel{
		ID:               core.NewID("cc"),
		OrgID:            orgID,
		SlackTeamID:      nil,
		SlackChannelID:   nil,
		DiscordGuildID:   &guildID,
		DiscordChannelID: &channelID,
		DefaultRepoURL:   &repoURL,
	}

	if err := s.connectedChannelsRepo.UpsertDiscordConnectedChannel(ctx, dbChannel); err != nil {
		return nil, fmt.Errorf("failed to update Discord connected channel default repo: %w", err)
	}

	// Convert to domain model
	discordChannel, err := dbChannel.ToDiscordConnectedChannel()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to Discord domain model: %w", err)
	}

	log.Printf("📋 Completed successfully - updated Discord connected channel default repo with ID: %s", discordChannel.ID)
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