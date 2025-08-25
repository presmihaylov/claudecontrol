package discord

import (
	"context"

	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockDiscordUseCase is a mock implementation of the DiscordUseCase
type MockDiscordUseCase struct {
	mock.Mock
}

func (m *MockDiscordUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, clientID, payload, orgID)
	return args.Error(0)
}

func (m *MockDiscordUseCase) ProcessProcessingMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingMessagePayload,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, clientID, payload, orgID)
	return args.Error(0)
}

func (m *MockDiscordUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	orgID models.OrgID,
) error {
	args := m.Called(ctx, clientID, payload, orgID)
	return args.Error(0)
}

func (m *MockDiscordUseCase) CleanupFailedDiscordJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	failureMessage string,
) error {
	args := m.Called(ctx, job, agentID, failureMessage)
	return args.Error(0)
}
