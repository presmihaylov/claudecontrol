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

func (s *ConnectedChannelsService) UpsertConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	channelID string,
	channelType string,
) (*models.ConnectedChannel, error) {
	log.Printf("📋 Starting to upsert connected channel: %s (%s) for org: %s", channelID, channelType, orgID)

	if channelID == "" {
		return nil, fmt.Errorf("channel ID cannot be empty")
	}
	if channelType != models.ChannelTypeSlack && channelType != models.ChannelTypeDiscord {
		return nil, fmt.Errorf("channel type must be 'slack' or 'discord'")
	}

	// Check if channel already exists
	existingChannel, err := s.connectedChannelsRepo.GetConnectedChannelByChannelID(ctx, orgID, channelID, channelType)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing channel: %w", err)
	}

	var defaultRepoURL *string
	if !existingChannel.IsPresent() {
		// New channel - assign default repo URL from first available agent
		defaultRepoURL, err = s.getFirstAvailableRepoURL(ctx, orgID)
		if err != nil {
			log.Printf("⚠️ Failed to get default repo URL for new channel, continuing without it: %v", err)
			defaultRepoURL = nil
		}
		log.Printf("📋 New channel detected, assigned default repo URL: %v", defaultRepoURL)
	} else {
		// Existing channel - preserve current default_repo_url
		existing := existingChannel.MustGet()
		defaultRepoURL = existing.DefaultRepoURL
		log.Printf("📋 Existing channel detected, preserving default repo URL: %v", defaultRepoURL)
	}

	channel := &models.ConnectedChannel{
		ID:             core.NewID("cc"),
		OrgID:          orgID,
		ChannelID:      channelID,
		ChannelType:    channelType,
		DefaultRepoURL: defaultRepoURL,
	}

	if err := s.connectedChannelsRepo.UpsertConnectedChannel(ctx, channel); err != nil {
		return nil, fmt.Errorf("failed to upsert connected channel: %w", err)
	}

	log.Printf("📋 Completed successfully - upserted connected channel with ID: %s", channel.ID)
	return channel, nil
}

func (s *ConnectedChannelsService) GetConnectedChannelByChannelID(
	ctx context.Context,
	orgID models.OrgID,
	channelID string,
	channelType string,
) (mo.Option[*models.ConnectedChannel], error) {
	log.Printf("📋 Starting to get connected channel by channel ID: %s (%s) for org: %s", channelID, channelType, orgID)

	if channelID == "" {
		return mo.None[*models.ConnectedChannel](), fmt.Errorf("channel ID cannot be empty")
	}
	if channelType != models.ChannelTypeSlack && channelType != models.ChannelTypeDiscord {
		return mo.None[*models.ConnectedChannel](), fmt.Errorf("channel type must be 'slack' or 'discord'")
	}

	channel, err := s.connectedChannelsRepo.GetConnectedChannelByChannelID(ctx, orgID, channelID, channelType)
	if err != nil {
		return mo.None[*models.ConnectedChannel](), fmt.Errorf("failed to get connected channel: %w", err)
	}

	if channel.IsPresent() {
		log.Printf("📋 Completed successfully - found connected channel with ID: %s", channel.MustGet().ID)
	} else {
		log.Printf("📋 Completed successfully - no connected channel found for channel ID: %s", channelID)
	}
	return channel, nil
}

func (s *ConnectedChannelsService) GetConnectedChannelByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.ConnectedChannel], error) {
	log.Printf("📋 Starting to get connected channel by ID: %s for org: %s", id, orgID)

	if !core.IsValidULID(id) {
		return mo.None[*models.ConnectedChannel](), fmt.Errorf("ID must be a valid ULID")
	}

	channel, err := s.connectedChannelsRepo.GetConnectedChannelByID(ctx, id, orgID)
	if err != nil {
		return mo.None[*models.ConnectedChannel](), fmt.Errorf("failed to get connected channel: %w", err)
	}

	if channel.IsPresent() {
		log.Printf("📋 Completed successfully - found connected channel: %s", id)
	} else {
		log.Printf("📋 Completed successfully - connected channel not found: %s", id)
	}
	return channel, nil
}

func (s *ConnectedChannelsService) GetConnectedChannelsByOrganization(
	ctx context.Context,
	orgID models.OrgID,
) ([]*models.ConnectedChannel, error) {
	log.Printf("📋 Starting to get connected channels for org: %s", orgID)

	channels, err := s.connectedChannelsRepo.GetConnectedChannelsByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connected channels: %w", err)
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