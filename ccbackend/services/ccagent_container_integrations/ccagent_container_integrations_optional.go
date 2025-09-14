package ccagentcontainerintegrations

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"ccbackend/models"
)

// OptionalCCAgentContainerIntegrationsService returns errors for all operations when SSH/Container is not configured
type OptionalCCAgentContainerIntegrationsService struct{}

// NewOptionalCCAgentContainerIntegrationsService creates a new optional CCAgent container integrations service
func NewOptionalCCAgentContainerIntegrationsService() *OptionalCCAgentContainerIntegrationsService {
	return &OptionalCCAgentContainerIntegrationsService{}
}

func (s *OptionalCCAgentContainerIntegrationsService) CreateCCAgentContainerIntegration(
	ctx context.Context,
	orgID models.OrgID,
	instancesCount int,
	repoURL string,
) (*models.CCAgentContainerIntegration, error) {
	return nil, fmt.Errorf("Service CCAgentContainer is not configured")
}

func (s *OptionalCCAgentContainerIntegrationsService) ListCCAgentContainerIntegrations(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.CCAgentContainerIntegration, error) {
	return nil, fmt.Errorf("Service CCAgentContainer is not configured")
}

func (s *OptionalCCAgentContainerIntegrationsService) GetCCAgentContainerIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.CCAgentContainerIntegration], error) {
	return mo.None[*models.CCAgentContainerIntegration](), fmt.Errorf("Service CCAgentContainer is not configured")
}

func (s *OptionalCCAgentContainerIntegrationsService) DeleteCCAgentContainerIntegration(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) error {
	return fmt.Errorf("Service CCAgentContainer is not configured")
}

func (s *OptionalCCAgentContainerIntegrationsService) RedeployCCAgentContainer(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
	updateConfigOnly bool,
) error {
	return fmt.Errorf("Service CCAgentContainer is not configured")
}