package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ccbackend/clients"
	"ccbackend/services"
	"ccbackend/usecases/agents"
	"ccbackend/usecases/slack"
)

func TestNewCoreUseCase(t *testing.T) {
	// Test that NewCoreUseCase properly initializes the struct
	mockAgentsService := new(services.MockAgentsService)
	mockWSClient := new(clients.MockSocketIOClient)
	mockJobsService := new(services.MockJobsService)
	mockSlackIntegrationsService := new(services.MockSlackIntegrationsService)
	mockOrganizationsService := new(services.MockOrganizationsService)
	mockAgentsUseCase := &agents.AgentsUseCase{}
	mockSlackUseCase := &slack.SlackUseCase{}

	useCase := NewCoreUseCase(
		mockWSClient,
		mockAgentsService,
		mockJobsService,
		mockSlackIntegrationsService,
		mockOrganizationsService,
		mockAgentsUseCase,
		mockSlackUseCase,
	)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockWSClient, useCase.wsClient)
	assert.Equal(t, mockAgentsService, useCase.agentsService)
	assert.Equal(t, mockJobsService, useCase.jobsService)
	assert.Equal(t, mockSlackIntegrationsService, useCase.slackIntegrationsService)
	assert.Equal(t, mockOrganizationsService, useCase.organizationsService)
	assert.Equal(t, mockAgentsUseCase, useCase.agentsUseCase)
	assert.Equal(t, mockSlackUseCase, useCase.slackUseCase)
}
