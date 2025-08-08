package testutils

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
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
	_ = godotenv.Load("../../.env.test")
	_ = godotenv.Load("../.env.test")
	_ = godotenv.Load(".env.test")
	_ = godotenv.Load()

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
	cfg, err := LoadTestConfig()
	require.NoError(t, err, "Failed to load test config")

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")
	defer dbConn.Close()

	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)

	// Create organization first
	testOrgID := core.NewID("org")
	organization := &models.Organization{ID: testOrgID}
	err = organizationsRepo.CreateOrganization(context.Background(), organization)
	require.NoError(t, err, "Failed to create test organization")

	// Create user with the organization ID
	testUserID := core.NewID("u")
	testUser, err := usersRepo.CreateUser(context.Background(), "test", testUserID, testOrgID)
	require.NoError(t, err, "Failed to create test user")
	return testUser
}

// CreateTestUserWithProvider creates a test user with a specific auth provider
func CreateTestUserWithProvider(t *testing.T, usersRepo *db.PostgresUsersRepository, authProvider string) *models.User {
	cfg, err := LoadTestConfig()
	require.NoError(t, err, "Failed to load test config")

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")
	defer dbConn.Close()

	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)

	// Create organization first
	testOrgID := core.NewID("org")
	organization := &models.Organization{ID: testOrgID}
	err = organizationsRepo.CreateOrganization(context.Background(), organization)
	require.NoError(t, err, "Failed to create test organization")

	// Create user with the organization ID
	testUserID := core.NewID("u")
	testUser, err := usersRepo.CreateUser(context.Background(), authProvider, testUserID, testOrgID)
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

// CleanupTestUser creates a cleanup function that deletes a test user from the database
func CleanupTestUser(t *testing.T, dbConn *sqlx.DB, databaseSchema string, userID string) func() {
	return func() {
		query := "DELETE FROM " + databaseSchema + ".users WHERE id = $1"
		_, err := dbConn.Exec(query, userID)
		if err != nil {
			t.Logf("⚠️ Failed to cleanup test user from database: %v", err)
		} else {
			t.Logf("🧹 Cleaned up test user from database: %s", userID)
		}
	}
}
