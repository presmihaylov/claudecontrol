package discordmessages

import (
	"context"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

type MockDiscordMessagesService struct {
	mock.Mock
}

func (m *MockDiscordMessagesService) CreateProcessedDiscordMessage(
	ctx context.Context,
	jobID string,
	discordMessageID, discordThreadID, textContent, discordIntegrationID, organizationID string,
	status models.ProcessedDiscordMessageStatus,
) (*models.ProcessedDiscordMessage, error) {
	args := m.Called(
		ctx,
		jobID,
		discordMessageID,
		discordThreadID,
		textContent,
		discordIntegrationID,
		organizationID,
		status,
	)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedDiscordMessage), args.Error(1)
}

func (m *MockDiscordMessagesService) UpdateProcessedDiscordMessage(
	ctx context.Context,
	id string,
	status models.ProcessedDiscordMessageStatus,
	discordIntegrationID string,
	organizationID string,
) (*models.ProcessedDiscordMessage, error) {
	args := m.Called(ctx, id, status, discordIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedDiscordMessage), args.Error(1)
}

func (m *MockDiscordMessagesService) GetProcessedMessagesByJobIDAndStatus(
	ctx context.Context,
	jobID string,
	status models.ProcessedDiscordMessageStatus,
	discordIntegrationID string,
	organizationID string,
) ([]*models.ProcessedDiscordMessage, error) {
	args := m.Called(ctx, jobID, status, discordIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProcessedDiscordMessage), args.Error(1)
}

func (m *MockDiscordMessagesService) GetProcessedDiscordMessageByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.ProcessedDiscordMessage], error) {
	args := m.Called(ctx, id, organizationID)
	if args.Get(0) == nil {
		return mo.None[*models.ProcessedDiscordMessage](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ProcessedDiscordMessage]), args.Error(1)
}

func (m *MockDiscordMessagesService) GetLatestProcessedMessageForJob(
	ctx context.Context,
	jobID string,
	discordIntegrationID string,
	organizationID string,
) (mo.Option[*models.ProcessedDiscordMessage], error) {
	args := m.Called(ctx, jobID, discordIntegrationID, organizationID)
	if args.Get(0) == nil {
		return mo.None[*models.ProcessedDiscordMessage](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ProcessedDiscordMessage]), args.Error(1)
}

func (m *MockDiscordMessagesService) GetActiveMessageCountForJobs(
	ctx context.Context,
	jobIDs []string,
	discordIntegrationID string,
	organizationID string,
) (int, error) {
	args := m.Called(ctx, jobIDs, discordIntegrationID, organizationID)
	return args.Int(0), args.Error(1)
}

func (m *MockDiscordMessagesService) TESTS_UpdateProcessedDiscordMessageUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	discordIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, id, updatedAt, discordIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockDiscordMessagesService) DeleteProcessedDiscordMessagesByJobID(
	ctx context.Context,
	jobID string,
	discordIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, jobID, discordIntegrationID, organizationID)
	return args.Error(0)
}
