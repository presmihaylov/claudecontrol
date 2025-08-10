package discord

import (
	"context"
	"log"
	"slices"

	"ccbackend/models"
)

func (d *DiscordUseCase) ProcessDiscordMessageEvent(
	ctx context.Context,
	event models.DiscordMessageEvent,
	discordIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to process Discord message event from user %s in guild %s, channel %s",
		event.UserID, event.GuildID, event.ChannelID)

	// Step 1: Get bot user information
	botUser, err := d.discordClient.GetBotUser()
	if err != nil {
		log.Printf("❌ Failed to get bot user: %v", err)
		return err
	}
	log.Printf("🤖 Bot user retrieved: %s (%s)", botUser.Username, botUser.ID)

	// Step 2: Check if bot was mentioned
	botMentioned := slices.Contains(event.Mentions, botUser.ID)

	if !botMentioned {
		log.Printf("🔍 Bot not mentioned in message from user %s - ignoring message", event.UserID)
		log.Printf("📋 Message mentions (%d): %v", len(event.Mentions), event.Mentions)
		return nil
	}
	log.Printf("🤖 Bot %s (%s) mentioned in message from user %s", botUser.Username, botUser.ID, event.UserID)

	// Step 3: Determine channel type based on ThreadID
	var channelType string
	if event.ThreadID != nil {
		channelType = "thread"
		log.Printf("📨 Message received in Discord thread:")
		log.Printf("   Thread ID: %s", *event.ThreadID)
		log.Printf("   Channel ID: %s", event.ChannelID)
	} else {
		channelType = "top-level-channel"
		log.Printf("📨 Message received in Discord top-level channel:")
		log.Printf("   Channel ID: %s", event.ChannelID)
	}

	// Step 4: Log message processing summary
	log.Printf("🔍 Discord message processing summary:")
	log.Printf("   Guild: %s", event.GuildID)
	log.Printf("   Channel: %s (%s)", event.ChannelID, channelType)
	log.Printf("   Message ID: %s", event.MessageID)
	log.Printf("   User ID: %s", event.UserID)
	log.Printf("   Content Length: %d characters", len(event.Content))
	log.Printf("   Bot Mentioned: %t (Bot ID: %s)", botMentioned, botUser.ID)
	log.Printf("   Mentions (%d): %v", len(event.Mentions), event.Mentions)
	if event.ThreadID != nil {
		log.Printf("   Thread ID: %s", *event.ThreadID)
	} else {
		log.Printf("   Thread ID: <nil>")
	}

	// TODO: Implement Discord job creation and processing logic similar to Slack
	// This would include:
	// - Bot validation (check if message is from bot - needs to be added to model)
	// - Thread vs top-level message handling for job management
	// - Job creation/assignment logic
	// - Agent assignment and message forwarding

	log.Printf("🔄 Processing completed for Discord message event:")
	log.Printf("   Guild: %s, Channel: %s, Message: %s", event.GuildID, event.ChannelID, event.MessageID)

	log.Printf("📋 Completed successfully - processed Discord message event from user %s", event.UserID)
	return nil
}

func (d *DiscordUseCase) ProcessDiscordReactionEvent(
	ctx context.Context,
	event models.DiscordReactionEvent,
	discordIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to process Discord reaction event: %s by user %s on message %s in guild %s, channel %s",
		event.EmojiName, event.UserID, event.MessageID, event.GuildID, event.ChannelID)

	// Step 1: Check if this is a completion reaction (similar to Slack white_check_mark)
	if event.EmojiName != "✅" && event.EmojiName != "white_check_mark" && event.EmojiName != "heavy_check_mark" {
		log.Printf("⏭️ Ignoring reaction: %s (not a completion emoji)", event.EmojiName)
		return nil
	}

	log.Printf("✅ Completion reaction detected: %s by user %s on message %s",
		event.EmojiName, event.UserID, event.MessageID)

	// Step 2: Determine channel type based on ThreadID
	var channelType string
	if event.ThreadID != nil {
		channelType = "thread"
		log.Printf("📨 Reaction received in Discord thread:")
		log.Printf("   Thread ID: %s", *event.ThreadID)
		log.Printf("   Channel ID: %s", event.ChannelID)
	} else {
		channelType = "top-level-channel"
		log.Printf("📨 Reaction received in Discord top-level channel:")
		log.Printf("   Channel ID: %s", event.ChannelID)
	}

	// Step 3: Log reaction processing summary
	log.Printf("🔍 Discord reaction processing summary:")
	log.Printf("   Guild: %s", event.GuildID)
	log.Printf("   Channel: %s (%s)", event.ChannelID, channelType)
	log.Printf("   Message ID: %s", event.MessageID)
	log.Printf("   User ID: %s", event.UserID)
	log.Printf("   Emoji: %s", event.EmojiName)
	if event.ThreadID != nil {
		log.Printf("   Thread ID: %s", *event.ThreadID)
	} else {
		log.Printf("   Thread ID: <nil>")
	}

	// TODO: Implement Discord job completion logic similar to Slack
	// This would include:
	// - Finding the job associated with this Discord thread/message
	// - Validating the user who added the reaction is the job creator
	// - Unassigning the agent and deleting the job
	// - Updating Discord message reactions
	// - Sending completion message to Discord thread

	log.Printf("🔄 Processing completed for Discord reaction event:")
	log.Printf("   Emoji: %s by user %s on message %s", event.EmojiName, event.UserID, event.MessageID)

	log.Printf("📋 Completed successfully - processed Discord reaction event from user %s", event.UserID)
	return nil
}

