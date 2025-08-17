package organizations

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/testutils"
)

func setupOrganizationsTest(t *testing.T) (*OrganizationsService, *db.PostgresOrganizationsRepository, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)

	repo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	service := NewOrganizationsService(repo)

	cleanup := func() {
		dbConn.Close()
	}

	return service, repo, cleanup
}

func TestOrganizationsService_CreateOrganization(t *testing.T) {
	service, repo, cleanup := setupOrganizationsTest(t)
	defer cleanup()

	t.Run("successful organization creation", func(t *testing.T) {
		ctx := context.Background()

		organization, err := service.CreateOrganization(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, organization.ID)
		assert.True(t, core.IsValidULID(organization.ID))
		assert.False(t, organization.CreatedAt.IsZero())
		assert.False(t, organization.UpdatedAt.IsZero())
		assert.Nil(t, organization.CCAgentSecretKey)
		assert.Nil(t, organization.CCAgentSecretKeyGeneratedAt)
		assert.NotEmpty(t, organization.CCAgentSystemSecretKey)
		assert.True(
			t,
			len(organization.CCAgentSystemSecretKey) > 4 && organization.CCAgentSystemSecretKey[:4] == "sys_",
			"system secret key should have sys_ prefix",
		)

		// Verify organization was stored in database
		maybeOrg, err := repo.GetOrganizationByID(ctx, organization.ID)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())

		storedOrg := maybeOrg.MustGet()
		assert.Equal(t, organization.ID, storedOrg.ID)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, organization.ID)
		}()
	})

	t.Run("creates organization with unique ID", func(t *testing.T) {
		ctx := context.Background()

		org1, err := service.CreateOrganization(ctx)
		require.NoError(t, err)

		org2, err := service.CreateOrganization(ctx)
		require.NoError(t, err)

		assert.NotEqual(t, org1.ID, org2.ID)
		assert.True(t, core.IsValidULID(org1.ID))
		assert.True(t, core.IsValidULID(org2.ID))

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id IN ($1, $2)", cfg.DatabaseSchema)
			dbConn.Exec(query, org1.ID, org2.ID)
		}()
	})
}

func TestOrganizationsService_GetOrganizationByID(t *testing.T) {
	service, repo, cleanup := setupOrganizationsTest(t)
	defer cleanup()

	t.Run("successful organization retrieval", func(t *testing.T) {
		ctx := context.Background()

		// Create organization directly through repository for testing
		testOrgID := core.NewID("org")
		systemSecretKey, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		createdOrg := &models.Organization{
			ID:                     testOrgID,
			CCAgentSystemSecretKey: systemSecretKey,
		}
		err = repo.CreateOrganization(ctx, createdOrg)
		require.NoError(t, err)

		// Retrieve organization through service
		maybeOrg, err := service.GetOrganizationByID(ctx, testOrgID)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())

		retrievedOrg := maybeOrg.MustGet()
		assert.Equal(t, testOrgID, retrievedOrg.ID)
		assert.False(t, retrievedOrg.CreatedAt.IsZero())
		assert.False(t, retrievedOrg.UpdatedAt.IsZero())

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, testOrgID)
		}()
	})

	t.Run("organization not found", func(t *testing.T) {
		ctx := context.Background()
		nonExistentID := core.NewID("org")

		maybeOrg, err := service.GetOrganizationByID(ctx, nonExistentID)
		require.NoError(t, err)
		assert.False(t, maybeOrg.IsPresent())
	})

	t.Run("invalid organization ID", func(t *testing.T) {
		ctx := context.Background()

		maybeOrg, err := service.GetOrganizationByID(ctx, "invalid-id")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
		assert.False(t, maybeOrg.IsPresent())
	})

	t.Run("empty organization ID", func(t *testing.T) {
		ctx := context.Background()

		maybeOrg, err := service.GetOrganizationByID(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
		assert.False(t, maybeOrg.IsPresent())
	})
}

func TestOrganizationsService_GenerateCCAgentSecretKey(t *testing.T) {
	service, repo, cleanup := setupOrganizationsTest(t)
	defer cleanup()

	t.Run("successful secret key generation", func(t *testing.T) {
		ctx := context.Background()

		// Create organization first
		testOrgID := core.NewID("org")
		systemSecretKey, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		createdOrg := &models.Organization{
			ID:                     testOrgID,
			CCAgentSystemSecretKey: systemSecretKey,
		}
		err = repo.CreateOrganization(ctx, createdOrg)
		require.NoError(t, err)

		// Generate secret key
		secretKey, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(testOrgID))
		require.NoError(t, err)
		assert.NotEmpty(t, secretKey)
		assert.Greater(t, len(secretKey), 40) // Base64 encoded 32 bytes should be longer than 40 chars

		// Verify organization was updated
		maybeOrg, err := repo.GetOrganizationByID(ctx, testOrgID)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())

		updatedOrg := maybeOrg.MustGet()
		assert.NotNil(t, updatedOrg.CCAgentSecretKey)
		assert.Equal(t, secretKey, *updatedOrg.CCAgentSecretKey)
		assert.NotNil(t, updatedOrg.CCAgentSecretKeyGeneratedAt)
		assert.True(t, updatedOrg.CCAgentSecretKeyGeneratedAt.After(createdOrg.CreatedAt))

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, testOrgID)
		}()
	})

	t.Run("regenerating secret key updates existing key", func(t *testing.T) {
		ctx := context.Background()

		// Create organization first
		testOrgID := core.NewID("org")
		systemSecretKey, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		createdOrg := &models.Organization{
			ID:                     testOrgID,
			CCAgentSystemSecretKey: systemSecretKey,
		}
		err = repo.CreateOrganization(ctx, createdOrg)
		require.NoError(t, err)

		// Generate first secret key
		firstSecretKey, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(testOrgID))
		require.NoError(t, err)

		// Get the first timestamp
		maybeOrg, err := repo.GetOrganizationByID(ctx, testOrgID)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())
		firstTimestamp := *maybeOrg.MustGet().CCAgentSecretKeyGeneratedAt

		// Generate second secret key
		secondSecretKey, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(testOrgID))
		require.NoError(t, err)

		// Keys should be different
		assert.NotEqual(t, firstSecretKey, secondSecretKey)

		// Verify organization was updated
		maybeOrg, err = repo.GetOrganizationByID(ctx, testOrgID)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())

		updatedOrg := maybeOrg.MustGet()
		assert.NotNil(t, updatedOrg.CCAgentSecretKey)
		assert.Equal(t, secondSecretKey, *updatedOrg.CCAgentSecretKey)
		assert.NotNil(t, updatedOrg.CCAgentSecretKeyGeneratedAt)
		assert.True(t, updatedOrg.CCAgentSecretKeyGeneratedAt.After(firstTimestamp))

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, testOrgID)
		}()
	})

	t.Run("invalid organization ID", func(t *testing.T) {
		ctx := context.Background()

		secretKey, err := service.GenerateCCAgentSecretKey(ctx, "invalid-id")
		assert.Error(t, err)
		assert.Empty(t, secretKey)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})

	t.Run("empty organization ID", func(t *testing.T) {
		ctx := context.Background()

		secretKey, err := service.GenerateCCAgentSecretKey(ctx, "")
		assert.Error(t, err)
		assert.Empty(t, secretKey)
		assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
	})

	t.Run("non-existent organization", func(t *testing.T) {
		ctx := context.Background()
		nonExistentOrgID := core.NewID("org")

		secretKey, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(nonExistentOrgID))
		assert.Error(t, err)
		assert.Empty(t, secretKey)
		assert.Contains(t, err.Error(), "organization not found")
	})

	t.Run("generated keys are unique", func(t *testing.T) {
		ctx := context.Background()

		// Create two organizations
		org1ID := core.NewID("org")
		systemSecretKey1, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		org1 := &models.Organization{
			ID:                     org1ID,
			CCAgentSystemSecretKey: systemSecretKey1,
		}
		err = repo.CreateOrganization(ctx, org1)
		require.NoError(t, err)

		org2ID := core.NewID("org")
		systemSecretKey2, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		org2 := &models.Organization{
			ID:                     org2ID,
			CCAgentSystemSecretKey: systemSecretKey2,
		}
		err = repo.CreateOrganization(ctx, org2)
		require.NoError(t, err)

		// Generate secret keys for both organizations
		secretKey1, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(org1ID))
		require.NoError(t, err)

		secretKey2, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(org2ID))
		require.NoError(t, err)

		// Keys should be different
		assert.NotEqual(t, secretKey1, secretKey2)
		assert.NotEmpty(t, secretKey1)
		assert.NotEmpty(t, secretKey2)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id IN ($1, $2)", cfg.DatabaseSchema)
			dbConn.Exec(query, org1ID, org2ID)
		}()
	})
}

func TestOrganizationsService_GetOrganizationBySecretKey(t *testing.T) {
	service, repo, cleanup := setupOrganizationsTest(t)
	defer cleanup()

	t.Run("successful organization retrieval by secret key", func(t *testing.T) {
		ctx := context.Background()

		// Create organization and generate secret key
		testOrgID := core.NewID("org")
		systemSecretKey, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		createdOrg := &models.Organization{
			ID:                     testOrgID,
			CCAgentSystemSecretKey: systemSecretKey,
		}
		err = repo.CreateOrganization(ctx, createdOrg)
		require.NoError(t, err)

		secretKey, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(testOrgID))
		require.NoError(t, err)

		// Retrieve organization by secret key
		maybeOrg, err := service.GetOrganizationBySecretKey(ctx, secretKey)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())

		retrievedOrg := maybeOrg.MustGet()
		assert.Equal(t, testOrgID, retrievedOrg.ID)
		assert.NotNil(t, retrievedOrg.CCAgentSecretKey)
		assert.Equal(t, secretKey, *retrievedOrg.CCAgentSecretKey)
		assert.NotNil(t, retrievedOrg.CCAgentSecretKeyGeneratedAt)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, testOrgID)
		}()
	})

	t.Run("organization not found for secret key", func(t *testing.T) {
		ctx := context.Background()
		nonExistentSecretKey := "non-existent-secret-key"

		maybeOrg, err := service.GetOrganizationBySecretKey(ctx, nonExistentSecretKey)
		require.NoError(t, err)
		assert.False(t, maybeOrg.IsPresent())
	})

	t.Run("empty secret key", func(t *testing.T) {
		ctx := context.Background()

		maybeOrg, err := service.GetOrganizationBySecretKey(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret key cannot be empty")
		assert.False(t, maybeOrg.IsPresent())
	})

	t.Run("secret key from different organization", func(t *testing.T) {
		ctx := context.Background()

		// Create two organizations
		org1ID := core.NewID("org")
		systemSecretKey1, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		org1 := &models.Organization{
			ID:                     org1ID,
			CCAgentSystemSecretKey: systemSecretKey1,
		}
		err = repo.CreateOrganization(ctx, org1)
		require.NoError(t, err)

		org2ID := core.NewID("org")
		systemSecretKey2, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		org2 := &models.Organization{
			ID:                     org2ID,
			CCAgentSystemSecretKey: systemSecretKey2,
		}
		err = repo.CreateOrganization(ctx, org2)
		require.NoError(t, err)

		// Generate secret keys for both
		secretKey1, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(org1ID))
		require.NoError(t, err)

		secretKey2, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(org2ID))
		require.NoError(t, err)

		// Verify each secret key returns the correct organization
		maybeOrg1, err := service.GetOrganizationBySecretKey(ctx, secretKey1)
		require.NoError(t, err)
		require.True(t, maybeOrg1.IsPresent())
		assert.Equal(t, org1ID, maybeOrg1.MustGet().ID)

		maybeOrg2, err := service.GetOrganizationBySecretKey(ctx, secretKey2)
		require.NoError(t, err)
		require.True(t, maybeOrg2.IsPresent())
		assert.Equal(t, org2ID, maybeOrg2.MustGet().ID)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id IN ($1, $2)", cfg.DatabaseSchema)
			dbConn.Exec(query, org1ID, org2ID)
		}()
	})

	t.Run("organization without secret key", func(t *testing.T) {
		ctx := context.Background()

		// Create organization without generating a secret key
		testOrgID := core.NewID("org")
		systemSecretKey, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		createdOrg := &models.Organization{
			ID:                     testOrgID,
			CCAgentSystemSecretKey: systemSecretKey,
		}
		err = repo.CreateOrganization(ctx, createdOrg)
		require.NoError(t, err)

		// Try to find organization with random secret key
		randomSecretKey := "random-secret-key-123"
		maybeOrg, err := service.GetOrganizationBySecretKey(ctx, randomSecretKey)
		require.NoError(t, err)
		assert.False(t, maybeOrg.IsPresent())

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, testOrgID)
		}()
	})
}

func TestOrganizationsService_GetOrganizationBySecretKey_UnifiedBehavior(t *testing.T) {
	service, _, cleanup := setupOrganizationsTest(t)
	defer cleanup()

	t.Run("retrieves organization by ccagent secret key", func(t *testing.T) {
		ctx := context.Background()

		// Create organization with system secret key
		organization, err := service.CreateOrganization(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, organization.CCAgentSystemSecretKey)

		// Generate a ccagent secret key
		ccagentSecretKey, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(organization.ID))
		require.NoError(t, err)
		require.NotEmpty(t, ccagentSecretKey)
		assert.True(
			t,
			len(ccagentSecretKey) > 8 && ccagentSecretKey[:8] == "ccagent_",
			"ccagent key should have ccagent_ prefix",
		)

		// Retrieve organization using ccagent secret key
		maybeOrg, err := service.GetOrganizationBySecretKey(ctx, ccagentSecretKey)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())

		retrievedOrg := maybeOrg.MustGet()
		assert.Equal(t, organization.ID, retrievedOrg.ID)
		assert.NotNil(t, retrievedOrg.CCAgentSecretKey)
		assert.Equal(t, ccagentSecretKey, *retrievedOrg.CCAgentSecretKey)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, organization.ID)
		}()
	})

	t.Run("retrieves organization by system secret key", func(t *testing.T) {
		ctx := context.Background()

		// Create organization with auto-generated system secret key
		organization, err := service.CreateOrganization(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, organization.CCAgentSystemSecretKey)
		assert.True(
			t,
			len(organization.CCAgentSystemSecretKey) > 4 && organization.CCAgentSystemSecretKey[:4] == "sys_",
			"system key should have sys_ prefix",
		)

		// Retrieve organization using system secret key
		maybeOrg, err := service.GetOrganizationBySecretKey(ctx, organization.CCAgentSystemSecretKey)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())

		retrievedOrg := maybeOrg.MustGet()
		assert.Equal(t, organization.ID, retrievedOrg.ID)
		assert.Equal(t, organization.CCAgentSystemSecretKey, retrievedOrg.CCAgentSystemSecretKey)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, organization.ID)
		}()
	})

	t.Run("retrieves same organization with both key types", func(t *testing.T) {
		ctx := context.Background()

		// Create organization with system secret key
		organization, err := service.CreateOrganization(ctx)
		require.NoError(t, err)
		systemSecretKey := organization.CCAgentSystemSecretKey
		require.NotEmpty(t, systemSecretKey)

		// Generate ccagent secret key for the same organization
		ccagentSecretKey, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(organization.ID))
		require.NoError(t, err)
		require.NotEmpty(t, ccagentSecretKey)

		// Verify keys are different
		assert.NotEqual(t, systemSecretKey, ccagentSecretKey)

		// Retrieve organization using system secret key
		maybeOrgBySystem, err := service.GetOrganizationBySecretKey(ctx, systemSecretKey)
		require.NoError(t, err)
		require.True(t, maybeOrgBySystem.IsPresent())

		// Retrieve organization using ccagent secret key
		maybeOrgByAgent, err := service.GetOrganizationBySecretKey(ctx, ccagentSecretKey)
		require.NoError(t, err)
		require.True(t, maybeOrgByAgent.IsPresent())

		// Both should return the same organization
		orgBySystem := maybeOrgBySystem.MustGet()
		orgByAgent := maybeOrgByAgent.MustGet()
		assert.Equal(t, organization.ID, orgBySystem.ID)
		assert.Equal(t, organization.ID, orgByAgent.ID)
		assert.Equal(t, orgBySystem.ID, orgByAgent.ID)

		// Verify the organization has both keys
		assert.Equal(t, systemSecretKey, orgBySystem.CCAgentSystemSecretKey)
		assert.Equal(t, systemSecretKey, orgByAgent.CCAgentSystemSecretKey)
		assert.NotNil(t, orgByAgent.CCAgentSecretKey)
		assert.Equal(t, ccagentSecretKey, *orgByAgent.CCAgentSecretKey)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, organization.ID)
		}()
	})

	t.Run("different organizations have different secret keys", func(t *testing.T) {
		ctx := context.Background()

		// Create two organizations
		org1, err := service.CreateOrganization(ctx)
		require.NoError(t, err)
		org2, err := service.CreateOrganization(ctx)
		require.NoError(t, err)

		// Verify different system secret keys
		assert.NotEqual(t, org1.CCAgentSystemSecretKey, org2.CCAgentSystemSecretKey)

		// Generate ccagent keys for both
		ccagentKey1, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(org1.ID))
		require.NoError(t, err)
		ccagentKey2, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(org2.ID))
		require.NoError(t, err)

		// Verify different ccagent keys
		assert.NotEqual(t, ccagentKey1, ccagentKey2)

		// Test cross-authentication doesn't work
		maybeOrg, err := service.GetOrganizationBySecretKey(ctx, org1.CCAgentSystemSecretKey)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())
		assert.Equal(t, org1.ID, maybeOrg.MustGet().ID)

		maybeOrg, err = service.GetOrganizationBySecretKey(ctx, ccagentKey2)
		require.NoError(t, err)
		require.True(t, maybeOrg.IsPresent())
		assert.Equal(t, org2.ID, maybeOrg.MustGet().ID)

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id IN ($1, $2)", cfg.DatabaseSchema)
			dbConn.Exec(query, org1.ID, org2.ID)
		}()
	})

	t.Run("system secret key format validation", func(t *testing.T) {
		ctx := context.Background()

		organization, err := service.CreateOrganization(ctx)
		require.NoError(t, err)

		// Verify system secret key format
		systemKey := organization.CCAgentSystemSecretKey
		assert.True(t, len(systemKey) > 4, "system key should be longer than prefix")
		assert.True(t, systemKey[:4] == "sys_", "system key should start with sys_")
		assert.Greater(t, len(systemKey), 40, "system key should be longer than 40 chars (32 bytes base64)")

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, organization.ID)
		}()
	})

	t.Run("ccagent secret key format validation", func(t *testing.T) {
		ctx := context.Background()

		organization, err := service.CreateOrganization(ctx)
		require.NoError(t, err)

		ccagentKey, err := service.GenerateCCAgentSecretKey(ctx, models.OrgID(organization.ID))
		require.NoError(t, err)

		// Verify ccagent secret key format
		assert.True(t, len(ccagentKey) > 8, "ccagent key should be longer than prefix")
		assert.True(t, ccagentKey[:8] == "ccagent_", "ccagent key should start with ccagent_")
		assert.Greater(t, len(ccagentKey), 40, "ccagent key should be longer than 40 chars (32 bytes base64)")

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id = $1", cfg.DatabaseSchema)
			dbConn.Exec(query, organization.ID)
		}()
	})
}

func TestOrganizationsService_GetAllOrganizations(t *testing.T) {
	service, repo, cleanup := setupOrganizationsTest(t)
	defer cleanup()

	t.Run("successful retrieval of all organizations", func(t *testing.T) {
		ctx := context.Background()

		// Create multiple organizations
		org1ID := core.NewID("org")
		systemSecretKey1, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		org1 := &models.Organization{
			ID:                     org1ID,
			CCAgentSystemSecretKey: systemSecretKey1,
		}
		err = repo.CreateOrganization(ctx, org1)
		require.NoError(t, err)

		org2ID := core.NewID("org")
		systemSecretKey2, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		org2 := &models.Organization{
			ID:                     org2ID,
			CCAgentSystemSecretKey: systemSecretKey2,
		}
		err = repo.CreateOrganization(ctx, org2)
		require.NoError(t, err)

		org3ID := core.NewID("org")
		systemSecretKey3, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		org3 := &models.Organization{
			ID:                     org3ID,
			CCAgentSystemSecretKey: systemSecretKey3,
		}
		err = repo.CreateOrganization(ctx, org3)
		require.NoError(t, err)

		// Retrieve all organizations
		organizations, err := service.GetAllOrganizations(ctx)
		require.NoError(t, err)

		// Should contain at least our 3 test organizations (there might be others from other tests)
		assert.GreaterOrEqual(t, len(organizations), 3)

		// Find our test organizations in the results
		orgIDs := make(map[string]bool)
		for _, org := range organizations {
			orgIDs[org.ID] = true
			assert.True(t, core.IsValidULID(org.ID))
			assert.False(t, org.CreatedAt.IsZero())
			assert.False(t, org.UpdatedAt.IsZero())
		}

		assert.True(t, orgIDs[org1ID], "org1 should be in results")
		assert.True(t, orgIDs[org2ID], "org2 should be in results")
		assert.True(t, orgIDs[org3ID], "org3 should be in results")

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id IN ($1, $2, $3)", cfg.DatabaseSchema)
			dbConn.Exec(query, org1ID, org2ID, org3ID)
		}()
	})

	t.Run("returns empty list when no organizations exist", func(t *testing.T) {
		ctx := context.Background()

		// Clean up any existing organizations first
		cfg, err := testutils.LoadTestConfig()
		require.NoError(t, err)
		dbConn, err := db.NewConnection(cfg.DatabaseURL)
		require.NoError(t, err)
		defer dbConn.Close()

		// Delete all organizations to ensure clean state
		query := fmt.Sprintf("DELETE FROM %s.organizations", cfg.DatabaseSchema)
		_, err = dbConn.Exec(query)
		require.NoError(t, err)

		// Retrieve all organizations
		organizations, err := service.GetAllOrganizations(ctx)
		require.NoError(t, err)
		assert.Empty(t, organizations)
	})

	t.Run("organizations are ordered by created_at ASC", func(t *testing.T) {
		ctx := context.Background()

		// Create organizations one by one to ensure different creation times
		org1ID := core.NewID("org")
		systemSecretKey1, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		org1 := &models.Organization{
			ID:                     org1ID,
			CCAgentSystemSecretKey: systemSecretKey1,
		}
		err = repo.CreateOrganization(ctx, org1)
		require.NoError(t, err)

		org2ID := core.NewID("org")
		systemSecretKey2, err := core.NewSecretKey("sys")
		require.NoError(t, err)
		org2 := &models.Organization{
			ID:                     org2ID,
			CCAgentSystemSecretKey: systemSecretKey2,
		}
		err = repo.CreateOrganization(ctx, org2)
		require.NoError(t, err)

		// Retrieve all organizations
		organizations, err := service.GetAllOrganizations(ctx)
		require.NoError(t, err)

		// Should have at least 2 organizations
		assert.GreaterOrEqual(t, len(organizations), 2)

		// Find our organizations in results and verify ordering
		var org1Idx, org2Idx int = -1, -1
		for i, org := range organizations {
			if org.ID == org1ID {
				org1Idx = i
			}
			if org.ID == org2ID {
				org2Idx = i
			}
		}

		assert.NotEqual(t, -1, org1Idx, "org1 should be found")
		assert.NotEqual(t, -1, org2Idx, "org2 should be found")
		assert.Less(t, org1Idx, org2Idx, "org1 should come before org2 (created first)")

		// Verify the ordering is consistent with creation time
		if org1Idx < len(organizations) && org2Idx < len(organizations) {
			assert.True(t, organizations[org1Idx].CreatedAt.Before(organizations[org2Idx].CreatedAt) ||
				organizations[org1Idx].CreatedAt.Equal(organizations[org2Idx].CreatedAt))
		}

		// Clean up
		defer func() {
			cfg, _ := testutils.LoadTestConfig()
			dbConn, _ := db.NewConnection(cfg.DatabaseURL)
			defer dbConn.Close()
			query := fmt.Sprintf("DELETE FROM %s.organizations WHERE id IN ($1, $2)", cfg.DatabaseSchema)
			dbConn.Exec(query, org1ID, org2ID)
		}()
	})
}
