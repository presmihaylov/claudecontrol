package ccagentcontainerintegrations

import (
	"context"
	"fmt"
	"log"

	"ccbackend/clients/ssh"
	"ccbackend/config"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/utils"

	"github.com/samber/mo"
)

// CCAgentContainerIntegrationsService handles CCAgent container integration operations
type CCAgentContainerIntegrationsService struct {
	repo                         *db.PostgresCCAgentContainerIntegrationsRepository
	config                       *config.AppConfig
	githubIntegrationsService    services.GitHubIntegrationsService
	anthropicIntegrationsService services.AnthropicIntegrationsService
	organizationsService         services.OrganizationsService
	sshClient                    ssh.SSHClientInterface
}

// NewCCAgentContainerIntegrationsService creates a new service instance
func NewCCAgentContainerIntegrationsService(
	repo *db.PostgresCCAgentContainerIntegrationsRepository,
	config *config.AppConfig,
	githubIntegrationsService services.GitHubIntegrationsService,
	anthropicIntegrationsService services.AnthropicIntegrationsService,
	organizationsService services.OrganizationsService,
	sshClient ssh.SSHClientInterface,
) *CCAgentContainerIntegrationsService {
	return &CCAgentContainerIntegrationsService{
		repo:                         repo,
		config:                       config,
		githubIntegrationsService:    githubIntegrationsService,
		anthropicIntegrationsService: anthropicIntegrationsService,
		organizationsService:         organizationsService,
		sshClient:                    sshClient,
	}
}

// CreateCCAgentContainerIntegration creates a new CCAgent container integration
func (s *CCAgentContainerIntegrationsService) CreateCCAgentContainerIntegration(
	ctx context.Context,
	orgID models.OrgID,
	instancesCount int,
	repoURL string,
) (*models.CCAgentContainerIntegration, error) {
	log.Printf("📋 Starting to create CCAgent container integration for org: %s", orgID)

	// Validation
	if instancesCount < 1 || instancesCount > 10 {
		return nil, fmt.Errorf("instances_count must be between 1 and 10")
	}
	if repoURL == "" {
		return nil, fmt.Errorf("repo_url cannot be empty")
	}

	// Sanitize repo URL to remove HTTPS prefixes
	sanitizedRepoURL := utils.SanitiseURL(repoURL)

	// Create new integration
	integration := &models.CCAgentContainerIntegration{
		ID:             core.NewID("cci"),
		InstancesCount: instancesCount,
		RepoURL:        sanitizedRepoURL,
		SSHHost:        s.config.DefaultSSHHost,
		OrgID:          orgID,
	}

	if err := s.repo.CreateCCAgentContainerIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create CCAgent container integration: %w", err)
	}

	log.Printf("📋 Completed successfully - created CCAgent container integration with ID: %s", integration.ID)
	return integration, nil
}

// ListCCAgentContainerIntegrations retrieves all CCAgent container integrations for an organization
func (s *CCAgentContainerIntegrationsService) ListCCAgentContainerIntegrations(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.CCAgentContainerIntegration, error) {
	log.Printf("📋 Starting to list CCAgent container integrations for org: %s", orgID)

	integrations, err := s.repo.ListCCAgentContainerIntegrations(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list CCAgent container integrations: %w", err)
	}

	log.Printf(
		"📋 Completed successfully - found %d CCAgent container integrations for org: %s",
		len(integrations),
		orgID,
	)
	return integrations, nil
}

// GetCCAgentContainerIntegrationByID retrieves a CCAgent container integration by ID
func (s *CCAgentContainerIntegrationsService) GetCCAgentContainerIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.CCAgentContainerIntegration], error) {
	log.Printf("📋 Starting to get CCAgent container integration: %s for org: %s", id, orgID)

	if !core.IsValidULID(id) {
		return mo.None[*models.CCAgentContainerIntegration](), fmt.Errorf("invalid integration ID")
	}

	integration, err := s.repo.GetCCAgentContainerIntegrationByID(ctx, orgID, id)
	if err != nil {
		return mo.None[*models.CCAgentContainerIntegration](), fmt.Errorf(
			"failed to get CCAgent container integration: %w",
			err,
		)
	}

	if integration.IsPresent() {
		log.Printf("📋 Completed successfully - found CCAgent container integration with ID: %s", id)
	} else {
		log.Printf("📋 Completed successfully - no CCAgent container integration found with ID: %s", id)
	}

	return integration, nil
}

// DeleteCCAgentContainerIntegration deletes a CCAgent container integration
func (s *CCAgentContainerIntegrationsService) DeleteCCAgentContainerIntegration(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) error {
	log.Printf("📋 Starting to delete CCAgent container integration: %s for org: %s", integrationID, orgID)

	// Validate ID
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("invalid integration ID")
	}

	// Delete the integration (repository method now handles organization scoping)
	if err := s.repo.DeleteCCAgentContainerIntegration(ctx, orgID, integrationID); err != nil {
		return fmt.Errorf("failed to delete CCAgent container integration: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted CCAgent container integration: %s", integrationID)
	return nil
}

// RedeployCCAgentContainer redeploys a CCAgent container using SSH
func (s *CCAgentContainerIntegrationsService) RedeployCCAgentContainer(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) error {
	log.Printf("📋 Starting to redeploy CCAgent container integration: %s for org: %s", integrationID, orgID)

	// Validate ID
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("invalid integration ID")
	}

	// Get CCAgent container integration
	integrationOpt, err := s.GetCCAgentContainerIntegrationByID(ctx, orgID, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get CCAgent container integration: %w", err)
	}
	if !integrationOpt.IsPresent() {
		return fmt.Errorf("CCAgent container integration not found")
	}
	integration := integrationOpt.MustGet()

	// Get organization to access CCAgent secret key
	organizationOpt, err := s.organizationsService.GetOrganizationByID(ctx, string(orgID))
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}
	if !organizationOpt.IsPresent() {
		return fmt.Errorf("organization not found")
	}
	organization := organizationOpt.MustGet()

	// CCAgentSystemSecretKey should always be present - this is a system invariant
	utils.AssertInvariant(
		organization.CCAgentSystemSecretKey != "",
		"CCAgent system secret key must be present for organization",
	)

	// Get GitHub integration for installation ID
	githubIntegrations, err := s.githubIntegrationsService.ListGitHubIntegrations(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to list GitHub integrations: %w", err)
	}
	if len(githubIntegrations) == 0 {
		return fmt.Errorf("no GitHub integration found for organization")
	}
	githubIntegration := githubIntegrations[0] // Use first integration

	// Get Anthropic integration for authentication
	anthropicIntegrations, err := s.anthropicIntegrationsService.ListAnthropicIntegrations(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to list Anthropic integrations: %w", err)
	}
	if len(anthropicIntegrations) == 0 {
		return fmt.Errorf("no Anthropic integration found for organization")
	}
	anthropicIntegration := anthropicIntegrations[0] // Use first integration

	// Validate we have at least one authentication method
	if anthropicIntegration.AnthropicAPIKey == nil && anthropicIntegration.ClaudeCodeOAuthToken == nil {
		return fmt.Errorf("anthropic integration does not have API key or OAuth token configured")
	}

	// Build the redeployccagent.sh command
	instanceName := fmt.Sprintf("ccagent-%s", integrationID)
	command := fmt.Sprintf("/root/redeployccagent.sh -n '%s' -k '%s' -r '%s' -i '%s'",
		instanceName,
		organization.CCAgentSystemSecretKey,
		integration.RepoURL,
		githubIntegration.GitHubInstallationID,
	)

	// Add authentication method
	if anthropicIntegration.AnthropicAPIKey != nil {
		command += fmt.Sprintf(" -a '%s'", *anthropicIntegration.AnthropicAPIKey)
	} else if anthropicIntegration.ClaudeCodeOAuthToken != nil {
		command += fmt.Sprintf(" -o '%s'", *anthropicIntegration.ClaudeCodeOAuthToken)
	}

	// Execute the command via SSH
	sshHost := integration.SSHHost
	log.Printf("📋 Executing redeployccagent.sh on host: %s", sshHost)
	if err := s.sshClient.ExecuteCommand(sshHost, command); err != nil {
		return fmt.Errorf("failed to execute redeployccagent.sh: %w", err)
	}

	log.Printf("📋 Completed successfully - redeployed CCAgent container integration: %s", integrationID)
	return nil
}
