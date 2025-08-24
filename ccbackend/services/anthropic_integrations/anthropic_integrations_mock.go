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
	organizationID models.OrgID,
	apiKey, oauthToken *string,
) (*models.AnthropicIntegration, error) {
	args := m.Called(ctx, organizationID, apiKey, oauthToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AnthropicIntegration), args.Error(1)
}

func (m *MockAnthropicIntegrationsService) ListAnthropicIntegrations(
	ctx context.Context,
	organizationID models.OrgID,
) ([]models.AnthropicIntegration, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.AnthropicIntegration), args.Error(1)
}

func (m *MockAnthropicIntegrationsService) GetAnthropicIntegrationByID(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) (mo.Option[*models.AnthropicIntegration], error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return mo.None[*models.AnthropicIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.AnthropicIntegration]), args.Error(1)
}

func (m *MockAnthropicIntegrationsService) DeleteAnthropicIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	integrationID string,
) error {
	args := m.Called(ctx, organizationID, integrationID)
	return args.Error(0)
}