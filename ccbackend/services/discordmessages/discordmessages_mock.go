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
	organizationID models.OrgID,
	jobID string,
	discordMessageID, discordThreadID, textContent, discordIntegrationID string,
	status models.ProcessedDiscordMessageStatus,
) (*models.ProcessedDiscordMessage, error) {
	args := m.Called(
		ctx,
		organizationID,
		jobID,
		discordMessageID,
		discordThreadID,
		textContent,
		discordIntegrationID,
		status,
	)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedDiscordMessage), args.Error(1)
}

func (m *MockDiscordMessagesService) UpdateProcessedDiscordMessage(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
	status models.ProcessedDiscordMessageStatus,
	discordIntegrationID string,
) (*models.ProcessedDiscordMessage, error) {
	args := m.Called(ctx, organizationID, id, status, discordIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedDiscordMessage), args.Error(1)
}

func (m *MockDiscordMessagesService) GetProcessedMessagesByJobIDAndStatus(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
	status models.ProcessedDiscordMessageStatus,
	discordIntegrationID string,
) ([]*models.ProcessedDiscordMessage, error) {
	args := m.Called(ctx, organizationID, jobID, status, discordIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProcessedDiscordMessage), args.Error(1)
}

func (m *MockDiscordMessagesService) GetProcessedDiscordMessageByID(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) (mo.Option[*models.ProcessedDiscordMessage], error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return mo.None[*models.ProcessedDiscordMessage](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ProcessedDiscordMessage]), args.Error(1)
}

func (m *MockDiscordMessagesService) GetLatestProcessedMessageForJob(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
	discordIntegrationID string,
) (mo.Option[*models.ProcessedDiscordMessage], error) {
	args := m.Called(ctx, organizationID, jobID, discordIntegrationID)
	if args.Get(0) == nil {
		return mo.None[*models.ProcessedDiscordMessage](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ProcessedDiscordMessage]), args.Error(1)
}

func (m *MockDiscordMessagesService) GetActiveMessageCountForJobs(
	ctx context.Context,
	organizationID models.OrgID,
	jobIDs []string,
	discordIntegrationID string,
) (int, error) {
	args := m.Called(ctx, organizationID, jobIDs, discordIntegrationID)
	return args.Int(0), args.Error(1)
}

func (m *MockDiscordMessagesService) TESTS_UpdateProcessedDiscordMessageUpdatedAt(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
	updatedAt time.Time,
	discordIntegrationID string,
) error {
	args := m.Called(ctx, organizationID, id, updatedAt, discordIntegrationID)
	return args.Error(0)
}

func (m *MockDiscordMessagesService) DeleteProcessedDiscordMessagesByJobID(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
	discordIntegrationID string,
) error {
	args := m.Called(ctx, organizationID, jobID, discordIntegrationID)
	return args.Error(0)
}

func (m *MockDiscordMessagesService) GetProcessedMessagesByStatus(
	ctx context.Context,
	organizationID models.OrgID,
	status models.ProcessedDiscordMessageStatus,
	discordIntegrationID string,
) ([]*models.ProcessedDiscordMessage, error) {
	args := m.Called(ctx, organizationID, status, discordIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProcessedDiscordMessage), args.Error(1)
}
