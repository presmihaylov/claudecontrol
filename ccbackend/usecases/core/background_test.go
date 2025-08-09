package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ccbackend/models"
)

func TestBroadcastCheckIdleJobs_Success_NoOrganizations(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	mockWsClient := new(MockSocketIOClient)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
		wsClient:            mockWsClient,
	}

	mockOrganizationsService.On("GetAllOrganizations", ctx).Return([]*models.Organization{}, nil)

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	assert.NoError(t, err)
	mockOrganizationsService.AssertNumberOfCalls(t, "GetAllOrganizations", 1)
	// Should not call GetClientIDs when no organizations
	mockWsClient.AssertNotCalled(t, "GetClientIDs")
	mockWsClient.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything)
}

func TestBroadcastCheckIdleJobs_Success_NoConnectedAgents(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	mockAgentsService := new(MockAgentsService)
	mockWsClient := new(MockSocketIOClient)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
		agentsService:        mockAgentsService,
		wsClient:            mockWsClient,
	}

	organizations := []*models.Organization{
		{
			ID: "org_1",
		},
		{
			ID: "org_2",
		},
	}

	connectedClientIDs := []string{"client_1", "client_2"}

	mockOrganizationsService.On("GetAllOrganizations", ctx).Return(organizations, nil)
	mockWsClient.On("GetClientIDs").Return(connectedClientIDs)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs).Return([]*models.ActiveAgent{}, nil)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_2", connectedClientIDs).Return([]*models.ActiveAgent{}, nil)

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	assert.NoError(t, err)
	mockAgentsService.AssertNumberOfCalls(t, "GetConnectedActiveAgents", 2)
	mockWsClient.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything)
}

func TestBroadcastCheckIdleJobs_Success_WithConnectedAgents(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	mockAgentsService := new(MockAgentsService)
	mockWsClient := new(MockSocketIOClient)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
		agentsService:        mockAgentsService,
		wsClient:            mockWsClient,
	}

	organizations := []*models.Organization{
		{
			ID: "org_1",
		},
		{
			ID: "org_2",
		},
	}

	connectedClientIDs := []string{"client_1", "client_2", "client_3"}

	agents1 := []*models.ActiveAgent{
		{
			ID:             "aa_1",
			WSConnectionID: "client_1",
			OrganizationID: "org_1",
		},
		{
			ID:             "aa_2",
			WSConnectionID: "client_2",
			OrganizationID: "org_1",
		},
	}

	agents2 := []*models.ActiveAgent{
		{
			ID:             "aa_3",
			WSConnectionID: "client_3",
			OrganizationID: "org_2",
		},
	}

	mockOrganizationsService.On("GetAllOrganizations", ctx).Return(organizations, nil)
	mockWsClient.On("GetClientIDs").Return(connectedClientIDs)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs).Return(agents1, nil)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_2", connectedClientIDs).Return(agents2, nil)
	
	// Mock SendMessage for each agent
	mockWsClient.On("SendMessage", "client_1", mock.MatchedBy(func(msg models.BaseMessage) bool {
		return msg.Type == models.MessageTypeCheckIdleJobs && strings.HasPrefix(msg.ID, "msg_")
	})).Return(nil)
	mockWsClient.On("SendMessage", "client_2", mock.MatchedBy(func(msg models.BaseMessage) bool {
		return msg.Type == models.MessageTypeCheckIdleJobs && strings.HasPrefix(msg.ID, "msg_")
	})).Return(nil)
	mockWsClient.On("SendMessage", "client_3", mock.MatchedBy(func(msg models.BaseMessage) bool {
		return msg.Type == models.MessageTypeCheckIdleJobs && strings.HasPrefix(msg.ID, "msg_")
	})).Return(nil)

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	assert.NoError(t, err)
	mockWsClient.AssertNumberOfCalls(t, "SendMessage", 3)
	mockAgentsService.AssertCalled(t, "GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs)
	mockAgentsService.AssertCalled(t, "GetConnectedActiveAgents", ctx, "org_2", connectedClientIDs)
}

func TestBroadcastCheckIdleJobs_GetOrganizationsError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
	}

	serviceErr := errors.New("database error")
	mockOrganizationsService.On("GetAllOrganizations", ctx).Return([]*models.Organization(nil), serviceErr)

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get organizations")
	assert.Contains(t, err.Error(), "database error")
}

func TestBroadcastCheckIdleJobs_GetConnectedAgentsError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	mockAgentsService := new(MockAgentsService)
	mockWsClient := new(MockSocketIOClient)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
		agentsService:        mockAgentsService,
		wsClient:            mockWsClient,
	}

	organizations := []*models.Organization{
		{
			ID: "org_1",
		},
	}

	connectedClientIDs := []string{"client_1"}
	agentErr := errors.New("agents fetch error")

	mockOrganizationsService.On("GetAllOrganizations", ctx).Return(organizations, nil)
	mockWsClient.On("GetClientIDs").Return(connectedClientIDs)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs).Return([]*models.ActiveAgent(nil), agentErr)

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get connected agents for organization org_1")
	assert.Contains(t, err.Error(), "agents fetch error")
}

func TestBroadcastCheckIdleJobs_SendMessageError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	mockAgentsService := new(MockAgentsService)
	mockWsClient := new(MockSocketIOClient)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
		agentsService:        mockAgentsService,
		wsClient:            mockWsClient,
	}

	organizations := []*models.Organization{
		{
			ID: "org_1",
		},
	}

	connectedClientIDs := []string{"client_1"}

	agents := []*models.ActiveAgent{
		{
			ID:             "aa_1",
			WSConnectionID: "client_1",
			OrganizationID: "org_1",
		},
	}

	sendErr := errors.New("websocket send error")

	mockOrganizationsService.On("GetAllOrganizations", ctx).Return(organizations, nil)
	mockWsClient.On("GetClientIDs").Return(connectedClientIDs)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs).Return(agents, nil)
	mockWsClient.On("SendMessage", "client_1", mock.Anything).Return(sendErr)

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send CheckIdleJobs message to agent aa_1")
	assert.Contains(t, err.Error(), "websocket send error")
}

func TestBroadcastCheckIdleJobs_MultipleOrganizations(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	mockAgentsService := new(MockAgentsService)
	mockWsClient := new(MockSocketIOClient)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
		agentsService:        mockAgentsService,
		wsClient:            mockWsClient,
	}

	organizations := []*models.Organization{
		{
			ID: "org_1",
		},
		{
			ID: "org_2",
		},
		{
			ID: "org_3",
		},
	}

	connectedClientIDs := []string{"client_1", "client_2", "client_3", "client_4", "client_5"}

	agents1 := []*models.ActiveAgent{
		{
			ID:             "aa_1",
			WSConnectionID: "client_1",
			OrganizationID: "org_1",
		},
	}

	agents2 := []*models.ActiveAgent{
		{
			ID:             "aa_2",
			WSConnectionID: "client_2",
			OrganizationID: "org_2",
		},
		{
			ID:             "aa_3",
			WSConnectionID: "client_3",
			OrganizationID: "org_2",
		},
	}

	agents3 := []*models.ActiveAgent{
		{
			ID:             "aa_4",
			WSConnectionID: "client_4",
			OrganizationID: "org_3",
		},
		{
			ID:             "aa_5",
			WSConnectionID: "client_5",
			OrganizationID: "org_3",
		},
	}

	mockOrganizationsService.On("GetAllOrganizations", ctx).Return(organizations, nil)
	mockWsClient.On("GetClientIDs").Return(connectedClientIDs)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs).Return(agents1, nil)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_2", connectedClientIDs).Return(agents2, nil)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_3", connectedClientIDs).Return(agents3, nil)
	
	// Mock SendMessage for all agents
	for _, clientID := range connectedClientIDs {
		mockWsClient.On("SendMessage", clientID, mock.Anything).Return(nil)
	}

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	assert.NoError(t, err)
	mockWsClient.AssertNumberOfCalls(t, "SendMessage", 5)
	// Verify each organization was queried with correct org ID
	mockAgentsService.AssertCalled(t, "GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs)
	mockAgentsService.AssertCalled(t, "GetConnectedActiveAgents", ctx, "org_2", connectedClientIDs)
	mockAgentsService.AssertCalled(t, "GetConnectedActiveAgents", ctx, "org_3", connectedClientIDs)
}

func TestBroadcastCheckIdleJobs_MessageStructure(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	mockAgentsService := new(MockAgentsService)
	mockWsClient := new(MockSocketIOClient)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
		agentsService:        mockAgentsService,
		wsClient:            mockWsClient,
	}

	organizations := []*models.Organization{
		{
			ID: "org_1",
		},
	}

	connectedClientIDs := []string{"client_1"}

	agents := []*models.ActiveAgent{
		{
			ID:             "aa_1",
			WSConnectionID: "client_1",
			OrganizationID: "org_1",
		},
	}

	var capturedMessage models.BaseMessage

	mockOrganizationsService.On("GetAllOrganizations", ctx).Return(organizations, nil)
	mockWsClient.On("GetClientIDs").Return(connectedClientIDs)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs).Return(agents, nil)
	mockWsClient.On("SendMessage", "client_1", mock.Anything).Run(func(args mock.Arguments) {
		capturedMessage = args.Get(1).(models.BaseMessage)
	}).Return(nil)

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	require.NoError(t, err)
	
	// Verify message structure
	assert.True(t, strings.HasPrefix(capturedMessage.ID, "msg_"), "Message ID should start with 'msg_'")
	assert.Equal(t, models.MessageTypeCheckIdleJobs, capturedMessage.Type)
	
	// Verify payload is CheckIdleJobsPayload (empty struct)
	payload, ok := capturedMessage.Payload.(models.CheckIdleJobsPayload)
	assert.True(t, ok, "Payload should be CheckIdleJobsPayload type")
	assert.Equal(t, models.CheckIdleJobsPayload{}, payload, "Payload should be empty CheckIdleJobsPayload struct")
}

func TestBroadcastCheckIdleJobs_EmptyConnectedClients(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	mockAgentsService := new(MockAgentsService)
	mockWsClient := new(MockSocketIOClient)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
		agentsService:        mockAgentsService,
		wsClient:            mockWsClient,
	}

	organizations := []*models.Organization{
		{
			ID: "org_1",
		},
	}

	// No connected clients
	connectedClientIDs := []string{}

	mockOrganizationsService.On("GetAllOrganizations", ctx).Return(organizations, nil)
	mockWsClient.On("GetClientIDs").Return(connectedClientIDs)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs).Return([]*models.ActiveAgent{}, nil)

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	assert.NoError(t, err)
	mockAgentsService.AssertCalled(t, "GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs)
	mockWsClient.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything)
}

func TestBroadcastCheckIdleJobs_PartialSuccess(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	mockAgentsService := new(MockAgentsService)
	mockWsClient := new(MockSocketIOClient)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
		agentsService:        mockAgentsService,
		wsClient:            mockWsClient,
	}

	organizations := []*models.Organization{
		{
			ID: "org_1",
		},
	}

	connectedClientIDs := []string{"client_1", "client_2"}

	agents := []*models.ActiveAgent{
		{
			ID:             "aa_1",
			WSConnectionID: "client_1",
			OrganizationID: "org_1",
		},
		{
			ID:             "aa_2",
			WSConnectionID: "client_2",
			OrganizationID: "org_1",
		},
	}

	mockOrganizationsService.On("GetAllOrganizations", ctx).Return(organizations, nil)
	mockWsClient.On("GetClientIDs").Return(connectedClientIDs)
	mockAgentsService.On("GetConnectedActiveAgents", ctx, "org_1", connectedClientIDs).Return(agents, nil)
	
	// First send succeeds, second fails
	mockWsClient.On("SendMessage", "client_1", mock.Anything).Return(nil)
	mockWsClient.On("SendMessage", "client_2", mock.Anything).Return(errors.New("send error"))

	// Act
	err := useCase.BroadcastCheckIdleJobs(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send CheckIdleJobs message to agent aa_2")
	// First message should have been sent successfully
	mockWsClient.AssertCalled(t, "SendMessage", "client_1", mock.Anything)
}

func TestBroadcastCheckIdleJobs_NilContext(t *testing.T) {
	// Setup
	mockOrganizationsService := new(MockOrganizationsService)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
	}

	// Service should handle nil context
	mockOrganizationsService.On("GetAllOrganizations", nil).Return([]*models.Organization{}, nil)

	// Act
	err := useCase.BroadcastCheckIdleJobs(nil)

	// Assert - assuming graceful handling
	assert.NoError(t, err)
	mockOrganizationsService.AssertNumberOfCalls(t, "GetAllOrganizations", 1)
}