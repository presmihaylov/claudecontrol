package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"ccbackend/clients/anthropic"
	"ccbackend/config"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services/anthropic_integrations"
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

	// Initialize services
	anthropicRepo := db.NewPostgresAnthropicIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	anthropicClient := anthropic.NewAnthropicClient()
	anthropicService := anthropic_integrations.NewAnthropicIntegrationsService(anthropicRepo, anthropicClient)

	ctx := context.Background()

	// Get all organizations (we need to fetch integrations per org)
	orgsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	organizations, err := orgsRepo.GetAllOrganizations(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get organizations: %v", err)
	}

	log.Printf("🔍 Found %d organizations to process", len(organizations))

	totalIntegrations := 0
	refreshedCount := 0
	errorCount := 0

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
		for _, integration := range integrations {
			if err := refreshIntegrationTokens(ctx, anthropicService, org.ID, &integration); err != nil {
				log.Printf("❌ Failed to refresh tokens for integration %s: %v", integration.ID, err)
				errorCount++
			} else {
				refreshedCount++
			}
		}
	}

	// Print summary
	log.Printf("✅ Token refresh process completed!")
	log.Printf("📊 Summary:")
	log.Printf("   - Organizations processed: %d", len(organizations))
	log.Printf("   - Total integrations found: %d", totalIntegrations)
	log.Printf("   - Tokens refreshed successfully: %d", refreshedCount)
	log.Printf("   - Errors encountered: %d", errorCount)

	if errorCount > 0 {
		os.Exit(1)
	}
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
