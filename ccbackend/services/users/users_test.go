package users

import (
	"context"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/db"
	organizations "ccbackend/services/organizations"
	"ccbackend/services/txmanager"
	"ccbackend/testutils"
)

// TestUserHelper provides utilities for creating and managing test users with Clerk
type TestUserHelper struct {
	clerkClient  *user.Client
	createdUsers []string // Track created user IDs for cleanup
}

// NewTestUserHelper creates a new test user helper with Clerk client
func NewTestUserHelper(t *testing.T) *TestUserHelper {
	// Load test config to get Clerk secret key
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err, "Failed to load test config")

	// Create Clerk client config
	clerkConfig := &clerk.ClientConfig{
		BackendConfig: clerk.BackendConfig{
			Key: clerk.String(cfg.ClerkSecretKey),
		},
	}

	clerkClient := user.NewClient(clerkConfig)

	return &TestUserHelper{
		clerkClient:  clerkClient,
		createdUsers: make([]string, 0),
	}
}

// CreateTestUser creates a test user via Clerk API and returns the user ID
func (h *TestUserHelper) CreateTestUser(t *testing.T, emailAddress string) string {
	ctx := context.Background()

	// Create test user with Clerk
	clerkUser, err := h.clerkClient.Create(ctx, &user.CreateParams{
		EmailAddresses:          &[]string{emailAddress},
		SkipPasswordChecks:      clerk.Bool(true),
		SkipPasswordRequirement: clerk.Bool(true),
	})
	require.NoError(t, err, "Failed to create test user via Clerk API")
	require.NotNil(t, clerkUser, "Created user should not be nil")

	// Track created user for cleanup
	h.createdUsers = append(h.createdUsers, clerkUser.ID)

	t.Logf("📋 Created test user with Clerk ID: %s, Email: %s", clerkUser.ID, emailAddress)
	return clerkUser.ID
}

// CleanupTestUsers deletes all created test users
func (h *TestUserHelper) CleanupTestUsers(t *testing.T) {
	ctx := context.Background()

	for _, userID := range h.createdUsers {
		_, err := h.clerkClient.Delete(ctx, userID)
		if err != nil {
			t.Logf("⚠️ Failed to cleanup test user %s: %v", userID, err)
		} else {
			t.Logf("🧹 Cleaned up test user: %s", userID)
		}
	}

	h.createdUsers = make([]string, 0)
}

func TestUsersService_GetOrCreateUser_WithRealClerkUser(t *testing.T) {
	// Load test config and initialize database connection
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	// Initialize repositories and services
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	organizationsService := organizations.NewOrganizationsService(organizationsRepo)
	txManager := txmanager.NewTransactionManager(dbConn)
	usersService := NewUsersService(usersRepo, organizationsService, txManager)

	// Create test user helper
	testHelper := NewTestUserHelper(t)
	defer testHelper.CleanupTestUsers(t)

	// Create a test user via Clerk API
	testEmail := "test-user-integration@example.com"
	clerkUserID := testHelper.CreateTestUser(t, testEmail)

	// Test GetOrCreateUser with the real Clerk user ID
	user, err := usersService.GetOrCreateUser(context.Background(), "clerk", clerkUserID)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "clerk", user.AuthProvider)
	assert.Equal(t, clerkUserID, user.AuthProviderID)
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.OrganizationID, "User should have an organization ID")

	// Test that calling GetOrCreateUser again returns the same user
	user2, err := usersService.GetOrCreateUser(context.Background(), "clerk", clerkUserID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, user2.ID)
	assert.Equal(t, user.AuthProviderID, user2.AuthProviderID)
	assert.Equal(t, user.OrganizationID, user2.OrganizationID, "Organization ID should be the same")

	// Cleanup database record
	defer func() {
		// Note: In a real test environment, you might want to clean up the database record too
		t.Logf("📋 Test completed for user: %s", user.ID)
	}()
}

func TestUsersService_GetOrCreateUser_BasicFunctionality(t *testing.T) {
	// Load test config and initialize database connection
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	// Initialize repositories and services
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	organizationsService := organizations.NewOrganizationsService(organizationsRepo)
	txManager := txmanager.NewTransactionManager(dbConn)
	usersService := NewUsersService(usersRepo, organizationsService, txManager)

	// Create test user using testutils
	testUser := testutils.CreateTestUserWithProvider(t, usersRepo, "test")
	defer testutils.CleanupTestUser(t, dbConn, cfg.DatabaseSchema, testUser.ID)()

	// Test GetOrCreateUser with test user
	user, err := usersService.GetOrCreateUser(context.Background(), testUser.AuthProvider, testUser.AuthProviderID)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, testUser.AuthProvider, user.AuthProvider)
	assert.Equal(t, testUser.AuthProviderID, user.AuthProviderID)
	assert.Equal(t, testUser.ID, user.ID)
	assert.Equal(t, testUser.OrganizationID, user.OrganizationID, "Should return same user with same organization ID")

	// Test that calling GetOrCreateUser again returns the same user
	user2, err := usersService.GetOrCreateUser(context.Background(), testUser.AuthProvider, testUser.AuthProviderID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, user2.ID)
	assert.Equal(t, user.AuthProviderID, user2.AuthProviderID)
	assert.Equal(t, user.OrganizationID, user2.OrganizationID, "Organization ID should remain consistent")
}

func TestUsersService_GetOrCreateUser_ValidationErrors(t *testing.T) {
	// Load test config and initialize database connection
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	// Initialize repositories and services
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	organizationsService := organizations.NewOrganizationsService(organizationsRepo)
	txManager := txmanager.NewTransactionManager(dbConn)
	usersService := NewUsersService(usersRepo, organizationsService, txManager)

	// Test with empty auth provider
	user, err := usersService.GetOrCreateUser(context.Background(), "", "test_user_id")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "auth_provider cannot be empty")

	// Test with empty auth provider ID
	user, err = usersService.GetOrCreateUser(context.Background(), "clerk", "")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "auth_provider_id cannot be empty")
}
