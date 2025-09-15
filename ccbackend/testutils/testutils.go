package testutils

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
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
		ClerkConfig: config.ClerkConfig{
			SecretKey: clerkSecretKey,
		},
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
	systemSecretKey, err := core.NewSecretKey("sys")
	require.NoError(t, err, "Failed to generate system secret key")
	organization := &models.Organization{
		ID:                     testOrgID,
		CCAgentSystemSecretKey: systemSecretKey,
	}
	err = organizationsRepo.CreateOrganization(context.Background(), organization)
	require.NoError(t, err, "Failed to create test organization")

	// Create user with the organization ID
	testUserID := core.NewID("u")
	testUser, err := usersRepo.CreateUser(
		context.Background(),
		"test",
		testUserID,
		"test@example.com",
		models.OrgID(testOrgID),
	)
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
	systemSecretKey, err := core.NewSecretKey("sys")
	require.NoError(t, err, "Failed to generate system secret key")
	organization := &models.Organization{
		ID:                     testOrgID,
		CCAgentSystemSecretKey: systemSecretKey,
	}
	err = organizationsRepo.CreateOrganization(context.Background(), organization)
	require.NoError(t, err, "Failed to create test organization")

	// Create user with the organization ID
	testUserID := core.NewID("u")
	testUser, err := usersRepo.CreateUser(
		context.Background(),
		authProvider,
		testUserID,
		"test@example.com",
		models.OrgID(testOrgID),
	)
	require.NoError(t, err, "Failed to create test user with provider %s", authProvider)
	return testUser
}

// CreateTestContext creates a context with the given user set for testing
func CreateTestContext(user *models.User) context.Context {
	ctx := context.Background()
	return appctx.SetUser(ctx, user)
}

// CreateTestContextWithUser creates a context with the given user and their organization set for testing
func CreateTestContextWithUser(user *models.User) context.Context {
	ctx := context.Background()
	ctx = appctx.SetUser(ctx, user)

	// Get the user's organization and add it to context
	// For tests, we need to create or fetch the organization
	org := &models.Organization{
		ID: string(user.OrgID),
	}
	ctx = appctx.SetOrganization(ctx, org)

	return ctx
}

// CreateTestSlackIntegration creates a test slack integration model for testing
func CreateTestSlackIntegration(orgID models.OrgID) *models.SlackIntegration {
	integrationID := core.NewID("si")
	teamIDSuffix := core.NewID("team")
	tokenSuffix := core.NewID("tok")

	return &models.SlackIntegration{
		ID:             integrationID,
		SlackTeamID:    "test-team-" + teamIDSuffix,
		SlackAuthToken: "xoxb-test-token-" + tokenSuffix,
		SlackTeamName:  "Test Team",
		OrgID:          orgID,
	}
}

// CreateTestDiscordIntegration creates a test discord integration model for testing
func CreateTestDiscordIntegration(orgID models.OrgID) *models.DiscordIntegration {
	integrationID := core.NewID("di")
	guildIDSuffix := core.NewID("guild")

	return &models.DiscordIntegration{
		ID:               integrationID,
		DiscordGuildID:   "test-guild-" + guildIDSuffix,
		DiscordGuildName: "Test Discord Guild",
		OrgID:            orgID,
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

// GenerateDiscordMessageID generates a random Discord message ID
func GenerateDiscordMessageID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("msg-%d", n.Int64()+100000)
}

// GenerateDiscordChannelID generates a random Discord channel ID
func GenerateDiscordChannelID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("channel-%d", n.Int64()+100000)
}

// GenerateDiscordThreadID generates a random Discord thread ID
func GenerateDiscordThreadID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("thread-%d", n.Int64()+100000)
}

// GenerateDiscordUserID generates a random Discord user ID
func GenerateDiscordUserID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("user-%d", n.Int64()+100000)
}

// GenerateDiscordIntegrationID generates a random Discord integration ID using ULID
func GenerateDiscordIntegrationID() string {
	return core.NewID("di")
}

// GenerateOrganizationID generates a random organization ID using ULID
func GenerateOrganizationID() string {
	return core.NewID("org")
}

// GenerateOrgID generates a random organization ID using ULID and returns models.OrgID type
func GenerateOrgID() models.OrgID {
	return models.OrgID(core.NewID("org"))
}

// GenerateDiscordGuildID generates a random Discord guild ID
func GenerateDiscordGuildID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("guild-%d", n.Int64()+100000)
}

// GenerateAgentID generates a random agent ID using ULID
func GenerateAgentID() string {
	return core.NewID("ag")
}

// GenerateWSConnectionID generates a random WebSocket connection ID using ULID
func GenerateWSConnectionID() string {
	return core.NewID("ws")
}

// GenerateClientID generates a random client ID using ULID
func GenerateClientID() string {
	return core.NewID("client")
}

// GenerateJobID generates a random job ID using ULID
func GenerateJobID() string {
	return core.NewID("j")
}

// GenerateProcessedMessageID generates a random processed message ID using ULID
func GenerateProcessedMessageID() string {
	return core.NewID("pm")
}

// GenerateDiscordBotID generates a random Discord bot ID
func GenerateDiscordBotID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("bot-%d", n.Int64()+100000)
}

// GenerateDiscordBotUsername generates a random Discord bot username
func GenerateDiscordBotUsername() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(9000))
	return fmt.Sprintf("testbot%d", n.Int64()+1000)
}

// GenerateSlackIntegrationID generates a random Slack integration ID using ULID
func GenerateSlackIntegrationID() string {
	return core.NewID("si")
}

// GenerateSlackChannelID generates a random Slack channel ID
func GenerateSlackChannelID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("C%d", n.Int64()+100000)
}

// GenerateSlackUserID generates a random Slack user ID
func GenerateSlackUserID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("U%d", n.Int64()+100000)
}

// GenerateSlackThreadTS generates a random Slack thread timestamp
func GenerateSlackThreadTS() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("1234567%d.123", n.Int64()+100000)
}

// GenerateSlackMessageID generates a random Slack message ID using ULID
func GenerateSlackMessageID() string {
	return core.NewID("sm")
}

// GenerateSlackToken generates a random Slack token
func GenerateSlackToken() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("xoxb-test-token-%d", n.Int64()+100000)
}

// CreateTestOrganization creates a test organization in the database
func CreateTestOrganization(t *testing.T, organizationsRepo *db.PostgresOrganizationsRepository) *models.Organization {
	testOrgID := core.NewID("org")
	systemSecretKey, err := core.NewSecretKey("sys")
	require.NoError(t, err, "Failed to generate system secret key")

	organization := &models.Organization{
		ID:                     testOrgID,
		CCAgentSystemSecretKey: systemSecretKey,
	}

	err = organizationsRepo.CreateOrganization(context.Background(), organization)
	require.NoError(t, err, "Failed to create test organization")

	return organization
}

// CreateTestProcessedSlackMessage creates a test processed Slack message
func CreateTestProcessedSlackMessage(
	jobID string,
	orgID models.OrgID,
	slackIntegrationID string,
	status models.ProcessedSlackMessageStatus,
) *models.ProcessedSlackMessage {
	return &models.ProcessedSlackMessage{
		ID:                 core.NewID("psm"),
		JobID:              jobID,
		SlackChannelID:     GenerateSlackChannelID(),
		SlackTS:            GenerateSlackThreadTS(),
		TextContent:        "Test message content",
		Status:             status,
		SlackIntegrationID: slackIntegrationID,
		OrgID:              orgID,
	}
}

// CreateTestProcessedDiscordMessage creates a test processed Discord message
func CreateTestProcessedDiscordMessage(
	jobID string,
	orgID models.OrgID,
	discordIntegrationID string,
	status models.ProcessedDiscordMessageStatus,
) *models.ProcessedDiscordMessage {
	return &models.ProcessedDiscordMessage{
		ID:                   core.NewID("pdm"),
		JobID:                jobID,
		DiscordMessageID:     GenerateDiscordMessageID(),
		DiscordThreadID:      GenerateDiscordThreadID(),
		TextContent:          "Test message content",
		Status:               status,
		DiscordIntegrationID: discordIntegrationID,
		OrgID:                orgID,
	}
}
