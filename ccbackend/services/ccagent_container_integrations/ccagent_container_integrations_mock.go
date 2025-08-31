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
	orgID models.OrgID,
	instancesCount int,
	repoURL string,
) (*models.CCAgentContainerIntegration, error) {
	args := m.Called(ctx, orgID, instancesCount, repoURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CCAgentContainerIntegration), args.Error(1)
}

func (m *MockCCAgentContainerIntegrationsService) ListCCAgentContainerIntegrations(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.CCAgentContainerIntegration, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.CCAgentContainerIntegration), args.Error(1)
}

func (m *MockCCAgentContainerIntegrationsService) GetCCAgentContainerIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.CCAgentContainerIntegration], error) {
	args := m.Called(ctx, orgID, id)
	if args.Get(0) == nil {
		return mo.None[*models.CCAgentContainerIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.CCAgentContainerIntegration]), args.Error(1)
}

func (m *MockCCAgentContainerIntegrationsService) DeleteCCAgentContainerIntegration(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) error {
	args := m.Called(ctx, orgID, integrationID)
	return args.Error(0)
}

func (m *MockCCAgentContainerIntegrationsService) RedeployCCAgentContainer(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
	updateConfigOnly bool,
) error {
	args := m.Called(ctx, orgID, integrationID, updateConfigOnly)
	return args.Error(0)
}
