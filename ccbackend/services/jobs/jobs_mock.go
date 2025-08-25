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
	orgID models.OrgID,
	slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string,
) (*models.Job, error) {
	args := m.Called(ctx, orgID, slackThreadTS, slackChannelID, slackUserID, slackIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobsService) GetJobByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, orgID, id)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetJobBySlackThread(
	ctx context.Context,
	orgID models.OrgID,
	threadTS, channelID, slackIntegrationID string,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, orgID, threadTS, channelID, slackIntegrationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetOrCreateJobForSlackThread(
	ctx context.Context,
	orgID models.OrgID,
	threadTS, channelID, slackUserID, slackIntegrationID string,
) (*models.JobCreationResult, error) {
	args := m.Called(ctx, orgID, threadTS, channelID, slackUserID, slackIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobCreationResult), args.Error(1)
}

func (m *MockJobsService) UpdateJobTimestamp(
	ctx context.Context,
	orgID models.OrgID,
	jobID string,
) error {
	args := m.Called(ctx, orgID, jobID)
	return args.Error(0)
}

func (m *MockJobsService) GetIdleJobs(
	ctx context.Context,
	orgID models.OrgID,
	idleMinutes int,
) ([]*models.Job, error) {
	args := m.Called(ctx, orgID, idleMinutes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Job), args.Error(1)
}

func (m *MockJobsService) DeleteJob(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) error {
	args := m.Called(ctx, orgID, id)
	return args.Error(0)
}

func (m *MockJobsService) TESTS_UpdateJobUpdatedAt(
	ctx context.Context,
	orgID models.OrgID,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
) error {
	args := m.Called(ctx, orgID, id, updatedAt, slackIntegrationID)
	return args.Error(0)
}

// Discord-specific methods

func (m *MockJobsService) CreateDiscordJob(
	ctx context.Context,
	orgID models.OrgID,
	discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
) (*models.Job, error) {
	args := m.Called(
		ctx,
		orgID,
		discordMessageID,
		discordChannelID,
		discordThreadID,
		discordUserID,
		discordIntegrationID,
	)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobsService) GetJobByDiscordThread(
	ctx context.Context,
	orgID models.OrgID,
	threadID, discordIntegrationID string,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, orgID, threadID, discordIntegrationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetOrCreateJobForDiscordThread(
	ctx context.Context,
	orgID models.OrgID,
	discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
) (*models.JobCreationResult, error) {
	args := m.Called(
		ctx,
		orgID,
		discordMessageID,
		discordChannelID,
		discordThreadID,
		discordUserID,
		discordIntegrationID,
	)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobCreationResult), args.Error(1)
}
