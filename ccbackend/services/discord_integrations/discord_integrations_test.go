package discordintegrations

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ccbackend/clients"
	discordclient "ccbackend/clients/discord"
	"ccbackend/core"
	"ccbackend/models"
)

// MockDiscordIntegrationsRepository implements a mock for the Discord integrations repository
type MockDiscordIntegrationsRepository struct {
	mock.Mock
}

func (m *MockDiscordIntegrationsRepository) CreateDiscordIntegration(
	ctx context.Context,
	integration *models.DiscordIntegration,
) error {
	args := m.Called(ctx, integration)
	return args.Error(0)
}

func (m *MockDiscordIntegrationsRepository) GetDiscordIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID string,
) ([]*models.DiscordIntegration, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DiscordIntegration), args.Error(1)
}

func (m *MockDiscordIntegrationsRepository) GetAllDiscordIntegrations(
	ctx context.Context,
) ([]*models.DiscordIntegration, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DiscordIntegration), args.Error(1)
}

func (m *MockDiscordIntegrationsRepository) DeleteDiscordIntegrationByID(
	ctx context.Context,
	integrationID, organizationID string,
) (bool, error) {
	args := m.Called(ctx, integrationID, organizationID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDiscordIntegrationsRepository) GetDiscordIntegrationByGuildID(
	ctx context.Context,
	guildID string,
) (mo.Option[*models.DiscordIntegration], error) {
	args := m.Called(ctx, guildID)
	if args.Get(0) == nil {
		return mo.None[*models.DiscordIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.DiscordIntegration]), args.Error(1)
}

func (m *MockDiscordIntegrationsRepository) GetDiscordIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.DiscordIntegration], error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return mo.None[*models.DiscordIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.DiscordIntegration]), args.Error(1)
}

func TestDiscordIntegrationsService_CreateDiscordIntegration_Success(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	organizationID := core.NewID("org")
	discordAuthCode := "test-auth-code"
	guildID := "1234567890"
	redirectURL := "https://example.com/redirect"

	// Mock guild response - OAuth is no longer used by service
	mockGuild := &clients.DiscordGuild{
		ID:   guildID,
		Name: "Test Guild",
	}

	// Setup expectations - only GetGuildByID is called now
	mockClient.On("GetGuildByID",
		guildID).Return(mockGuild, nil)

	mockRepo.On("CreateDiscordIntegration", ctx, mock.AnythingOfType("*models.DiscordIntegration")).Return(nil)

	// Act
	result, err := service.CreateDiscordIntegration(ctx, organizationID, discordAuthCode, guildID, redirectURL)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, guildID, result.DiscordGuildID)
	assert.Equal(t, "Test Guild", result.DiscordGuildName)
	assert.Equal(t, organizationID, result.OrganizationID)
	// DiscordAuthToken field was removed from the model
	assert.True(t, core.IsValidULID(result.ID))

	// Verify all expectations were met
	mockClient.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// EmptyAccessToken test removed - OAuth is no longer part of the service flow

func TestDiscordIntegrationsService_CreateDiscordIntegration_GuildNotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	organizationID := core.NewID("org")
	discordAuthCode := "test-auth-code"
	guildID := "1234567890"
	redirectURL := "https://example.com/redirect"

	// Mock guild fetch failure
	mockClient.On("GetGuildByID",
		guildID).Return(nil, fmt.Errorf("Discord API error: guild not found"))

	// Act
	result, err := service.CreateDiscordIntegration(ctx, organizationID, discordAuthCode, guildID, redirectURL)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch Discord guild information")
	assert.Contains(t, err.Error(), "guild not found")

	// Verify expectations
	mockClient.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestDiscordIntegrationsService_CreateDiscordIntegration_EmptyGuildName(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	organizationID := core.NewID("org")
	discordAuthCode := "test-auth-code"
	guildID := "1234567890"
	redirectURL := "https://example.com/redirect"

	// Mock guild response with empty name
	mockGuild := &clients.DiscordGuild{
		ID:   guildID,
		Name: "", // Empty guild name
	}

	mockClient.On("GetGuildByID",
		guildID).Return(mockGuild, nil)

	// Act
	result, err := service.CreateDiscordIntegration(ctx, organizationID, discordAuthCode, guildID, redirectURL)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "guild name not found in Discord API response")

	// Verify expectations
	mockClient.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestDiscordIntegrationsService_GetDiscordIntegrationsByOrganizationID_Success(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	organizationID := core.NewID("org")

	expectedIntegrations := []*models.DiscordIntegration{
		{
			ID:               core.NewID("di"),
			DiscordGuildID:   "123456789012345678",
			DiscordGuildName: "Guild 1",
			OrganizationID:   organizationID,
		},
		{
			ID:               core.NewID("di"),
			DiscordGuildID:   "987654321098765432",
			DiscordGuildName: "Guild 2",
			OrganizationID:   organizationID,
		},
	}

	mockRepo.On("GetDiscordIntegrationsByOrganizationID", ctx, organizationID).
		Return(expectedIntegrations, nil)

	// Act
	result, err := service.GetDiscordIntegrationsByOrganizationID(ctx, organizationID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedIntegrations, result)

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestDiscordIntegrationsService_GetDiscordIntegrationsByOrganizationID_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	organizationID := core.NewID("org")

	mockRepo.On("GetDiscordIntegrationsByOrganizationID", ctx, organizationID).
		Return(nil, fmt.Errorf("database connection error"))

	// Act
	result, err := service.GetDiscordIntegrationsByOrganizationID(ctx, organizationID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get discord integrations for organization")
	assert.Contains(t, err.Error(), "database connection error")

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestDiscordIntegrationsService_GetAllDiscordIntegrations_Success(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()

	expectedIntegrations := []*models.DiscordIntegration{
		{
			ID:               core.NewID("di"),
			DiscordGuildID:   "123456789012345678",
			DiscordGuildName: "Guild 1",
			OrganizationID:   core.NewID("org"),
		},
	}

	mockRepo.On("GetAllDiscordIntegrations", ctx).Return(expectedIntegrations, nil)

	// Act
	result, err := service.GetAllDiscordIntegrations(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedIntegrations, result)

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestDiscordIntegrationsService_DeleteDiscordIntegration_Success(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	organizationID := core.NewID("org")
	integrationID := core.NewID("di")

	mockRepo.On("DeleteDiscordIntegrationByID", ctx, integrationID, organizationID).Return(true, nil)

	// Act
	err := service.DeleteDiscordIntegration(ctx, organizationID, integrationID)

	// Assert
	require.NoError(t, err)

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestDiscordIntegrationsService_DeleteDiscordIntegration_NotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	organizationID := core.NewID("org")
	integrationID := core.NewID("di")

	mockRepo.On("DeleteDiscordIntegrationByID", ctx, integrationID, organizationID).Return(false, nil)

	// Act
	err := service.DeleteDiscordIntegration(ctx, organizationID, integrationID)

	// Assert
	require.Error(t, err)
	assert.Equal(t, core.ErrNotFound, err)

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestDiscordIntegrationsService_GetDiscordIntegrationByGuildID_Success(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	guildID := "123456789012345678"

	expectedIntegration := &models.DiscordIntegration{
		ID:               core.NewID("di"),
		DiscordGuildID:   guildID,
		DiscordGuildName: "Test Guild",
		OrganizationID:   core.NewID("org"),
	}

	mockRepo.On("GetDiscordIntegrationByGuildID", ctx, guildID).
		Return(mo.Some(expectedIntegration), nil)

	// Act
	result, err := service.GetDiscordIntegrationByGuildID(ctx, guildID)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.IsPresent())
	assert.Equal(t, expectedIntegration, result.MustGet())

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestDiscordIntegrationsService_GetDiscordIntegrationByGuildID_NotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	guildID := "123456789012345678"

	mockRepo.On("GetDiscordIntegrationByGuildID", ctx, guildID).
		Return(mo.None[*models.DiscordIntegration](), nil)

	// Act
	result, err := service.GetDiscordIntegrationByGuildID(ctx, guildID)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.IsPresent())

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestDiscordIntegrationsService_GetDiscordIntegrationByID_Success(t *testing.T) {
	// Arrange
	mockRepo := &MockDiscordIntegrationsRepository{}
	mockClient := &discordclient.MockDiscordClient{}

	service := NewDiscordIntegrationsService(mockRepo, mockClient, "test-client-id", "test-client-secret")

	ctx := context.Background()
	integrationID := core.NewID("di")

	expectedIntegration := &models.DiscordIntegration{
		ID:               integrationID,
		DiscordGuildID:   "123456789012345678",
		DiscordGuildName: "Test Guild",
		OrganizationID:   core.NewID("org"),
	}

	mockRepo.On("GetDiscordIntegrationByID", ctx, integrationID).
		Return(mo.Some(expectedIntegration), nil)

	// Act
	result, err := service.GetDiscordIntegrationByID(ctx, integrationID)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.IsPresent())
	assert.Equal(t, expectedIntegration, result.MustGet())

	// Verify expectations
	mockRepo.AssertExpectations(t)
}
