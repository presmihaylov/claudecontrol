package core

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"

	"ccbackend/clients"
	"ccbackend/models"
	agentsmocks "ccbackend/services/agents"
	jobsmocks "ccbackend/services/jobs"
	organizationsmocks "ccbackend/services/organizations"
	slackintegrationsmocks "ccbackend/services/slack_integrations"
)

func TestValidateAPIKey(t *testing.T) {
	t.Run("valid_api_key", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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

	t.Run("service_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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

		// Configure expectations
		mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey).
			Return(mo.None[*models.Organization](), fmt.Errorf("database connection error"))

		// Execute
		orgID, err := useCase.ValidateAPIKey(ctx, apiKey)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")
		assert.Equal(t, "", orgID)
		mockOrganizationsService.AssertExpectations(t)
	})

	t.Run("empty_api_key", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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

		apiKey := ""

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

	t.Run("concurrent_validation", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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

		apiKey1 := "test-api-key-1"
		apiKey2 := "test-api-key-2"
		organization1 := &models.Organization{
			ID:               "org-1",
			CCAgentSecretKey: &apiKey1,
		}
		organization2 := &models.Organization{
			ID:               "org-2",
			CCAgentSecretKey: &apiKey2,
		}

		// Configure expectations
		mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey1).
			Return(mo.Some(organization1), nil)
		mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey2).
			Return(mo.Some(organization2), nil)

		// Execute concurrently
		done := make(chan bool, 2)
		var orgID1, orgID2 string
		var err1, err2 error

		go func() {
			orgID1, err1 = useCase.ValidateAPIKey(ctx, apiKey1)
			done <- true
		}()

		go func() {
			orgID2, err2 = useCase.ValidateAPIKey(ctx, apiKey2)
			done <- true
		}()

		// Wait for both to complete
		<-done
		<-done

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, "org-1", orgID1)
		assert.Equal(t, "org-2", orgID2)
		mockOrganizationsService.AssertExpectations(t)
	})
}