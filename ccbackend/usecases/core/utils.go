package core

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

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
	organizationID string,
) ([]agentWithLoad, error) {
	agentsWithLoad := make([]agentWithLoad, 0, len(agents))

	for _, agent := range agents {
		// Get job IDs assigned to this agent
		jobIDs, err := s.agentsService.GetActiveAgentJobAssignments(ctx, agent.ID, organizationID)
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

// IsAgentErrorMessage determines if a system message from ccagent indicates an error or failure
func IsAgentErrorMessage(message string) bool {
	errorPatterns := []string{
		"error:",
		"failed to",
		"failure:",
		"unable to",
		"could not",
		"cannot ",
		"timeout",
		"timed out",
		"disconnected",
		"connection lost",
		"connection failed",
		"connection error",
		"exceeded max",
		"limit exceeded",
		"prompt is too long",
		"too many tokens",
		"rate limit",
		"permission denied",
		"access denied",
		"unauthorized",
		"invalid credentials",
		"authentication failed",
		"fatal:",
		"panic:",
		"crashed",
		"aborted",
		"terminated",
		"killed",
		"not found",
		"doesn't exist",
		"does not exist",
		"no such",
		"missing required",
		"invalid configuration",
		"bad request",
		"internal server error",
		"service unavailable",
		"bad gateway",
		"network error",
		"dns error",
		"resolution failed",
		"host not found",
		"connection refused",
		"broken pipe",
		"context canceled",
		"context deadline exceeded",
		"operation timed out",
		"request canceled",
		"unexpected end",
		"unexpected error",
		"unknown error",
		"critical error",
		"exception:",
		"stack trace:",
		"runtime error",
		"segmentation fault",
		"out of memory",
		"disk full",
		"quota exceeded",
		"resource exhausted",
	}

	// Convert message to lowercase for case-insensitive matching
	lowerMessage := strings.ToLower(message)

	// Check if message contains any error pattern
	for _, pattern := range errorPatterns {
		if strings.Contains(lowerMessage, pattern) {
			return true
		}
	}

	return false
}
