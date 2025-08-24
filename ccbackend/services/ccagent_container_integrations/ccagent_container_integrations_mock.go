package ccagentcontainerintegrations

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

type MockCCAgentContainerIntegrationsService struct {
	mock.Mock
}

func (m *MockCCAgentContainerIntegrationsService) CreateCCAgentContainerIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	instancesCount int,
	repoURL string,
) (*models.CCAgentContainerIntegration, error) {
	args := m.Called(ctx, organizationID, instancesCount, repoURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CCAgentContainerIntegration), args.Error(1)
}

func (m *MockCCAgentContainerIntegrationsService) ListCCAgentContainerIntegrations(
	ctx context.Context,
	organizationID models.OrgID,
) ([]models.CCAgentContainerIntegration, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.CCAgentContainerIntegration), args.Error(1)
}

func (m *MockCCAgentContainerIntegrationsService) GetCCAgentContainerIntegrationByID(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) (mo.Option[*models.CCAgentContainerIntegration], error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return mo.None[*models.CCAgentContainerIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.CCAgentContainerIntegration]), args.Error(1)
}

func (m *MockCCAgentContainerIntegrationsService) DeleteCCAgentContainerIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	integrationID string,
) error {
	args := m.Called(ctx, organizationID, integrationID)
	return args.Error(0)
}
