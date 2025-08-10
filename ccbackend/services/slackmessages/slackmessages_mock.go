package slackmessages

import (
	"context"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

type MockSlackMessagesService struct {
	mock.Mock
}

func (m *MockSlackMessagesService) CreateProcessedSlackMessage(
	ctx context.Context,
	jobID string,
	slackChannelID, slackTS, textContent, slackIntegrationID, organizationID string,
	status models.ProcessedSlackMessageStatus,
) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, jobID, slackChannelID, slackTS, textContent, slackIntegrationID, organizationID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockSlackMessagesService) UpdateProcessedSlackMessage(
	ctx context.Context,
	id string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
	organizationID string,
) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, id, status, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockSlackMessagesService) GetProcessedMessagesByJobIDAndStatus(
	ctx context.Context,
	jobID string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
	organizationID string,
) ([]*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, jobID, status, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockSlackMessagesService) GetProcessedSlackMessageByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	args := m.Called(ctx, id, organizationID)
	if args.Get(0) == nil {
		return mo.None[*models.ProcessedSlackMessage](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ProcessedSlackMessage]), args.Error(1)
}

func (m *MockSlackMessagesService) GetLatestProcessedMessageForJob(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	args := m.Called(ctx, jobID, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return mo.None[*models.ProcessedSlackMessage](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ProcessedSlackMessage]), args.Error(1)
}

func (m *MockSlackMessagesService) GetActiveMessageCountForJobs(
	ctx context.Context,
	jobIDs []string,
	slackIntegrationID string,
	organizationID string,
) (int, error) {
	args := m.Called(ctx, jobIDs, slackIntegrationID, organizationID)
	return args.Int(0), args.Error(1)
}

func (m *MockSlackMessagesService) TESTS_UpdateProcessedSlackMessageUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, id, updatedAt, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockSlackMessagesService) DeleteProcessedSlackMessagesByJobID(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, jobID, slackIntegrationID, organizationID)
	return args.Error(0)
}
