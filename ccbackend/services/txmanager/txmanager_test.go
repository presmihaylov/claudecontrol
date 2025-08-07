package txmanager

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	dbtx "ccbackend/db/tx"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/testutils"
)

func setupTransactionTest(
	t *testing.T,
) (services.TransactionManager, *db.PostgresJobsRepository, *models.SlackIntegration, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	// Create transaction manager
	txManager := NewTransactionManager(dbConn)

	// Create repositories
	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Create test user and slack integration
	testUser := testutils.CreateTestUser(t, usersRepo)
	testIntegration := testutils.CreateTestSlackIntegration(testUser.ID)
	err = slackIntegrationsRepo.CreateSlackIntegration(context.Background(), testIntegration)
	require.NoError(t, err, "Failed to create test slack integration")

	cleanup := func() {
		// Clean up test data
		slackIntegrationsRepo.DeleteSlackIntegrationByID(context.Background(), testIntegration.ID, testUser.ID)
		dbConn.Close()
	}

	return txManager, jobsRepo, testIntegration, cleanup
}

func TestTransactionManager_WithTransaction_Success(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()

	var createdJob *models.Job

	// Execute transaction that should succeed
	err := txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Create a job within the transaction
		job := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-ts",
			SlackChannelID:     "test-channel",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJob(ctx, job); err != nil {
			return err
		}

		createdJob = job
		return nil
	})

	// Transaction should succeed
	require.NoError(t, err)
	require.NotNil(t, createdJob)

	// Job should exist in database after transaction commit
	maybeJob, err := jobsRepo.GetJobByID(ctx, createdJob.ID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, createdJob.ID, maybeJob.OrEmpty().ID)
	assert.Equal(t, createdJob.SlackThreadTS, maybeJob.OrEmpty().SlackThreadTS)

	// Clean up
	jobsRepo.DeleteJob(ctx, createdJob.ID, testIntegration.ID)
}

func TestTransactionManager_WithTransaction_Rollback_OnError(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()

	var jobID string

	// Execute transaction that should fail and rollback
	err := txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Create a job within the transaction
		job := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-ts-rollback",
			SlackChannelID:     "test-channel",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJob(ctx, job); err != nil {
			return err
		}

		jobID = job.ID

		// Return an error to trigger rollback
		return errors.New("intentional error to trigger rollback")
	})

	// Transaction should fail with our error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "intentional error to trigger rollback")

	// Job should NOT exist in database after rollback
	maybeJob1, err := jobsRepo.GetJobByID(ctx, jobID, testIntegration.ID)
	require.NoError(t, err)
	require.True(t, maybeJob1.IsAbsent(), "Job1 should not exist after rollback")
}

func TestTransactionManager_WithTransaction_Rollback_OnPanic(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()

	var jobID string

	// Execute transaction that should panic and rollback
	func() {
		defer func() {
			// Catch the panic
			r := recover()
			require.NotNil(t, r, "Expected panic")
			assert.Equal(t, "intentional panic to test rollback", r)
		}()

		txManager.WithTransaction(ctx, func(ctx context.Context) error {
			// Create a job within the transaction
			job := &models.Job{
				ID:                 core.NewID("j"),
				SlackThreadTS:      "test-thread-ts-panic",
				SlackChannelID:     "test-channel",
				SlackUserID:        "test-user",
				SlackIntegrationID: testIntegration.ID,
			}

			if err := jobsRepo.CreateJob(ctx, job); err != nil {
				return err
			}

			jobID = job.ID

			// Trigger panic
			panic("intentional panic to test rollback")
		})
	}()

	// Job should NOT exist in database after panic rollback
	maybeJob1, err := jobsRepo.GetJobByID(ctx, jobID, testIntegration.ID)
	require.NoError(t, err)
	require.True(t, maybeJob1.IsAbsent(), "Job1 should not exist after rollback")
}

func TestTransactionManager_WithTransaction_MultipleDatabaseOperations(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()

	var job1ID, job2ID string

	// Test multiple operations within single transaction
	err := txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Create first job
		job1 := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-1",
			SlackChannelID:     "test-channel-1",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJob(ctx, job1); err != nil {
			return err
		}
		job1ID = job1.ID

		// Create second job
		job2 := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-2",
			SlackChannelID:     "test-channel-2",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJob(ctx, job2); err != nil {
			return err
		}
		job2ID = job2.ID

		// Verify both jobs exist within transaction
		_, err := jobsRepo.GetJobByID(ctx, job1ID, testIntegration.ID)
		if err != nil {
			return fmt.Errorf("job1 should exist within transaction: %w", err)
		}

		_, err = jobsRepo.GetJobByID(ctx, job2ID, testIntegration.ID)
		if err != nil {
			return fmt.Errorf("job2 should exist within transaction: %w", err)
		}

		return nil
	})

	// Transaction should succeed
	require.NoError(t, err)

	// Both jobs should exist after commit
	maybeJob1, err := jobsRepo.GetJobByID(ctx, job1ID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-thread-1", maybeJob1.OrEmpty().SlackThreadTS)

	maybeJob2, err := jobsRepo.GetJobByID(ctx, job2ID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-thread-2", maybeJob2.OrEmpty().SlackThreadTS)

	// Clean up
	jobsRepo.DeleteJob(ctx, job1ID, testIntegration.ID)
	jobsRepo.DeleteJob(ctx, job2ID, testIntegration.ID)
}

func TestTransactionManager_WithTransaction_MultipleDatabaseOperations_PartialRollback(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()

	var job1ID, job2ID string

	// Test rollback of multiple operations
	err := txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Create first job
		job1 := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-rollback-1",
			SlackChannelID:     "test-channel-1",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJob(ctx, job1); err != nil {
			return err
		}
		job1ID = job1.ID

		// Create second job
		job2 := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-rollback-2",
			SlackChannelID:     "test-channel-2",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJob(ctx, job2); err != nil {
			return err
		}
		job2ID = job2.ID

		// Fail after both operations
		return errors.New("rollback both operations")
	})

	// Transaction should fail
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rollback both operations")

	// Neither job should exist after rollback
	maybeJob1, err := jobsRepo.GetJobByID(ctx, job1ID, testIntegration.ID)
	require.NoError(t, err)
	require.True(t, maybeJob1.IsAbsent(), "Job1 should not exist after rollback")

	maybeJob2, err := jobsRepo.GetJobByID(ctx, job2ID, testIntegration.ID)
	require.NoError(t, err)
	require.True(t, maybeJob2.IsAbsent(), "Job2 should not exist after rollback")
}

func TestTransactionManager_NestedTransactions(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()

	var jobID string

	// Test nested transaction support
	err := txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Create job in outer transaction
		job := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-nested-thread",
			SlackChannelID:     "test-channel",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJob(ctx, job); err != nil {
			return err
		}
		jobID = job.ID

		// Nested transaction (should reuse existing transaction)
		return txManager.WithTransaction(ctx, func(nestedCtx context.Context) error {
			// Verify job exists within nested context
			_, err := jobsRepo.GetJobByID(nestedCtx, jobID, testIntegration.ID)
			if err != nil {
				return fmt.Errorf("job should exist in nested transaction: %w", err)
			}

			// Update job within nested transaction
			return jobsRepo.UpdateJobTimestamp(nestedCtx, jobID, testIntegration.ID)
		})
	})

	// Both transactions should succeed
	require.NoError(t, err)

	// Job should exist after both transactions
	maybeJob, err := jobsRepo.GetJobByID(ctx, jobID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-nested-thread", maybeJob.OrEmpty().SlackThreadTS)

	// Clean up
	jobsRepo.DeleteJob(ctx, jobID, testIntegration.ID)
}

func TestTransactionManager_ManualTransaction_Success(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()

	// Begin manual transaction
	txCtx, err := txManager.BeginTransaction(ctx)
	require.NoError(t, err)

	// Create job within manual transaction
	job := &models.Job{
		ID:                 core.NewID("j"),
		SlackThreadTS:      "test-manual-thread",
		SlackChannelID:     "test-channel",
		SlackUserID:        "test-user",
		SlackIntegrationID: testIntegration.ID,
	}

	err = jobsRepo.CreateJob(txCtx, job)
	require.NoError(t, err)

	// Commit manual transaction
	err = txManager.CommitTransaction(txCtx)
	require.NoError(t, err)

	// Job should exist after commit
	maybeRetrievedJob, err := jobsRepo.GetJobByID(ctx, job.ID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, job.SlackThreadTS, maybeRetrievedJob.OrEmpty().SlackThreadTS)

	// Clean up
	jobsRepo.DeleteJob(ctx, job.ID, testIntegration.ID)
}

func TestTransactionManager_ManualTransaction_Rollback(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()

	// Begin manual transaction
	txCtx, err := txManager.BeginTransaction(ctx)
	require.NoError(t, err)

	// Create job within manual transaction
	job := &models.Job{
		ID:                 core.NewID("j"),
		SlackThreadTS:      "test-manual-rollback-thread",
		SlackChannelID:     "test-channel",
		SlackUserID:        "test-user",
		SlackIntegrationID: testIntegration.ID,
	}

	err = jobsRepo.CreateJob(txCtx, job)
	require.NoError(t, err)

	// Rollback manual transaction
	err = txManager.RollbackTransaction(txCtx)
	require.NoError(t, err)

	// Job should NOT exist after rollback
	maybeJob1, err := jobsRepo.GetJobByID(ctx, job.ID, testIntegration.ID)
	require.NoError(t, err)
	require.True(t, maybeJob1.IsAbsent(), "Job1 should not exist after rollback")
}

// Test context propagation functions
func TestTransactionFromContext(t *testing.T) {
	ctx := context.Background()

	// Test that no transaction exists initially
	tx, ok := dbtx.TransactionFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, tx)

	// Test with mock transaction context
	// Note: We can't test the internal context key directly since it's not exported
	// This test validates the concept but in practice we use the exported functions
}

func TestGetTransactional_ReturnsTransaction_WhenInTransactionContext(t *testing.T) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	ctx := context.Background()

	// Test without transaction - should return db
	transactional := dbtx.GetTransactional(ctx, dbConn)
	assert.Equal(t, dbConn, transactional)

	// Test with transaction context
	tx, err := dbConn.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	txCtx := dbtx.WithTransaction(ctx, tx)
	transactional = dbtx.GetTransactional(txCtx, dbConn)
	assert.Equal(t, tx, transactional)
}
