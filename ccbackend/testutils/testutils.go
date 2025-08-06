package testutils

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"ccbackend/appctx"
	"ccbackend/config"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

// LoadTestConfig loads configuration for tests from environment variables
func LoadTestConfig() (*config.AppConfig, error) {
	// Try to load environment variables from various possible locations
	_ = godotenv.Load("../.env.test") // From services/ directory
	_ = godotenv.Load(".env.test")    // From root directory
	_ = godotenv.Load()               // Default .env file

	databaseURL := os.Getenv("DB_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DB_URL is not set")
	}

	databaseSchema := os.Getenv("DB_SCHEMA")
	if databaseSchema == "" {
		return nil, fmt.Errorf("DB_SCHEMA is not set")
	}

	clerkSecretKey := os.Getenv("CLERK_SECRET_KEY")
	if clerkSecretKey == "" {
		return nil, fmt.Errorf("CLERK_SECRET_KEY is not set")
	}

	return &config.AppConfig{
		DatabaseURL:    databaseURL,
		DatabaseSchema: databaseSchema,
		ClerkSecretKey: clerkSecretKey,
	}, nil
}

// CreateTestUser creates a test user with a unique ID to avoid constraint violations
func CreateTestUser(t *testing.T, usersRepo *db.PostgresUsersRepository) *models.User {
	testUserID := core.NewID("u")
	testUser, err := usersRepo.GetOrCreateUser(context.Background(), "test", testUserID)
	require.NoError(t, err, "Failed to create test user")
	return testUser
}

// CreateTestUserWithProvider creates a test user with a specific auth provider
func CreateTestUserWithProvider(t *testing.T, usersRepo *db.PostgresUsersRepository, authProvider string) *models.User {
	testUserID := core.NewID("u")
	testUser, err := usersRepo.GetOrCreateUser(context.Background(), authProvider, testUserID)
	require.NoError(t, err, "Failed to create test user with provider %s", authProvider)
	return testUser
}

// CreateTestContext creates a context with the given user set for testing
func CreateTestContext(user *models.User) context.Context {
	ctx := context.Background()
	return appctx.SetUser(ctx, user)
}

// CreateTestSlackIntegration creates a test slack integration model for testing
func CreateTestSlackIntegration(userID string) *models.SlackIntegration {
	integrationID := core.NewID("si")
	teamIDSuffix := core.NewID("team")
	tokenSuffix := core.NewID("tok")

	return &models.SlackIntegration{
		ID:             integrationID,
		SlackTeamID:    "test-team-" + teamIDSuffix,
		SlackAuthToken: "xoxb-test-token-" + tokenSuffix,
		SlackTeamName:  "Test Team",
		UserID:         userID,
	}
}
