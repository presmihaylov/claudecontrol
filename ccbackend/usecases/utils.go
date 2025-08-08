package usecases

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/utils"
)

type agentWithLoad struct {
	agent *models.ActiveAgent
	load  int
}

func (s *CoreUseCase) sortAgentsByLoad(
	ctx context.Context,
	agents []*models.ActiveAgent,
	slackIntegrationID string,
) ([]agentWithLoad, error) {
	agentsWithLoad := make([]agentWithLoad, 0, len(agents))

	for _, agent := range agents {
		// Get job IDs assigned to this agent
		jobIDs, err := s.agentsService.GetActiveAgentJobAssignments(ctx, agent.ID, slackIntegrationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get job assignments for agent %s: %w", agent.ID, err)
		}

		jobCount := len(jobIDs)

		agentsWithLoad = append(agentsWithLoad, agentWithLoad{agent: agent, load: jobCount})
	}

	// Sort by load (ascending - least loaded first)
	sort.Slice(agentsWithLoad, func(i, j int) bool {
		return agentsWithLoad[i].load < agentsWithLoad[j].load
	})

	return agentsWithLoad, nil
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

func (s *CoreUseCase) getBotUserID(ctx context.Context, slackIntegrationID string) (string, error) {
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

func (s *CoreUseCase) getBotReactionsOnMessage(
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

func DeriveMessageReactionFromStatus(status models.ProcessedSlackMessageStatus) string {
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