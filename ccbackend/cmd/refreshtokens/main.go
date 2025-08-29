package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"ccbackend/clients/anthropic"
	"ccbackend/clients/github"
	"ccbackend/clients/ssh"
	"ccbackend/config"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services/anthropic_integrations"
	ccagentcontainerintegrations "ccbackend/services/ccagent_container_integrations"
	"ccbackend/services/github_integrations"
	"ccbackend/services/organizations"
)

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

	// Create database connection
	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
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
		log.Fatalf("❌ Failed to decode GitHub app private key: %v", err)
	}

	githubClient, err := github.NewGitHubClient(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.GitHubAppID, privateKey)
	if err != nil {
		log.Fatalf("❌ Failed to create GitHub client: %v", err)
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
		log.Fatalf("❌ Failed to get organizations: %v", err)
	}

	log.Printf("🔍 Found %d organizations to process", len(organizations))

	totalIntegrations := 0
	refreshedCount := 0
	errorCount := 0
	deploymentErrorCount := 0
	organizationsWithUpdates := make(map[string]bool)

	// Process each organization
	for _, org := range organizations {
		log.Printf("🏢 Processing organization: %s", org.ID)

		// Get all Anthropic integrations for this organization
		integrations, err := anthropicService.ListAnthropicIntegrations(ctx, models.OrgID(org.ID))
		if err != nil {
			log.Printf("❌ Failed to get Anthropic integrations for org %s: %v", org.ID, err)
			errorCount++
			continue
		}

		if len(integrations) == 0 {
			log.Printf("⏭️  No Anthropic integrations found for organization: %s", org.ID)
			continue
		}

		log.Printf("🔍 Found %d Anthropic integrations in org %s", len(integrations), org.ID)
		totalIntegrations += len(integrations)

		// Refresh tokens for each integration
		orgHasUpdates := false
		for _, integration := range integrations {
			if err := refreshIntegrationTokens(ctx, anthropicService, org.ID, &integration); err != nil {
				log.Printf("❌ Failed to refresh tokens for integration %s: %v", integration.ID, err)
				errorCount++
			} else {
				refreshedCount++
				orgHasUpdates = true
			}
		}

		// Track organizations that had successful token refreshes for container updates
		if orgHasUpdates {
			organizationsWithUpdates[org.ID] = true
		}

		// After refreshing tokens for this organization, update its container configurations
		if orgHasUpdates {
			if err := redeployContainersForOrg(ctx, ccagentContainerService, sshClient, org.ID); err != nil {
				log.Printf("❌ Failed to update container configurations for org %s: %v", org.ID, err)
				deploymentErrorCount++
			}
		}
	}

	// After all organizations are processed, finalize deployment for those with updates
	if len(organizationsWithUpdates) > 0 {
		log.Printf("🚀 Finalizing deployment for %d organizations with token updates...", len(organizationsWithUpdates))
		for orgID := range organizationsWithUpdates {
			if err := finalizeDeployment(ccagentContainerService, sshClient, orgID); err != nil {
				log.Printf("❌ Failed to finalize deployment for org %s: %v", orgID, err)
				deploymentErrorCount++
			}
		}
	}

	// Print summary
	log.Printf("✅ Token refresh and deployment process completed!")
	log.Printf("📊 Summary:")
	log.Printf("   - Organizations processed: %d", len(organizations))
	log.Printf("   - Total integrations found: %d", totalIntegrations)
	log.Printf("   - Tokens refreshed successfully: %d", refreshedCount)
	log.Printf("   - Token refresh errors: %d", errorCount)
	log.Printf("   - Organizations with updates: %d", len(organizationsWithUpdates))
	log.Printf("   - Deployment errors: %d", deploymentErrorCount)

	if errorCount > 0 || deploymentErrorCount > 0 {
		os.Exit(1)
	}
}

// buildRedeployAllCommand builds the SSH command for redeployall.sh
func buildRedeployAllCommand() string {
	return "/root/redeployall.sh"
}

// redeployContainersForOrg updates container configurations for all CCAgent container integrations in an organization
func redeployContainersForOrg(
	ctx context.Context,
	ccagentContainerService *ccagentcontainerintegrations.CCAgentContainerIntegrationsService,
	sshClient ssh.SSHClientInterface,
	orgID string,
) error {
	log.Printf("🔄 Updating container configurations for organization: %s", orgID)

	// Get all CCAgent container integrations for this organization
	integrations, err := ccagentContainerService.ListCCAgentContainerIntegrations(ctx, models.OrgID(orgID))
	if err != nil {
		return fmt.Errorf("failed to list CCAgent container integrations: %w", err)
	}

	if len(integrations) == 0 {
		log.Printf("⏭️  No CCAgent container integrations found for organization: %s", orgID)
		return nil
	}

	log.Printf("🔍 Found %d CCAgent container integrations to update in org %s", len(integrations), orgID)

	// Update configuration for each integration using the existing deployment logic
	for _, integration := range integrations {
		log.Printf("🔄 Updating config for CCAgent container integration: %s", integration.ID)

		// Update the configuration only (don't redeploy containers yet)
		if err := ccagentContainerService.RedeployCCAgentContainer(ctx, models.OrgID(orgID), integration.ID, true); err != nil {
			log.Printf("❌ Failed to update config for integration %s: %v", integration.ID, err)
			return fmt.Errorf("failed to update config for integration %s: %w", integration.ID, err)
		}

		log.Printf("✅ Successfully updated config for integration %s", integration.ID)
	}

	log.Printf("✅ Successfully updated container configurations for organization: %s", orgID)
	return nil
}

// finalizeDeployment runs redeployall.sh to apply all configuration changes for an organization
func finalizeDeployment(
	ccagentContainerService *ccagentcontainerintegrations.CCAgentContainerIntegrationsService,
	sshClient ssh.SSHClientInterface,
	orgID string,
) error {
	log.Printf("🚀 Finalizing deployment for organization: %s", orgID)

	// Get any CCAgent container integration to get the SSH host (they should all use the same SSH host per org)
	ctx := context.Background()
	integrations, err := ccagentContainerService.ListCCAgentContainerIntegrations(ctx, models.OrgID(orgID))
	if err != nil {
		return fmt.Errorf("failed to list CCAgent container integrations: %w", err)
	}

	if len(integrations) == 0 {
		log.Printf("⏭️  No CCAgent container integrations found for organization: %s, skipping deployment", orgID)
		return nil
	}

	// Use the SSH host from the first integration (they should all be the same per organization)
	sshHost := integrations[0].SSHHost
	command := buildRedeployAllCommand()

	log.Printf("🔄 Executing redeployall.sh on host: %s", sshHost)

	if err := sshClient.ExecuteCommand(sshHost, command); err != nil {
		return fmt.Errorf("failed to execute redeployall.sh: %w", err)
	}

	log.Printf("✅ Successfully finalized deployment for organization: %s", orgID)
	return nil
}

func refreshIntegrationTokens(
	ctx context.Context,
	service *anthropic_integrations.AnthropicIntegrationsService,
	orgID string,
	integration *models.AnthropicIntegration,
) error {
	log.Printf("🔄 Processing integration %s (org: %s)", integration.ID, orgID)

	// Check if this integration has OAuth tokens
	if integration.ClaudeCodeOAuthRefreshToken == nil || *integration.ClaudeCodeOAuthRefreshToken == "" {
		log.Printf("⏭️  Skipping integration %s - no refresh token (likely API key auth)", integration.ID)
		return nil
	}

	// Log current token status
	if integration.ClaudeCodeOAuthTokenExpiresAt != nil {
		timeUntilExpiry := time.Until(*integration.ClaudeCodeOAuthTokenExpiresAt)
		log.Printf(
			"🔄 Refreshing integration %s - current tokens expire in %v",
			integration.ID,
			timeUntilExpiry.Round(time.Minute),
		)
	} else {
		log.Printf("🔄 Refreshing integration %s - no expiration time recorded", integration.ID)
	}

	// Refresh the tokens
	_, err := service.RefreshTokens(ctx, models.OrgID(orgID), integration.ID)
	if err != nil {
		return fmt.Errorf("failed to refresh tokens: %w", err)
	}

	log.Printf("✅ Successfully refreshed tokens for integration %s", integration.ID)
	return nil
}
