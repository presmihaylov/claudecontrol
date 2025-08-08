package organizations

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/testutils"
)

func setupTestService(t *testing.T) (*OrganizationsService, *sqlx.DB, string, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	// Create repository and service
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	organizationsService := NewOrganizationsService(organizationsRepo)

	cleanup := func() {
		dbConn.Close()
	}

	return organizationsService, dbConn, cfg.DatabaseSchema, cleanup
}

func TestOrganizationsService(t *testing.T) {
	organizationsService, dbConn, databaseSchema, cleanup := setupTestService(t)
	defer cleanup()

	t.Run("CreateOrganization", func(t *testing.T) {
		// Test CreateOrganization
		organization, err := organizationsService.CreateOrganization(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, organization)
		assert.NotEmpty(t, organization.ID)
		assert.True(t, core.IsValidULID(organization.ID), "Organization ID should be a valid ULID")
		assert.Contains(t, organization.ID, "org_", "Organization ID should have org_ prefix")
		assert.NotZero(t, organization.CreatedAt)
		assert.NotZero(t, organization.UpdatedAt)

		// Cleanup - delete the created organization
		defer func() {
			_, err := dbConn.Exec("DELETE FROM "+databaseSchema+".organizations WHERE id = $1", organization.ID)
			if err != nil {
				t.Logf("⚠️ Failed to cleanup organization %s: %v", organization.ID, err)
			} else {
				t.Logf("🧹 Cleaned up organization: %s", organization.ID)
			}
		}()
	})

	t.Run("GetOrganizationByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a test organization using the service
			testOrganization, err := organizationsService.CreateOrganization(context.Background())
			require.NoError(t, err, "Failed to create test organization")

			// Cleanup function
			defer func() {
				_, err := dbConn.Exec("DELETE FROM "+databaseSchema+".organizations WHERE id = $1", testOrganization.ID)
				if err != nil {
					t.Logf("⚠️ Failed to cleanup organization %s: %v", testOrganization.ID, err)
				} else {
					t.Logf("🧹 Cleaned up organization: %s", testOrganization.ID)
				}
			}()

			// Test GetOrganizationByID with existing organization
			organizationOpt, err := organizationsService.GetOrganizationByID(context.Background(), testOrganization.ID)
			require.NoError(t, err)
			assert.True(t, organizationOpt.IsPresent(), "Organization should be found")

			organization := organizationOpt.MustGet()
			assert.Equal(t, testOrganization.ID, organization.ID)
			assert.NotZero(t, organization.CreatedAt)
			assert.NotZero(t, organization.UpdatedAt)
		})

		t.Run("NotFound", func(t *testing.T) {
			// Test GetOrganizationByID with non-existent organization
			nonExistentID := core.NewID("org")
			organizationOpt, err := organizationsService.GetOrganizationByID(context.Background(), nonExistentID)
			require.NoError(t, err)
			assert.False(t, organizationOpt.IsPresent(), "Organization should not be found")
		})

		t.Run("ValidationErrors", func(t *testing.T) {
			// Test with invalid ULID format
			organizationOpt, err := organizationsService.GetOrganizationByID(context.Background(), "invalid-id")
			assert.Error(t, err)
			assert.False(t, organizationOpt.IsPresent())
			assert.Contains(t, err.Error(), "organization ID must be a valid ULID")

			// Test with empty ID
			organizationOpt, err = organizationsService.GetOrganizationByID(context.Background(), "")
			assert.Error(t, err)
			assert.False(t, organizationOpt.IsPresent())
			assert.Contains(t, err.Error(), "organization ID must be a valid ULID")
		})
	})

	t.Run("CreateMultipleOrganizations", func(t *testing.T) {
		// Create multiple organizations
		org1, err := organizationsService.CreateOrganization(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, org1)

		org2, err := organizationsService.CreateOrganization(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, org2)

		// Ensure they have different IDs
		assert.NotEqual(t, org1.ID, org2.ID, "Organizations should have unique IDs")

		// Ensure both can be retrieved
		org1Opt, err := organizationsService.GetOrganizationByID(context.Background(), org1.ID)
		require.NoError(t, err)
		assert.True(t, org1Opt.IsPresent())

		org2Opt, err := organizationsService.GetOrganizationByID(context.Background(), org2.ID)
		require.NoError(t, err)
		assert.True(t, org2Opt.IsPresent())

		// Cleanup
		defer func() {
			for _, orgID := range []string{org1.ID, org2.ID} {
				_, err := dbConn.Exec("DELETE FROM "+databaseSchema+".organizations WHERE id = $1", orgID)
				if err != nil {
					t.Logf("⚠️ Failed to cleanup organization %s: %v", orgID, err)
				} else {
					t.Logf("🧹 Cleaned up organization: %s", orgID)
				}
			}
		}()
	})
}
