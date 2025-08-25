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
	orgID models.OrgID,
) (string, error) {
	args := m.Called(ctx, job, threadTS, orgID)
	return args.String(0), args.Error(1)
}

func (m *MockAgentsUseCase) AssignJobToAvailableAgent(
	ctx context.Context,
	job *models.Job,
	threadTS string,
	orgID models.OrgID,
) (string, error) {
	args := m.Called(ctx, job, threadTS, orgID)
	return args.String(0), args.Error(1)
}

func (m *MockAgentsUseCase) TryAssignJobToAgent(
	ctx context.Context,
	jobID string,
	orgID models.OrgID,
) (string, bool, error) {
	args := m.Called(ctx, jobID, orgID)
	return args.String(0), args.Bool(1), args.Error(2)
}

func (m *MockAgentsUseCase) ValidateJobBelongsToAgent(
	ctx context.Context,
	agentID, jobID string,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, agentID, jobID, orgID)
	return args.Error(0)
}
