package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/samber/lo"

	"ccbackend/clients/anthropic"
	"ccbackend/clients/github"
	"ccbackend/clients/ssh"
	"ccbackend/config"
	"ccbackend/db"
	"ccbackend/middleware"
	"ccbackend/models"
	"ccbackend/services/anthropic_integrations"
	ccagentcontainerintegrations "ccbackend/services/ccagent_container_integrations"
	"ccbackend/services/github_integrations"
	"ccbackend/services/organizations"
)

type OrgProcessingResults struct {
	HasUpdates bool
}

func main() {
	log.Printf("🔄 Starting Anthropic OAuth token refresh process...")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Initialize error alert middleware
	alertMiddleware := middleware.NewErrorAlertMiddleware(middleware.SlackAlertConfig{
		WebhookURL:  cfg.SlackAlertWebhookURL,
		Environment: cfg.Environment,
		AppName:     "ccbackend-refreshtokens",
		LogsURL:     cfg.ServerLogsURL,
	})

	// Wrap the entire run method with error alerting
	wrappedRun := alertMiddleware.WrapBackgroundTask("RefreshTokensProcess", run)
	if err := wrappedRun(); err != nil {
		log.Printf("❌ Fatal error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create database connection
	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbConn.Close()

	// Initialize repositories
	anthropicRepo := db.NewPostgresAnthropicIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	githubRepo := db.NewPostgresGitHubIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	ccagentContainerRepo := db.NewPostgresCCAgentContainerIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Initialize clients
	anthropicClient := anthropic.NewAnthropicClient()

	// Decode base64 GitHub app private key
	privateKey, err := base64.StdEncoding.DecodeString(cfg.GitHubAppPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to decode GitHub app private key: %w", err)
	}

	githubClient, err := github.NewGitHubClient(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.GitHubAppID, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	sshClient := ssh.NewSSHClient(cfg.SSHPrivateKeyBase64)

	// Initialize services
	organizationsService := organizations.NewOrganizationsService(organizationsRepo)
	anthropicService := anthropic_integrations.NewAnthropicIntegrationsService(anthropicRepo, anthropicClient)
	githubService := github_integrations.NewGitHubIntegrationsService(githubRepo, githubClient)
	ccagentContainerService := ccagentcontainerintegrations.NewCCAgentContainerIntegrationsService(
		ccagentContainerRepo,
		cfg,
		githubService,
		anthropicService,
		organizationsService,
		sshClient,
	)

	ctx := context.Background()

	// Get all organizations (we need to fetch integrations per org)
	organizations, err := organizationsService.GetAllOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get organizations: %w", err)
	}

	log.Printf("🔍 Found %d organizations to process", len(organizations))

	organizationsWithUpdates := make(map[string]bool)

	// Process each organization
	for _, org := range organizations {
		orgResults, err := refreshTokensForOrg(ctx, org.ID, anthropicService, ccagentContainerService)
		if err != nil {
			return fmt.Errorf("failed to process organization %s: %w", org.ID, err)
		}

		// Track organizations that had successful token refreshes for container updates
		if orgResults.HasUpdates {
			organizationsWithUpdates[org.ID] = true
		}
	}

	// After all organizations are processed, finalize deployment for unique SSH hosts
	if len(organizationsWithUpdates) > 0 {
		orgIDs := lo.Keys(organizationsWithUpdates)
		if err := redeployAllImpactedCCAgents(ctx, ccagentContainerService, sshClient, orgIDs); err != nil {
			return fmt.Errorf("failed to finalize deployment: %w", err)
		}
	}

	// Print summary
	log.Printf("✅ Token refresh and deployment process completed successfully!")
	log.Printf("📊 Organizations processed: %d, Organizations with updates: %d", len(organizations), len(organizationsWithUpdates))

	return nil
}

// refreshTokensForOrg processes token refresh and container updates for a single organization
func refreshTokensForOrg(
	ctx context.Context,
	orgID string,
	anthropicService *anthropic_integrations.AnthropicIntegrationsService,
	ccagentContainerService *ccagentcontainerintegrations.CCAgentContainerIntegrationsService,
) (*OrgProcessingResults, error) {
	log.Printf("🏢 Processing organization: %s", orgID)

	results := &OrgProcessingResults{}

	// Get all Anthropic integrations for this organization
	integrations, err := anthropicService.ListAnthropicIntegrations(ctx, models.OrgID(orgID))
	if err != nil {
		return nil, fmt.Errorf("failed to get Anthropic integrations for org %s: %w", orgID, err)
	}

	if len(integrations) == 0 {
		log.Printf("⏭️  No Anthropic integrations found for organization: %s", orgID)
		return results, nil
	}

	log.Printf("🔍 Found %d Anthropic integrations in organization %s", len(integrations), orgID)

	// Refresh tokens for each integration
	for _, integration := range integrations {
		if err := refreshIntegrationTokens(ctx, anthropicService, orgID, &integration); err != nil {
			return nil, fmt.Errorf("failed to refresh tokens for integration %s in org %s: %w", integration.ID, orgID, err)
		}
		log.Printf("✅ Successfully refreshed tokens for integration %s in organization %s", integration.ID, orgID)
		results.HasUpdates = true
	}

	// After refreshing tokens for this organization, update its container configurations
	if results.HasUpdates {
		log.Printf("🔄 Updating container configurations for organization: %s", orgID)
		if err := updateRemoteContainerConfig(ctx, ccagentContainerService, orgID); err != nil {
			return nil, fmt.Errorf("failed to update container configurations for org %s: %w", orgID, err)
		}
		log.Printf("✅ Successfully updated container configurations for organization: %s", orgID)
	}

	log.Printf("✅ Successfully completed processing for organization: %s", orgID)
	return results, nil
}

// updateRemoteContainerConfig updates container configurations for all CCAgent container integrations in an organization
func updateRemoteContainerConfig(
	ctx context.Context,
	ccagentContainerService *ccagentcontainerintegrations.CCAgentContainerIntegrationsService,
	orgID string,
) error {
	log.Printf("🔄 Updating container configurations for organization: %s", orgID)
	integrations, err := ccagentContainerService.ListCCAgentContainerIntegrations(ctx, models.OrgID(orgID))
	if err != nil {
		return fmt.Errorf("failed to list CCAgent container integrations: %w", err)
	}

	if len(integrations) == 0 {
		log.Printf("⏭️  No CCAgent container integrations found for organization: %s", orgID)
		return nil
	}

	log.Printf("🔍 Found %d CCAgent container integrations to update in org %s", len(integrations), orgID)
	for _, integration := range integrations {
		log.Printf("🔄 Updating config for CCAgent container integration: %s", integration.ID)

		updateConfigOnly := true
		if err := ccagentContainerService.RedeployCCAgentContainer(ctx, models.OrgID(orgID), integration.ID, updateConfigOnly); err != nil {
			log.Printf("❌ Failed to update config for integration %s: %v", integration.ID, err)
			return fmt.Errorf("failed to update config for integration %s: %w", integration.ID, err)
		}

		log.Printf("✅ Successfully updated config for integration %s", integration.ID)
	}

	log.Printf("✅ Successfully updated container configurations for organization: %s", orgID)
	return nil
}

// redeployAllImpactedCCAgents runs redeployall.sh on unique SSH hosts for multiple organizations
func redeployAllImpactedCCAgents(
	ctx context.Context,
	ccagentContainerService *ccagentcontainerintegrations.CCAgentContainerIntegrationsService,
	sshClient ssh.SSHClientInterface,
	orgIDs []string,
) error {
	log.Printf("🚀 Finalizing deployment for %d organizations with token updates...", len(orgIDs))

	// Collect all SSH hosts from organizations with updates
	var allSSHHosts []string
	for _, orgID := range orgIDs {
		integrations, err := ccagentContainerService.ListCCAgentContainerIntegrations(ctx, models.OrgID(orgID))
		if err != nil {
			return fmt.Errorf("failed to get SSH hosts for org %s: %w", orgID, err)
		}

		sshHosts := lo.FilterMap(integrations, func(integration models.CCAgentContainerIntegration, _ int) (string, bool) {
			return integration.SSHHost, integration.SSHHost != ""
		})
		allSSHHosts = append(allSSHHosts, sshHosts...)
	}

	// Get unique SSH hosts and run redeployall.sh on each
	uniqueSSHHosts := lo.Uniq(allSSHHosts)
	for _, sshHost := range uniqueSSHHosts {
		log.Printf("🔄 Executing redeployall.sh on SSH host: %s", sshHost)
		command := "/root/redeployall.sh"
		if err := sshClient.ExecuteCommand(sshHost, command); err != nil {
			return fmt.Errorf("failed to execute redeployall.sh on host %s: %w", sshHost, err)
		}
		log.Printf("✅ Successfully executed redeployall.sh on SSH host: %s", sshHost)
	}

	log.Printf("✅ Successfully finalized deployment on %d unique SSH hosts", len(uniqueSSHHosts))
	return nil
}

func refreshIntegrationTokens(
	ctx context.Context,
	service *anthropic_integrations.AnthropicIntegrationsService,
	orgID string,
	integration *models.AnthropicIntegration,
) error {
	log.Printf("🔄 Processing integration %s in organization %s", integration.ID, orgID)

	// Check if this integration has OAuth tokens
	if integration.ClaudeCodeOAuthRefreshToken == nil || *integration.ClaudeCodeOAuthRefreshToken == "" {
		log.Printf("⏭️  Skipping integration %s in org %s - no refresh token (likely API key auth)", integration.ID, orgID)
		return nil
	}

	// Log current token status
	if integration.ClaudeCodeOAuthTokenExpiresAt != nil {
		timeUntilExpiry := time.Until(*integration.ClaudeCodeOAuthTokenExpiresAt)
		log.Printf(
			"🔄 Refreshing tokens for integration %s in org %s - current tokens expire in %v",
			integration.ID,
			orgID,
			timeUntilExpiry.Round(time.Minute),
		)
	} else {
		log.Printf("🔄 Refreshing tokens for integration %s in org %s - no expiration time recorded", integration.ID, orgID)
	}

	// Refresh the tokens
	_, err := service.RefreshTokens(ctx, models.OrgID(orgID), integration.ID)
	if err != nil {
		return fmt.Errorf("failed to refresh tokens for integration %s in org %s: %w", integration.ID, orgID, err)
	}

	log.Printf("✅ Successfully refreshed tokens for integration %s in org %s", integration.ID, orgID)
	return nil
}
