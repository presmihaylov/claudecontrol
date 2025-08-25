package slack

import (
	"context"

	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockSlackUseCase is a mock implementation of the SlackUseCase
type MockSlackUseCase struct {
	mock.Mock
}

func (m *MockSlackUseCase) ProcessSlackMessageEvent(
	ctx context.Context,
	event models.SlackMessageEvent,
	slackIntegrationID string,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, event, slackIntegrationID, orgID)
	return args.Error(0)
}

func (m *MockSlackUseCase) ProcessReactionAdded(
	ctx context.Context,
	reactionName, userID, channelID, messageTS, slackIntegrationID string,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, reactionName, userID, channelID, messageTS, slackIntegrationID, orgID)
	return args.Error(0)
}

func (m *MockSlackUseCase) ProcessProcessingMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingMessagePayload,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, clientID, payload, orgID)
	return args.Error(0)
}

func (m *MockSlackUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, clientID, payload, orgID)
	return args.Error(0)
}

func (m *MockSlackUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, clientID, payload, orgID)
	return args.Error(0)
}

func (m *MockSlackUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, clientID, payload, orgID)
	return args.Error(0)
}

func (m *MockSlackUseCase) ProcessQueuedJobs(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSlackUseCase) CleanupFailedSlackJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	failureMessage string,
) error {
	args := m.Called(ctx, job, agentID, failureMessage)
	return args.Error(0)
}
