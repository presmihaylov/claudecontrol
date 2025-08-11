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

func (m *MockJobsService) CreateSlackJob(
	ctx context.Context,
	slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string,
	organizationID models.OrgID,
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
	organizationID models.OrgID,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, id, organizationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetJobBySlackThread(
	ctx context.Context,
	threadTS, channelID, slackIntegrationID string,
	organizationID models.OrgID,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, threadTS, channelID, slackIntegrationID, organizationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetOrCreateJobForSlackThread(
	ctx context.Context,
	threadTS, channelID, slackUserID, slackIntegrationID string,
	organizationID models.OrgID,
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
	organizationID models.OrgID,
) error {
	args := m.Called(ctx, jobID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) GetIdleJobs(
	ctx context.Context,
	idleMinutes int,
	organizationID models.OrgID,
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
	organizationID models.OrgID,
) error {
	args := m.Called(ctx, id, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) TESTS_UpdateJobUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
	organizationID models.OrgID,
) error {
	args := m.Called(ctx, id, updatedAt, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) GetJobsWithQueuedMessages(
	ctx context.Context,
	jobType models.JobType,
	integrationID string,
	organizationID models.OrgID,
) ([]*models.Job, error) {
	args := m.Called(ctx, jobType, integrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Job), args.Error(1)
}

// Discord-specific methods

func (m *MockJobsService) CreateDiscordJob(
	ctx context.Context,
	discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
	organizationID models.OrgID,
) (*models.Job, error) {
	args := m.Called(
		ctx,
		discordMessageID,
		discordChannelID,
		discordThreadID,
		discordUserID,
		discordIntegrationID,
		organizationID,
	)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobsService) GetJobByDiscordThread(
	ctx context.Context,
	threadID, discordIntegrationID string,
	organizationID models.OrgID,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, threadID, discordIntegrationID, organizationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetOrCreateJobForDiscordThread(
	ctx context.Context,
	discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
	organizationID models.OrgID,
) (*models.JobCreationResult, error) {
	args := m.Called(
		ctx,
		discordMessageID,
		discordChannelID,
		discordThreadID,
		discordUserID,
		discordIntegrationID,
		organizationID,
	)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobCreationResult), args.Error(1)
}
