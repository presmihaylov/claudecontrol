package core

import (
	"ccbackend/clients"
	"ccbackend/services"
)

// CoreUseCase orchestrates all core business operations
type CoreUseCase struct {
	wsClient                 clients.SocketIOClient
	agentsService            services.AgentsService
	jobsService              services.JobsService
	slackIntegrationsService services.SlackIntegrationsService
	organizationsService     services.OrganizationsService
	txManager                services.TransactionManager
}

// NewCoreUseCase creates a new instance of CoreUseCase
func NewCoreUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	slackIntegrationsService services.SlackIntegrationsService,
	organizationsService services.OrganizationsService,
	txManager services.TransactionManager,
) *CoreUseCase {
	return &CoreUseCase{
		wsClient:                 wsClient,
		agentsService:            agentsService,
		jobsService:              jobsService,
		slackIntegrationsService: slackIntegrationsService,
		organizationsService:     organizationsService,
		txManager:                txManager,
	}
}
