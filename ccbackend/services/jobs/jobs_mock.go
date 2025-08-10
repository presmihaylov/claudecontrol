package jobs

import (
	"context"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockJobsService is a mock implementation of the JobsService interface
type MockJobsService struct {
	mock.Mock
}

func (m *MockJobsService) GetActiveMessageCountForJobs(ctx context.Context, jobIDs []string, slackIntegrationID, organizationID string) (int, error) {
	args := m.Called(ctx, jobIDs, slackIntegrationID, organizationID)
	return args.Int(0), args.Error(1)
}

func (m *MockJobsService) CreateJob(ctx context.Context, slackThreadTS, slackChannelID, slackUserID, slackIntegrationID, organizationID string) (*models.Job, error) {
	args := m.Called(ctx, slackThreadTS, slackChannelID, slackUserID, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobsService) GetJobByID(ctx context.Context, id, organizationID string) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, id, organizationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetJobBySlackThread(ctx context.Context, threadTS, channelID, slackIntegrationID, organizationID string) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, threadTS, channelID, slackIntegrationID, organizationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetOrCreateJobForSlackThread(ctx context.Context, threadTS, channelID, slackUserID, slackIntegrationID, organizationID string) (*models.JobCreationResult, error) {
	args := m.Called(ctx, threadTS, channelID, slackUserID, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobCreationResult), args.Error(1)
}

func (m *MockJobsService) UpdateJobTimestamp(ctx context.Context, jobID, slackIntegrationID, organizationID string) error {
	args := m.Called(ctx, jobID, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) GetIdleJobs(ctx context.Context, idleMinutes int, organizationID string) ([]*models.Job, error) {
	args := m.Called(ctx, idleMinutes, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Job), args.Error(1)
}

func (m *MockJobsService) DeleteJob(ctx context.Context, id, slackIntegrationID, organizationID string) error {
	args := m.Called(ctx, id, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) CreateProcessedSlackMessage(ctx context.Context, jobID, slackChannelID, slackTS, textContent, slackIntegrationID, organizationID string, status models.ProcessedSlackMessageStatus) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, jobID, slackChannelID, slackTS, textContent, slackIntegrationID, organizationID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockJobsService) UpdateProcessedSlackMessage(ctx context.Context, id string, status models.ProcessedSlackMessageStatus, slackIntegrationID, organizationID string) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, id, status, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockJobsService) GetProcessedMessagesByJobIDAndStatus(ctx context.Context, jobID string, status models.ProcessedSlackMessageStatus, slackIntegrationID, organizationID string) ([]*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, jobID, status, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockJobsService) GetProcessedSlackMessageByID(ctx context.Context, id, organizationID string) (mo.Option[*models.ProcessedSlackMessage], error) {
	args := m.Called(ctx, id, organizationID)
	return args.Get(0).(mo.Option[*models.ProcessedSlackMessage]), args.Error(1)
}

func (m *MockJobsService) TESTS_UpdateJobUpdatedAt(ctx context.Context, id string, updatedAt time.Time, slackIntegrationID, organizationID string) error {
	args := m.Called(ctx, id, updatedAt, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) TESTS_UpdateProcessedSlackMessageUpdatedAt(ctx context.Context, id string, updatedAt time.Time, slackIntegrationID, organizationID string) error {
	args := m.Called(ctx, id, updatedAt, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) GetJobsWithQueuedMessages(ctx context.Context, slackIntegrationID, organizationID string) ([]*models.Job, error) {
	args := m.Called(ctx, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Job), args.Error(1)
}

func (m *MockJobsService) GetLatestProcessedMessageForJob(ctx context.Context, jobID, slackIntegrationID, organizationID string) (mo.Option[*models.ProcessedSlackMessage], error) {
	args := m.Called(ctx, jobID, slackIntegrationID, organizationID)
	return args.Get(0).(mo.Option[*models.ProcessedSlackMessage]), args.Error(1)
}