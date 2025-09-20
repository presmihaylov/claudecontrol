package handlers

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases"
	"ccbackend/usecases/core"
)

type DiscordEventsHandler struct {
	discordSDKClient           *discordgo.Session
	discordClient              clients.DiscordClient
	coreUseCase                *core.CoreUseCase
	discordIntegrationsService services.DiscordIntegrationsService
	discordUseCase             usecases.DiscordUseCaseInterface
	connectedChannelsService   services.ConnectedChannelsService
}

func NewDiscordEventsHandler(
	botToken string,
	discordClient clients.DiscordClient,
	coreUseCase *core.CoreUseCase,
	discordIntegrationsService services.DiscordIntegrationsService,
	discordUseCase usecases.DiscordUseCaseInterface,
	connectedChannelsService services.ConnectedChannelsService,
) (*DiscordEventsHandler, error) {
	// Create a new Discord session using the provided bot token
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	handler := &DiscordEventsHandler{
		discordSDKClient:           session,
		discordClient:              discordClient,
		coreUseCase:                coreUseCase,
		discordIntegrationsService: discordIntegrationsService,
		discordUseCase:             discordUseCase,
		connectedChannelsService:   connectedChannelsService,
	}

	// Register event handlers
	session.AddHandler(handler.handleMessageCreatedEvent)
	session.AddHandler(handler.handleReactionAddedEvent)

	// Set intents to receive message and reaction events
	session.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions

	return handler, nil
}

// StartBot opens the Discord connection and starts listening for events
func (h *DiscordEventsHandler) StartBot() error {
	// Open a websocket connection to Discord and begin listening
	err := h.discordSDKClient.Open()
	if err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
	}

	log.Printf("🤖 Discord bot is now running and listening for events")
	return nil
}

// StopBot gracefully closes the Discord connection
func (h *DiscordEventsHandler) StopBot() {
	h.discordSDKClient.Close()
}

// handleMessageCreatedEvent handles incoming Discord messages
func (h *DiscordEventsHandler) handleMessageCreatedEvent(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Printf("📨 Discord message received from %s in guild %s, channel %s",
		m.Author.Username, m.GuildID, m.ChannelID)

	ctx := context.Background()
	guildID := m.GuildID

	log.Printf("📨 Processing Discord message in guild %s", guildID)
	maybeDiscordInt, err := h.discordIntegrationsService.GetDiscordIntegrationByGuildID(ctx, guildID)
	if err != nil {
		log.Printf("❌ Failed to find Discord integration for guild %s: %v", guildID, err)
		return
	}
	if !maybeDiscordInt.IsPresent() {
		log.Printf("❌ Discord integration not found for guild %s - ignoring message", guildID)
		return // Don't treat this as an error, just ignore
	}
	discordIntegration := maybeDiscordInt.MustGet()

	// Map Discord SDK event to our model
	messageEvent, err := h.mapToDiscordMessageEvent(s, m)
	if err != nil {
		log.Printf("❌ Failed to map Discord message event: %v", err)
		return
	}

	// Track the channel in connected_channels table
	_, err = h.connectedChannelsService.UpsertDiscordConnectedChannel(ctx, discordIntegration.OrgID, guildID, messageEvent.ChannelID)
	if err != nil {
		log.Printf("⚠️ Failed to track Discord channel %s: %v", messageEvent.ChannelID, err)
		// Continue processing even if channel tracking fails
	}

	log.Printf("🔑 Found Discord integration for guild %s (ID: %s)", guildID, discordIntegration.ID)
	err = h.discordUseCase.ProcessDiscordMessageEvent(
		ctx,
		messageEvent,
		discordIntegration.ID,
		discordIntegration.OrgID,
	)
	if err != nil {
		log.Printf("❌ Failed to process Discord message: %v", err)
	}
}

// handleReactionAddedEvent handles when a reaction is added to a message
func (h *DiscordEventsHandler) handleReactionAddedEvent(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	log.Printf("🤖 Discord reaction %s added by user %s on message %s in guild %s",
		r.Emoji.Name, r.UserID, r.MessageID, r.GuildID)

	ctx := context.Background()
	guildID := r.GuildID

	log.Printf("📨 Processing Discord reaction %s in guild %s", r.Emoji.Name, guildID)
	maybeDiscordInt, err := h.discordIntegrationsService.GetDiscordIntegrationByGuildID(ctx, guildID)
	if err != nil {
		log.Printf("❌ Failed to find Discord integration for guild %s: %v", guildID, err)
		return
	}
	if !maybeDiscordInt.IsPresent() {
		log.Printf("❌ Discord integration not found for guild %s - ignoring reaction", guildID)
		return // Don't treat this as an error, just ignore
	}
	discordIntegration := maybeDiscordInt.MustGet()

	// Map Discord SDK event to our model
	reactionEvent, err := h.mapToDiscordReactionEvent(s, r)
	if err != nil {
		log.Printf("❌ Failed to map Discord reaction event: %v", err)
		return
	}

	// Track the channel in connected_channels table
	_, err = h.connectedChannelsService.UpsertDiscordConnectedChannel(ctx, discordIntegration.OrgID, guildID, reactionEvent.ChannelID)
	if err != nil {
		log.Printf("⚠️ Failed to track Discord channel %s: %v", reactionEvent.ChannelID, err)
		// Continue processing even if channel tracking fails
	}

	log.Printf("🔑 Found Discord integration for guild %s (ID: %s)", guildID, discordIntegration.ID)
	err = h.discordUseCase.ProcessDiscordReactionEvent(
		ctx,
		reactionEvent,
		discordIntegration.ID,
		discordIntegration.OrgID,
	)
	if err != nil {
		log.Printf("❌ Failed to process Discord reaction: %v", err)
	}
}

// mapToDiscordMessageEvent maps a Discord SDK message event to our domain model
func (h *DiscordEventsHandler) mapToDiscordMessageEvent(
	s *discordgo.Session,
	m *discordgo.MessageCreate,
) (models.DiscordMessageEvent, error) {
	// Get channel information to determine if this is a thread
	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		return models.DiscordMessageEvent{}, fmt.Errorf("failed to get channel info: %w", err)
	}

	var threadID *string
	if isThreadChannel(channel.Type) {
		threadID = &m.ChannelID
	}

	// Extract mentioned user IDs
	mentions := make([]string, len(m.Mentions))
	for i, mentionedUser := range m.Mentions {
		mentions[i] = mentionedUser.ID
	}

	return models.DiscordMessageEvent{
		GuildID:   m.GuildID,
		ChannelID: m.ChannelID,
		MessageID: m.ID,
		UserID:    m.Author.ID,
		Content:   m.Content,
		ThreadID:  threadID,
		Mentions:  mentions,
	}, nil
}

// mapToDiscordReactionEvent maps a Discord SDK reaction event to our domain model
func (h *DiscordEventsHandler) mapToDiscordReactionEvent(
	s *discordgo.Session,
	r *discordgo.MessageReactionAdd,
) (models.DiscordReactionEvent, error) {
	// Get channel information to determine if this is a thread
	channel, err := s.Channel(r.ChannelID)
	if err != nil {
		return models.DiscordReactionEvent{}, fmt.Errorf("failed to get channel info: %w", err)
	}

	var threadID *string
	if isThreadChannel(channel.Type) {
		threadID = &r.ChannelID
	}

	return models.DiscordReactionEvent{
		GuildID:   r.GuildID,
		ChannelID: r.ChannelID,
		MessageID: r.MessageID,
		UserID:    r.UserID,
		EmojiName: r.Emoji.Name,
		ThreadID:  threadID,
	}, nil
}

// isThreadChannel checks if the given channel type is a thread
func isThreadChannel(channelType discordgo.ChannelType) bool {
	return channelType == discordgo.ChannelTypeGuildPublicThread ||
		channelType == discordgo.ChannelTypeGuildPrivateThread ||
		channelType == discordgo.ChannelTypeGuildNewsThread
}
