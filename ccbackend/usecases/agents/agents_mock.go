package agents

import (
	"context"

	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockAgentsUseCase is a mock implementation of the AgentsUseCase
type MockAgentsUseCase struct {
	mock.Mock
}

func (m *MockAgentsUseCase) GetOrAssignAgentForJob(
	ctx context.Context,
	job *models.Job,
	threadTS string,
	organizationID models.OrgID,
) (string, error) {
	args := m.Called(ctx, job, threadTS, organizationID)
	return args.String(0), args.Error(1)
}

func (m *MockAgentsUseCase) AssignJobToAvailableAgent(
	ctx context.Context,
	job *models.Job,
	threadTS string,
	organizationID models.OrgID,
) (string, error) {
	args := m.Called(ctx, job, threadTS, organizationID)
	return args.String(0), args.Error(1)
}

func (m *MockAgentsUseCase) TryAssignJobToAgent(
	ctx context.Context,
	jobID string,
	organizationID string,
) (string, bool, error) {
	args := m.Called(ctx, jobID, organizationID)
	return args.String(0), args.Bool(1), args.Error(2)
}

func (m *MockAgentsUseCase) ValidateJobBelongsToAgent(
	ctx context.Context,
	agentID, jobID string,
	organizationID string,
) error {
	args := m.Called(ctx, agentID, jobID, organizationID)
	return args.Error(0)
}
