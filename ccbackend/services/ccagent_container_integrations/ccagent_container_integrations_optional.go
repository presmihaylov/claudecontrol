package ccagentcontainerintegrations

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"ccbackend/models"
)

// UnconfiguredCCAgentContainerIntegrationsService returns errors for all operations when SSH/Container is not configured
type UnconfiguredCCAgentContainerIntegrationsService struct{}

// NewUnconfiguredCCAgentContainerIntegrationsService creates a new unconfigured CCAgent container integrations service
func NewUnconfiguredCCAgentContainerIntegrationsService() *UnconfiguredCCAgentContainerIntegrationsService {
	return &UnconfiguredCCAgentContainerIntegrationsService{}
}

func (s *UnconfiguredCCAgentContainerIntegrationsService) CreateCCAgentContainerIntegration(
	ctx context.Context,
	orgID models.OrgID,
	instancesCount int,
	repoURL string,
) (*models.CCAgentContainerIntegration, error) {
	return nil, fmt.Errorf("Service CCAgentContainer is not configured")
}

func (s *UnconfiguredCCAgentContainerIntegrationsService) ListCCAgentContainerIntegrations(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.CCAgentContainerIntegration, error) {
	return nil, fmt.Errorf("Service CCAgentContainer is not configured")
}

func (s *UnconfiguredCCAgentContainerIntegrationsService) GetCCAgentContainerIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.CCAgentContainerIntegration], error) {
	return mo.None[*models.CCAgentContainerIntegration](), fmt.Errorf("Service CCAgentContainer is not configured")
}

func (s *UnconfiguredCCAgentContainerIntegrationsService) DeleteCCAgentContainerIntegration(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) error {
	return fmt.Errorf("Service CCAgentContainer is not configured")
}

func (s *UnconfiguredCCAgentContainerIntegrationsService) RedeployCCAgentContainer(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
	updateConfigOnly bool,
) error {
	return fmt.Errorf("Service CCAgentContainer is not configured")
}