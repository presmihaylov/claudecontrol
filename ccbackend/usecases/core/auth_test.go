package core

import (
	"context"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"

	"ccbackend/clients/socketio"
	"ccbackend/models"
	"ccbackend/services/agents"
	"ccbackend/services/jobs"
	"ccbackend/services/organizations"
	slackintegrations "ccbackend/services/slack_integrations"
)

func TestValidateAPIKey(t *testing.T) {
	t.Run("valid_api_key", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)
		// Pass nil for use cases that aren't used in this test
		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		apiKey := "test-api-key-123"
		organization := &models.Organization{
			ID:               "org-456",
			CCAgentSecretKey: &apiKey,
		}

		// Configure expectations
		mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey).
			Return(mo.Some(organization), nil)

		// Execute
		orgID, err := useCase.ValidateAPIKey(ctx, apiKey)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "org-456", orgID)
		mockOrganizationsService.AssertExpectations(t)
	})

	t.Run("invalid_api_key", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)
		// Pass nil for use cases that aren't used in this test
		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		apiKey := "invalid-api-key"

		// Configure expectations
		mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey).
			Return(mo.None[*models.Organization](), nil)

		// Execute
		orgID, err := useCase.ValidateAPIKey(ctx, apiKey)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
		assert.Equal(t, "", orgID)
		mockOrganizationsService.AssertExpectations(t)
	})
}
