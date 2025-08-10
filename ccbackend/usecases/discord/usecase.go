package discord

import (
	"ccbackend/clients"
	"ccbackend/services"
	"ccbackend/usecases/agents"
)

// DiscordUseCase handles all Discord-specific operations
type DiscordUseCase struct {
	wsClient                    clients.SocketIOClient
	agentsService               services.AgentsService
	jobsService                 services.JobsService
	discordMessagesService      services.DiscordMessagesService
	discordIntegrationsService  services.DiscordIntegrationsService
	txManager                   services.TransactionManager
	agentsUseCase               agents.AgentsUseCaseInterface
}

// NewDiscordUseCase creates a new instance of DiscordUseCase
func NewDiscordUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	discordMessagesService services.DiscordMessagesService,
	discordIntegrationsService services.DiscordIntegrationsService,
	txManager services.TransactionManager,
	agentsUseCase agents.AgentsUseCaseInterface,
) *DiscordUseCase {
	return &DiscordUseCase{
		wsClient:                   wsClient,
		agentsService:              agentsService,
		jobsService:                jobsService,
		discordMessagesService:     discordMessagesService,
		discordIntegrationsService: discordIntegrationsService,
		txManager:                  txManager,
		agentsUseCase:              agentsUseCase,
	}
}