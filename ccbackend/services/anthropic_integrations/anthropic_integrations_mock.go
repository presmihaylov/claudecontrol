package anthropic_integrations

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockAnthropicIntegrationsService is a mock implementation of the AnthropicIntegrationsService interface
type MockAnthropicIntegrationsService struct {
	mock.Mock
}

func (m *MockAnthropicIntegrationsService) CreateAnthropicIntegration(
	ctx context.Context,
	orgID models.OrgID,
	apiKey, oauthToken, codeVerifier *string,
) (*models.AnthropicIntegration, error) {
	args := m.Called(ctx, orgID, apiKey, oauthToken, codeVerifier)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AnthropicIntegration), args.Error(1)
}

func (m *MockAnthropicIntegrationsService) ListAnthropicIntegrations(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.AnthropicIntegration, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.AnthropicIntegration), args.Error(1)
}

func (m *MockAnthropicIntegrationsService) GetAnthropicIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.AnthropicIntegration], error) {
	args := m.Called(ctx, orgID, id)
	if args.Get(0) == nil {
		return mo.None[*models.AnthropicIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.AnthropicIntegration]), args.Error(1)
}

func (m *MockAnthropicIntegrationsService) DeleteAnthropicIntegration(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) error {
	args := m.Called(ctx, orgID, integrationID)
	return args.Error(0)
}

func (m *MockAnthropicIntegrationsService) RefreshTokens(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) (*models.AnthropicIntegration, error) {
	args := m.Called(ctx, orgID, integrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AnthropicIntegration), args.Error(1)
}
