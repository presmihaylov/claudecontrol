package core

import (
	"context"

	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases/agents"
	"ccbackend/usecases/slack"
)

// CoreUseCase orchestrates all core business operations
type CoreUseCase struct {
	wsClient                 clients.SocketIOClient
	agentsService            services.AgentsService
	jobsService              services.JobsService
	slackIntegrationsService services.SlackIntegrationsService
	organizationsService     services.OrganizationsService

	// Use case dependencies
	agentsUseCase *agents.AgentsUseCase
	slackUseCase  *slack.SlackUseCase
}

// NewCoreUseCase creates a new instance of CoreUseCase
func NewCoreUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	slackIntegrationsService services.SlackIntegrationsService,
	organizationsService services.OrganizationsService,
	agentsUseCase *agents.AgentsUseCase,
	slackUseCase *slack.SlackUseCase,
) *CoreUseCase {
	return &CoreUseCase{
		wsClient:                 wsClient,
		agentsService:            agentsService,
		jobsService:              jobsService,
		slackIntegrationsService: slackIntegrationsService,
		organizationsService:     organizationsService,
		agentsUseCase:            agentsUseCase,
		slackUseCase:             slackUseCase,
	}
}

// Proxy methods to SlackUseCase

// ProcessSlackMessageEvent proxies to SlackUseCase
func (s *CoreUseCase) ProcessSlackMessageEvent(
	ctx context.Context,
	event models.SlackMessageEvent,
	slackIntegrationID string,
	organizationID string,
) error {
	return s.slackUseCase.ProcessSlackMessageEvent(ctx, event, slackIntegrationID, organizationID)
}

// ProcessReactionAdded proxies to SlackUseCase
func (s *CoreUseCase) ProcessReactionAdded(
	ctx context.Context,
	reactionName, userID, channelID, messageTS, slackIntegrationID string,
	organizationID string,
) error {
	return s.slackUseCase.ProcessReactionAdded(
		ctx,
		reactionName,
		userID,
		channelID,
		messageTS,
		slackIntegrationID,
		organizationID,
	)
}

// ProcessProcessingSlackMessage proxies to SlackUseCase
func (s *CoreUseCase) ProcessProcessingSlackMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingSlackMessagePayload,
	organizationID string,
) error {
	return s.slackUseCase.ProcessProcessingSlackMessage(ctx, clientID, payload, organizationID)
}

// ProcessAssistantMessage proxies to SlackUseCase
func (s *CoreUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	organizationID string,
) error {
	return s.slackUseCase.ProcessAssistantMessage(ctx, clientID, payload, organizationID)
}

// ProcessSystemMessage proxies to SlackUseCase
func (s *CoreUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	organizationID string,
) error {
	return s.slackUseCase.ProcessSystemMessage(ctx, clientID, payload, organizationID)
}

// ProcessJobComplete proxies to SlackUseCase
func (s *CoreUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	organizationID string,
) error {
	return s.slackUseCase.ProcessJobComplete(ctx, clientID, payload, organizationID)
}

// ProcessQueuedJobs proxies to SlackUseCase
func (s *CoreUseCase) ProcessQueuedJobs(ctx context.Context) error {
	return s.slackUseCase.ProcessQueuedJobs(ctx)
}
