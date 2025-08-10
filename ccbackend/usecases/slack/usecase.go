package slack

import (
	"ccbackend/clients"
	"ccbackend/services"
	"ccbackend/usecases/agents"
)

// SlackUseCase handles all Slack-specific operations
type SlackUseCase struct {
	wsClient                 clients.SocketIOClient
	agentsService            services.AgentsService
	jobsService              services.JobsService
	slackIntegrationsService services.SlackIntegrationsService
	txManager                services.TransactionManager
	agentsUseCase            agents.AgentsUseCaseInterface
}

// NewSlackUseCase creates a new instance of SlackUseCase
func NewSlackUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	slackIntegrationsService services.SlackIntegrationsService,
	txManager services.TransactionManager,
	agentsUseCase agents.AgentsUseCaseInterface,
) *SlackUseCase {
	return &SlackUseCase{
		wsClient:                 wsClient,
		agentsService:            agentsService,
		jobsService:              jobsService,
		slackIntegrationsService: slackIntegrationsService,
		txManager:                txManager,
		agentsUseCase:            agentsUseCase,
	}
}
