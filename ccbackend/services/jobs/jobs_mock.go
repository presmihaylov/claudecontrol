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

func (m *MockJobsService) CreateJob(
	ctx context.Context,
	slackThreadTS, slackChannelID, slackUserID, slackIntegrationID, organizationID string,
) (*models.Job, error) {
	args := m.Called(ctx, slackThreadTS, slackChannelID, slackUserID, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobsService) GetJobByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, id, organizationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetJobBySlackThread(
	ctx context.Context,
	threadTS, channelID, slackIntegrationID, organizationID string,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, threadTS, channelID, slackIntegrationID, organizationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetOrCreateJobForSlackThread(
	ctx context.Context,
	threadTS, channelID, slackUserID, slackIntegrationID, organizationID string,
) (*models.JobCreationResult, error) {
	args := m.Called(ctx, threadTS, channelID, slackUserID, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobCreationResult), args.Error(1)
}

func (m *MockJobsService) UpdateJobTimestamp(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, jobID, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) GetIdleJobs(
	ctx context.Context,
	idleMinutes int,
	organizationID string,
) ([]*models.Job, error) {
	args := m.Called(ctx, idleMinutes, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Job), args.Error(1)
}

func (m *MockJobsService) DeleteJob(
	ctx context.Context,
	id string,
	slackIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, id, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) TESTS_UpdateJobUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, id, updatedAt, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) GetJobsWithQueuedMessages(
	ctx context.Context,
	slackIntegrationID string,
	organizationID string,
) ([]*models.Job, error) {
	args := m.Called(ctx, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Job), args.Error(1)
}
