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
	organizationID models.OrgID,
	jobID string,
	slackChannelID, slackTS, textContent, slackIntegrationID string,
	status models.ProcessedSlackMessageStatus,
) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, organizationID, jobID, slackChannelID, slackTS, textContent, slackIntegrationID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockSlackMessagesService) UpdateProcessedSlackMessage(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, organizationID, id, status, slackIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockSlackMessagesService) GetProcessedMessagesByJobIDAndStatus(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
) ([]*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, organizationID, jobID, status, slackIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockSlackMessagesService) GetProcessedSlackMessageByID(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return mo.None[*models.ProcessedSlackMessage](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ProcessedSlackMessage]), args.Error(1)
}

func (m *MockSlackMessagesService) GetLatestProcessedMessageForJob(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
	slackIntegrationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	args := m.Called(ctx, organizationID, jobID, slackIntegrationID)
	if args.Get(0) == nil {
		return mo.None[*models.ProcessedSlackMessage](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ProcessedSlackMessage]), args.Error(1)
}

func (m *MockSlackMessagesService) GetActiveMessageCountForJobs(
	ctx context.Context,
	organizationID models.OrgID,
	jobIDs []string,
	slackIntegrationID string,
) (int, error) {
	args := m.Called(ctx, organizationID, jobIDs, slackIntegrationID)
	return args.Int(0), args.Error(1)
}

func (m *MockSlackMessagesService) TESTS_UpdateProcessedSlackMessageUpdatedAt(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
) error {
	args := m.Called(ctx, organizationID, id, updatedAt, slackIntegrationID)
	return args.Error(0)
}

func (m *MockSlackMessagesService) DeleteProcessedSlackMessagesByJobID(
	ctx context.Context,
	organizationID models.OrgID,
	jobID string,
	slackIntegrationID string,
) error {
	args := m.Called(ctx, organizationID, jobID, slackIntegrationID)
	return args.Error(0)
}

func (m *MockSlackMessagesService) GetProcessedMessagesByStatus(
	ctx context.Context,
	organizationID models.OrgID,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
) ([]*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, organizationID, status, slackIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProcessedSlackMessage), args.Error(1)
}
