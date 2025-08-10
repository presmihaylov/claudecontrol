package slack

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/samber/mo"

	"ccbackend/clients"
	"ccbackend/core"
	"ccbackend/models"
	"ccbackend/utils"
)

func (s *SlackUseCase) getSlackClientForIntegration(
	ctx context.Context,
	slackIntegrationID string,
) (clients.SlackClient, error) {
	maybeSlackInt, err := s.slackIntegrationsService.GetSlackIntegrationByID(ctx, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integration: %w", err)
	}
	if !maybeSlackInt.IsPresent() {
		return nil, fmt.Errorf("slack integration not found: %s", slackIntegrationID)
	}
	integration := maybeSlackInt.MustGet()

	return s.slackClientFactory(integration.SlackAuthToken), nil
}

func (s *SlackUseCase) sendStartConversationToAgent(
	ctx context.Context,
	clientID string,
	message *models.ProcessedSlackMessage,
) error {
	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(ctx, message.SlackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Get job to access thread timestamp
	maybeJob, err := s.jobsService.GetJobByID(ctx, message.JobID, message.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		return fmt.Errorf("job not found: %s", message.JobID)
	}
	job := maybeJob.MustGet()

	// Generate permalink for the thread's first message
	if job.SlackPayload == nil {
		return fmt.Errorf("job has no Slack payload")
	}
	permalink, err := slackClient.GetPermalink(&clients.SlackPermalinkParameters{
		Channel: message.SlackChannelID,
		TS:      job.SlackPayload.ThreadTS,
	})
	if err != nil {
		return fmt.Errorf("failed to get permalink for slack message: %w", err)
	}

	// Resolve user mentions in the message text before sending to agent
	resolvedText := slackClient.ResolveMentionsInMessage(ctx, message.TextContent)
	startConversationMessage := models.BaseMessage{
		ID:   core.NewID("msg"),
		Type: models.MessageTypeStartConversation,
		Payload: models.StartConversationPayload{
			JobID:              message.JobID,
			Message:            resolvedText,
			ProcessedMessageID: message.ID,
			MessageLink:        permalink,
		},
	}

	if err := s.wsClient.SendMessage(clientID, startConversationMessage); err != nil {
		return fmt.Errorf("failed to send start conversation message to client %s: %v", clientID, err)
	}
	log.Printf("🚀 Sent start conversation message to client %s", clientID)
	return nil
}

func (s *SlackUseCase) sendUserMessageToAgent(
	ctx context.Context,
	clientID string,
	message *models.ProcessedSlackMessage,
) error {
	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(ctx, message.SlackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Get job to access thread timestamp
	maybeJob, err := s.jobsService.GetJobByID(ctx, message.JobID, message.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		return fmt.Errorf("job not found: %s", message.JobID)
	}
	job := maybeJob.MustGet()

	// Generate permalink for the thread's first message
	if job.SlackPayload == nil {
		return fmt.Errorf("job has no Slack payload")
	}
	permalink, err := slackClient.GetPermalink(&clients.SlackPermalinkParameters{
		Channel: message.SlackChannelID,
		TS:      job.SlackPayload.ThreadTS,
	})
	if err != nil {
		return fmt.Errorf("failed to get permalink for slack message: %w", err)
	}

	// Resolve user mentions in the message text before sending to agent
	resolvedText := slackClient.ResolveMentionsInMessage(ctx, message.TextContent)
	userMessage := models.BaseMessage{
		ID:   core.NewID("msg"),
		Type: models.MessageTypeUserMessage,
		Payload: models.UserMessagePayload{
			JobID:              message.JobID,
			Message:            resolvedText,
			ProcessedMessageID: message.ID,
			MessageLink:        permalink,
		},
	}

	if err := s.wsClient.SendMessage(clientID, userMessage); err != nil {
		return fmt.Errorf("failed to send user message to client %s: %v", clientID, err)
	}
	log.Printf("💬 Sent user message to client %s", clientID)
	return nil
}

func (s *SlackUseCase) updateSlackMessageReaction(
	ctx context.Context,
	channelID, messageTS, newEmoji, slackIntegrationID string,
) error {
	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(ctx, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Get only reactions added by our bot
	botReactions, err := s.getBotReactionsOnMessage(ctx, channelID, messageTS, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get bot reactions: %w", err)
	}

	// Only remove reactions that are incompatible with the new state
	reactionsToRemove := getOldReactions(newEmoji)
	for _, emoji := range reactionsToRemove {
		if slices.Contains(botReactions, emoji) {
			if err := slackClient.RemoveReaction(emoji, clients.SlackItemRef{
				Channel:   channelID,
				Timestamp: messageTS,
			}); err != nil {
				return fmt.Errorf("failed to remove %s reaction: %w", emoji, err)
			}
		}
	}

	// Add new reaction if not already there
	if newEmoji != "" && !slices.Contains(botReactions, newEmoji) {
		if err := slackClient.AddReaction(newEmoji, clients.SlackItemRef{
			Channel:   channelID,
			Timestamp: messageTS,
		}); err != nil {
			return fmt.Errorf("failed to add %s reaction: %w", newEmoji, err)
		}
	}

	return nil
}

func (s *SlackUseCase) sendSlackMessage(
	ctx context.Context,
	slackIntegrationID, channelID, threadTS, message string,
) error {
	log.Printf("📋 Starting to send message to channel %s, thread %s: %s", channelID, threadTS, message)

	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(ctx, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Send message to Slack
	params := clients.SlackMessageParams{
		Text: utils.ConvertMarkdownToSlack(message),
	}
	if threadTS != "" {
		params.ThreadTS = mo.Some(threadTS)
	}
	_, err = slackClient.PostMessage(channelID, params)
	if err != nil {
		return fmt.Errorf("failed to send message to Slack: %w", err)
	}

	log.Printf("📋 Completed successfully - sent message to channel %s, thread %s", channelID, threadTS)
	return nil
}

func (s *SlackUseCase) sendSystemMessage(
	ctx context.Context,
	slackIntegrationID, channelID, threadTS, message string,
) error {
	log.Printf("📋 Starting to send system message to channel %s, thread %s: %s", channelID, threadTS, message)

	// Prepend gear emoji to message
	systemMessage := ":gear: " + message

	// Use the base sendSlackMessage function
	return s.sendSlackMessage(ctx, slackIntegrationID, channelID, threadTS, systemMessage)
}

func (s *SlackUseCase) getBotUserID(ctx context.Context, slackIntegrationID string) (string, error) {
	slackClient, err := s.getSlackClientForIntegration(ctx, slackIntegrationID)
	if err != nil {
		return "", fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	authTest, err := slackClient.AuthTest()
	if err != nil {
		return "", fmt.Errorf("failed to get bot user ID: %w", err)
	}
	return authTest.UserID, nil
}

func (s *SlackUseCase) getBotReactionsOnMessage(
	ctx context.Context,
	channelID, messageTS string,
	slackIntegrationID string,
) ([]string, error) {
	slackClient, err := s.getSlackClientForIntegration(ctx, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	botUserID, err := s.getBotUserID(ctx, slackIntegrationID)
	if err != nil {
		return nil, err
	}

	// Get reactions directly using GetReactions - much less rate limited
	reactions, err := slackClient.GetReactions(clients.SlackItemRef{
		Channel:   channelID,
		Timestamp: messageTS,
	}, clients.SlackGetReactionsParameters{})
	if err != nil {
		return nil, fmt.Errorf("failed to get reactions: %w", err)
	}

	var botReactions []string
	for _, reaction := range reactions {
		// Check if bot added this reaction
		if slices.Contains(reaction.Users, botUserID) {
			botReactions = append(botReactions, reaction.Name)
		}
	}

	return botReactions, nil
}

func deriveMessageReactionFromStatus(status models.ProcessedSlackMessageStatus) string {
	switch status {
	case models.ProcessedSlackMessageStatusInProgress:
		return "hourglass"
	case models.ProcessedSlackMessageStatusQueued:
		return "hourglass"
	case models.ProcessedSlackMessageStatusCompleted:
		return "white_check_mark"
	default:
		utils.AssertInvariant(false, "invalid status received")
		return ""
	}
}

func getOldReactions(newEmoji string) []string {
	allReactions := []string{"hourglass", "eyes", "white_check_mark", "hand", "x"}

	var result []string
	for _, reaction := range allReactions {
		if reaction != newEmoji {
			result = append(result, reaction)
		}
	}

	return result
}

// isAgentErrorMessage determines if a system message from ccagent indicates an error or failure
func isAgentErrorMessage(message string) bool {
	// Check if message starts with the specific error prefix from ccagent
	return strings.HasPrefix(message, "ccagent encountered error:")
}
