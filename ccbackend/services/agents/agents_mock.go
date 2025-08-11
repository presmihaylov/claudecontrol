package agents

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockAgentsService is a mock implementation of the AgentsService interface
type MockAgentsService struct {
	mock.Mock
}

func (m *MockAgentsService) UpsertActiveAgent(
	ctx context.Context,
	wsConnectionID string, organizationID models.OrgID,
	agentID string,
) (*models.ActiveAgent, error) {
	args := m.Called(ctx, wsConnectionID, organizationID, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) DeleteActiveAgentByWsConnectionID(
	ctx context.Context,
	wsConnectionID string, organizationID models.OrgID,
) error {
	args := m.Called(ctx, wsConnectionID, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) DeleteActiveAgent(ctx context.Context, id string, organizationID models.OrgID) error {
	args := m.Called(ctx, id, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) GetAgentByID(
	ctx context.Context,
	id string,
	organizationID models.OrgID,
) (mo.Option[*models.ActiveAgent], error) {
	args := m.Called(ctx, id, organizationID)
	return args.Get(0).(mo.Option[*models.ActiveAgent]), args.Error(1)
}

func (m *MockAgentsService) GetAvailableAgents(
	ctx context.Context,
	organizationID models.OrgID,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) GetConnectedActiveAgents(
	ctx context.Context,
	organizationID models.OrgID,
	connectedClientIDs []string,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, organizationID, connectedClientIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) GetConnectedAvailableAgents(
	ctx context.Context,
	organizationID models.OrgID,
	connectedClientIDs []string,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, organizationID, connectedClientIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) CheckAgentHasActiveConnection(agent *models.ActiveAgent, connectedClientIDs []string) bool {
	args := m.Called(agent, connectedClientIDs)
	return args.Bool(0)
}

func (m *MockAgentsService) AssignAgentToJob(
	ctx context.Context,
	agentID, jobID string,
	organizationID models.OrgID,
) error {
	args := m.Called(ctx, agentID, jobID, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) UnassignAgentFromJob(
	ctx context.Context,
	agentID, jobID string,
	organizationID models.OrgID,
) error {
	args := m.Called(ctx, agentID, jobID, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) GetAgentByJobID(
	ctx context.Context,
	jobID string,
	organizationID models.OrgID,
) (mo.Option[*models.ActiveAgent], error) {
	args := m.Called(ctx, jobID, organizationID)
	return args.Get(0).(mo.Option[*models.ActiveAgent]), args.Error(1)
}

func (m *MockAgentsService) GetAgentByWSConnectionID(
	ctx context.Context,
	wsConnectionID string, organizationID models.OrgID,
) (mo.Option[*models.ActiveAgent], error) {
	args := m.Called(ctx, wsConnectionID, organizationID)
	return args.Get(0).(mo.Option[*models.ActiveAgent]), args.Error(1)
}

func (m *MockAgentsService) GetActiveAgentJobAssignments(
	ctx context.Context,
	agentID string,
	organizationID models.OrgID,
) ([]string, error) {
	args := m.Called(ctx, agentID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAgentsService) UpdateAgentLastActiveAt(
	ctx context.Context,
	wsConnectionID string, organizationID models.OrgID,
) error {
	args := m.Called(ctx, wsConnectionID, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) GetInactiveAgents(
	ctx context.Context,
	organizationID models.OrgID,
	inactiveThresholdMinutes int,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, organizationID, inactiveThresholdMinutes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}
