package commands

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"ccbackend/appctx"
	"ccbackend/models"
	"ccbackend/services"
)

type CommandsService struct {
	agentsService            services.AgentsService
	connectedChannelsService services.ConnectedChannelsService
}

func NewCommandsService(
	agentsService services.AgentsService,
	connectedChannelsService services.ConnectedChannelsService,
) *CommandsService {
	return &CommandsService{
		agentsService:            agentsService,
		connectedChannelsService: connectedChannelsService,
	}
}

func (s *CommandsService) ProcessCommand(
	ctx context.Context,
	request models.CommandRequest,
) (*models.CommandResult, error) {
	log.Printf("📋 Starting to process command: %s from platform: %s", request.Command, request.Platform)

	// Get organization from context
	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return nil, fmt.Errorf("organization not found in context")
	}

	// Parse the command
	commandType, repoURL, err := s.parseCommand(request.Command)
	if err != nil {
		log.Printf("❌ Failed to parse command: %v", err)
		return &models.CommandResult{
			Success: false,
			Message: fmt.Sprintf("Invalid command format. Use: `--cmd repo=<repository_url>`"),
		}, nil
	}

	switch commandType {
	case "repo":
		return s.processRepoCommand(ctx, models.OrgID(org.ID), request.Platform, request.TeamID, request.ChannelID, repoURL)
	default:
		return &models.CommandResult{
			Success: false,
			Message: fmt.Sprintf("Unknown command: %s", commandType),
		}, nil
	}
}

func (s *CommandsService) parseCommand(command string) (commandType string, value string, err error) {
	log.Printf("📋 Starting to parse command: %s", command)

	// Remove any leading/trailing whitespace
	command = strings.TrimSpace(command)

	// Check if command starts with --cmd
	if !strings.HasPrefix(command, "--cmd") {
		return "", "", fmt.Errorf("command must start with --cmd")
	}

	// Remove --cmd prefix and any following whitespace
	command = strings.TrimSpace(strings.TrimPrefix(command, "--cmd"))

	// Parse command format: key=value
	parts := strings.SplitN(command, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("command must be in format: key=value")
	}

	commandType = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])

	if commandType == "" || value == "" {
		return "", "", fmt.Errorf("command and value cannot be empty")
	}

	log.Printf("📋 Completed successfully - parsed command type: %s, value: %s", commandType, value)
	return commandType, value, nil
}

func (s *CommandsService) processRepoCommand(
	ctx context.Context,
	orgID models.OrgID,
	platform models.ChannelType,
	teamID string,
	channelID string,
	repoURL string,
) (*models.CommandResult, error) {
	log.Printf("📋 Starting to process repo command for org: %s, platform: %s, channel: %s, repo: %s", orgID, platform, channelID, repoURL)

	// Normalize the repository URL
	normalizedRepoURL, err := s.normalizeRepoURL(repoURL)
	if err != nil {
		log.Printf("❌ Failed to normalize repo URL: %v", err)
		return &models.CommandResult{
			Success: false,
			Message: fmt.Sprintf("Invalid repository URL format: %s", repoURL),
		}, nil
	}

	// Validate that the repository exists in active agents
	exists, err := s.validateRepoExistsInActiveAgents(ctx, orgID, normalizedRepoURL)
	if err != nil {
		log.Printf("❌ Failed to validate repo exists in active agents: %v", err)
		return nil, fmt.Errorf("failed to validate repository: %w", err)
	}

	if !exists {
		log.Printf("❌ Repository %s not found in active agents for org: %s", normalizedRepoURL, orgID)
		return &models.CommandResult{
			Success: false,
			Message: fmt.Sprintf("Repository %s not found in active agents for this organization", normalizedRepoURL),
		}, nil
	}

	// Update the connected channel's default repository
	err = s.updateChannelDefaultRepo(ctx, orgID, platform, teamID, channelID, normalizedRepoURL)
	if err != nil {
		log.Printf("❌ Failed to update channel default repo: %v", err)
		return nil, fmt.Errorf("failed to update channel repository: %w", err)
	}

	log.Printf("📋 Completed successfully - updated channel default repo to: %s", normalizedRepoURL)
	return &models.CommandResult{
		Success: true,
		Message: fmt.Sprintf("✅ Repository set to %s", normalizedRepoURL),
	}, nil
}

func (s *CommandsService) normalizeRepoURL(repoURL string) (string, error) {
	log.Printf("📋 Starting to normalize repo URL: %s", repoURL)

	// Remove any surrounding angle brackets (from Slack links)
	repoURL = strings.Trim(repoURL, "<>")

	// Handle GitHub URLs in various formats
	githubRegex := regexp.MustCompile(`^(?:https?://)?(?:www\.)?github\.com/([^/]+/[^/]+)/?$`)
	matches := githubRegex.FindStringSubmatch(repoURL)
	if len(matches) == 2 {
		normalized := "github.com/" + matches[1]
		log.Printf("📋 Completed successfully - normalized GitHub URL to: %s", normalized)
		return normalized, nil
	}

	// Handle URLs that are already in normalized format
	if strings.HasPrefix(repoURL, "github.com/") {
		log.Printf("📋 Completed successfully - URL already normalized: %s", repoURL)
		return repoURL, nil
	}

	// Try to parse as a generic URL and extract path
	parsedURL, err := url.Parse(repoURL)
	if err == nil && parsedURL.Host == "github.com" {
		path := strings.Trim(parsedURL.Path, "/")
		if path != "" {
			normalized := "github.com/" + path
			log.Printf("📋 Completed successfully - normalized generic GitHub URL to: %s", normalized)
			return normalized, nil
		}
	}

	return "", fmt.Errorf("unsupported repository URL format")
}

func (s *CommandsService) validateRepoExistsInActiveAgents(
	ctx context.Context,
	orgID models.OrgID,
	repoURL string,
) (bool, error) {
	log.Printf("📋 Starting to validate repo exists in active agents: %s for org: %s", repoURL, orgID)

	// Get all active agents for the organization
	agents, err := s.agentsService.GetAvailableAgents(ctx, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to get active agents: %w", err)
	}

	// Check if any agent has the matching repository URL
	for _, agent := range agents {
		if agent.RepoURL == repoURL {
			log.Printf("📋 Completed successfully - found repo %s in agent: %s", repoURL, agent.ID)
			return true, nil
		}
	}

	log.Printf("📋 Completed successfully - repo %s not found in any active agents", repoURL)
	return false, nil
}

func (s *CommandsService) updateChannelDefaultRepo(
	ctx context.Context,
	orgID models.OrgID,
	platform models.ChannelType,
	teamID string,
	channelID string,
	repoURL string,
) error {
	log.Printf("📋 Starting to update channel default repo for platform: %s, channel: %s, repo: %s", platform, channelID, repoURL)

	switch platform {
	case models.ChannelTypeSlack:
		_, err := s.connectedChannelsService.UpdateSlackChannelDefaultRepo(ctx, orgID, teamID, channelID, repoURL)
		if err != nil {
			return fmt.Errorf("failed to update Slack channel default repo: %w", err)
		}
	case models.ChannelTypeDiscord:
		_, err := s.connectedChannelsService.UpdateDiscordChannelDefaultRepo(ctx, orgID, teamID, channelID, repoURL)
		if err != nil {
			return fmt.Errorf("failed to update Discord channel default repo: %w", err)
		}
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	log.Printf("📋 Completed successfully - updated channel default repo")
	return nil
}