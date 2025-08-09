package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ccbackend/usecases/agents"
	"ccbackend/usecases/slack"
)

func TestNewCoreUseCase(t *testing.T) {
	// Test that NewCoreUseCase properly initializes the struct
	mockAgentsService := new(MockAgentsService)
	mockWSClient := new(MockSocketIOClient)
	mockJobsService := new(MockJobsService)
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	mockOrganizationsService := new(MockOrganizationsService)
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