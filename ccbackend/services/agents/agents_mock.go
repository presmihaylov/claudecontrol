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
	orgID models.OrgID,
	wsConnectionID string,
	agentID string,
) (*models.ActiveAgent, error) {
	args := m.Called(ctx, orgID, wsConnectionID, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) DeleteActiveAgentByWsConnectionID(
	ctx context.Context,
	orgID models.OrgID,
	wsConnectionID string,
) error {
	args := m.Called(ctx, orgID, wsConnectionID)
	return args.Error(0)
}

func (m *MockAgentsService) DeleteActiveAgent(ctx context.Context, orgID models.OrgID, id string) error {
	args := m.Called(ctx, orgID, id)
	return args.Error(0)
}

func (m *MockAgentsService) GetAgentByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.ActiveAgent], error) {
	args := m.Called(ctx, orgID, id)
	return args.Get(0).(mo.Option[*models.ActiveAgent]), args.Error(1)
}

func (m *MockAgentsService) GetAvailableAgents(
	ctx context.Context,
	orgID models.OrgID,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) GetConnectedActiveAgents(
	ctx context.Context,
	orgID models.OrgID,
	connectedClientIDs []string,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, orgID, connectedClientIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) GetConnectedAvailableAgents(
	ctx context.Context,
	orgID models.OrgID,
	connectedClientIDs []string,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, orgID, connectedClientIDs)
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
	orgID models.OrgID,
	agentID, jobID string,
) error {
	args := m.Called(ctx, orgID, agentID, jobID)
	return args.Error(0)
}

func (m *MockAgentsService) UnassignAgentFromJob(
	ctx context.Context,
	orgID models.OrgID,
	agentID, jobID string,
) error {
	args := m.Called(ctx, orgID, agentID, jobID)
	return args.Error(0)
}

func (m *MockAgentsService) GetAgentByJobID(
	ctx context.Context,
	orgID models.OrgID,
	jobID string,
) (mo.Option[*models.ActiveAgent], error) {
	args := m.Called(ctx, orgID, jobID)
	return args.Get(0).(mo.Option[*models.ActiveAgent]), args.Error(1)
}

func (m *MockAgentsService) GetAgentByWSConnectionID(
	ctx context.Context,
	orgID models.OrgID,
	wsConnectionID string,
) (mo.Option[*models.ActiveAgent], error) {
	args := m.Called(ctx, orgID, wsConnectionID)
	return args.Get(0).(mo.Option[*models.ActiveAgent]), args.Error(1)
}

func (m *MockAgentsService) GetActiveAgentJobAssignments(
	ctx context.Context,
	orgID models.OrgID,
	agentID string,
) ([]string, error) {
	args := m.Called(ctx, orgID, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAgentsService) UpdateAgentLastActiveAt(
	ctx context.Context,
	orgID models.OrgID,
	wsConnectionID string,
) error {
	args := m.Called(ctx, orgID, wsConnectionID)
	return args.Error(0)
}

func (m *MockAgentsService) GetInactiveAgents(
	ctx context.Context,
	orgID models.OrgID,
	inactiveThresholdMinutes int,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, orgID, inactiveThresholdMinutes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) DisconnectAllActiveAgentsByOrganization(
	ctx context.Context,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, orgID)
	return args.Error(0)
}
