package txmanager

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/testutils"
)

// TransactionalJobsRepo wraps the jobs repository to use transactions
type TransactionalJobsRepo struct {
	*db.PostgresJobsRepository
	db     *sqlx.DB // Keep reference to original db for GetQueryable
	schema string   // Database schema for queries
}

func NewTransactionalJobsRepo(repo *db.PostgresJobsRepository, db *sqlx.DB, schema string) *TransactionalJobsRepo {
	return &TransactionalJobsRepo{
		PostgresJobsRepository: repo,
		db:                     db,
		schema:                 schema,
	}
}

// CreateJobTx creates a job using transaction context if available
func (r *TransactionalJobsRepo) CreateJobTx(ctx context.Context, job *models.Job) error {
	queryable := GetQueryable(ctx, r.db)
	
	query := fmt.Sprintf(`
		INSERT INTO %s.jobs (id, slack_thread_ts, slack_channel_id, slack_user_id, slack_integration_id, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING id, slack_thread_ts, slack_channel_id, slack_user_id, slack_integration_id, created_at, updated_at`, r.schema)

	err := queryable.QueryRowxContext(ctx, query, job.ID, job.SlackThreadTS, job.SlackChannelID, job.SlackUserID, job.SlackIntegrationID).StructScan(job)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// GetJobByIDTx gets a job by ID using transaction context if available
func (r *TransactionalJobsRepo) GetJobByIDTx(ctx context.Context, id string, slackIntegrationID string) (*models.Job, error) {
	queryable := GetQueryable(ctx, r.db)
	
	query := fmt.Sprintf(`
		SELECT id, slack_thread_ts, slack_channel_id, slack_user_id, slack_integration_id, created_at, updated_at 
		FROM %s.jobs 
		WHERE id = $1 AND slack_integration_id = $2`, r.schema)

	job := &models.Job{}
	err := queryable.GetContext(ctx, job, query, id, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

// DeleteJobTx deletes a job using transaction context if available
func (r *TransactionalJobsRepo) DeleteJobTx(ctx context.Context, id string, slackIntegrationID string) error {
	queryable := GetQueryable(ctx, r.db)
	
	query := fmt.Sprintf(`DELETE FROM %s.jobs WHERE id = $1 AND slack_integration_id = $2`, r.schema)
	
	_, err := queryable.ExecContext(ctx, query, id, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}

func setupTransactionTest(t *testing.T) (services.TransactionManager, *TransactionalJobsRepo, *models.SlackIntegration, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	// Create transaction manager
	txManager := NewTransactionManager(dbConn)

	// Create repositories
	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)
	txJobsRepo := NewTransactionalJobsRepo(jobsRepo, dbConn, cfg.DatabaseSchema)
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

	return txManager, txJobsRepo, testIntegration, cleanup
}

func TestTransactionManager_WithTransaction_Success(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()
	
	var createdJob *models.Job
	
	// Execute transaction that should succeed
	err := txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create a job within the transaction
		job := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-ts",
			SlackChannelID:     "test-channel",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJobTx(txCtx, job); err != nil {
			return err
		}

		createdJob = job
		return nil
	})

	// Transaction should succeed
	require.NoError(t, err)
	require.NotNil(t, createdJob)

	// Job should exist in database after transaction commit
	retrievedJob, err := jobsRepo.GetJobByIDTx(ctx, createdJob.ID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, createdJob.ID, retrievedJob.ID)
	assert.Equal(t, createdJob.SlackThreadTS, retrievedJob.SlackThreadTS)
	
	// Clean up
	jobsRepo.DeleteJobTx(ctx, createdJob.ID, testIntegration.ID)
}

func TestTransactionManager_WithTransaction_Rollback_OnError(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()
	
	var jobID string
	
	// Execute transaction that should fail and rollback
	err := txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create a job within the transaction
		job := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-ts-rollback",
			SlackChannelID:     "test-channel",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJobTx(txCtx, job); err != nil {
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
	_, err = jobsRepo.GetJobByIDTx(ctx, jobID, testIntegration.ID)
	require.Error(t, err, "Job should not exist after rollback")
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

		txManager.WithTransaction(ctx, func(txCtx context.Context) error {
			// Create a job within the transaction
			job := &models.Job{
				ID:                 core.NewID("j"),
				SlackThreadTS:      "test-thread-ts-panic",
				SlackChannelID:     "test-channel",
				SlackUserID:        "test-user",
				SlackIntegrationID: testIntegration.ID,
			}

			if err := jobsRepo.CreateJobTx(txCtx, job); err != nil {
				return err
			}

			jobID = job.ID
			
			// Trigger panic
			panic("intentional panic to test rollback")
		})
	}()

	// Job should NOT exist in database after panic rollback
	_, err := jobsRepo.GetJobByIDTx(ctx, jobID, testIntegration.ID)
	require.Error(t, err, "Job should not exist after panic rollback")
}

func TestTransactionManager_WithTransaction_MultipleDatabaseOperations(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()
	
	var job1ID, job2ID string
	
	// Test multiple operations within single transaction
	err := txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create first job
		job1 := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-1",
			SlackChannelID:     "test-channel-1",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJobTx(txCtx, job1); err != nil {
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

		if err := jobsRepo.CreateJobTx(txCtx, job2); err != nil {
			return err
		}
		job2ID = job2.ID

		// Verify both jobs exist within transaction
		_, err := jobsRepo.GetJobByIDTx(txCtx, job1ID, testIntegration.ID)
		if err != nil {
			return fmt.Errorf("job1 should exist within transaction: %w", err)
		}

		_, err = jobsRepo.GetJobByIDTx(txCtx, job2ID, testIntegration.ID)  
		if err != nil {
			return fmt.Errorf("job2 should exist within transaction: %w", err)
		}

		return nil
	})

	// Transaction should succeed
	require.NoError(t, err)

	// Both jobs should exist after commit
	job1, err := jobsRepo.GetJobByIDTx(ctx, job1ID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-thread-1", job1.SlackThreadTS)

	job2, err := jobsRepo.GetJobByIDTx(ctx, job2ID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-thread-2", job2.SlackThreadTS)

	// Clean up
	jobsRepo.DeleteJobTx(ctx, job1ID, testIntegration.ID)
	jobsRepo.DeleteJobTx(ctx, job2ID, testIntegration.ID)
}

func TestTransactionManager_WithTransaction_MultipleDatabaseOperations_PartialRollback(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()
	
	var job1ID, job2ID string
	
	// Test rollback of multiple operations
	err := txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create first job
		job1 := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-thread-rollback-1",
			SlackChannelID:     "test-channel-1",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJobTx(txCtx, job1); err != nil {
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

		if err := jobsRepo.CreateJobTx(txCtx, job2); err != nil {
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
	_, err = jobsRepo.GetJobByIDTx(ctx, job1ID, testIntegration.ID)
	require.Error(t, err, "Job1 should not exist after rollback")

	_, err = jobsRepo.GetJobByIDTx(ctx, job2ID, testIntegration.ID)
	require.Error(t, err, "Job2 should not exist after rollback")
}

func TestTransactionManager_NestedTransactions(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()
	
	var jobID string
	
	// Test nested transaction support
	err := txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create job in outer transaction
		job := &models.Job{
			ID:                 core.NewID("j"),
			SlackThreadTS:      "test-nested-thread",
			SlackChannelID:     "test-channel",
			SlackUserID:        "test-user",
			SlackIntegrationID: testIntegration.ID,
		}

		if err := jobsRepo.CreateJobTx(txCtx, job); err != nil {
			return err
		}
		jobID = job.ID

		// Nested transaction (should reuse existing transaction)
		return txManager.WithTransaction(txCtx, func(nestedTxCtx context.Context) error {
			// Verify job exists within nested context
			_, err := jobsRepo.GetJobByIDTx(nestedTxCtx, jobID, testIntegration.ID)
			if err != nil {
				return fmt.Errorf("job should exist in nested transaction: %w", err)
			}

			// This should succeed and not interfere with outer transaction
			return nil
		})
	})

	// Both transactions should succeed
	require.NoError(t, err)

	// Job should exist after both transactions
	job, err := jobsRepo.GetJobByIDTx(ctx, jobID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-nested-thread", job.SlackThreadTS)

	// Clean up
	jobsRepo.DeleteJobTx(ctx, jobID, testIntegration.ID)
}

func TestTransactionManager_ManualTransactionControl(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()
	
	// Test manual transaction control
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

	err = jobsRepo.CreateJobTx(txCtx, job)
	require.NoError(t, err)

	// Job should not be visible outside transaction yet
	_, err = jobsRepo.GetJobByIDTx(ctx, job.ID, testIntegration.ID)
	require.Error(t, err, "Job should not be visible outside transaction before commit")

	// Commit manual transaction
	err = txManager.CommitTransaction(txCtx)
	require.NoError(t, err)

	// Job should now be visible
	retrievedJob, err := jobsRepo.GetJobByIDTx(ctx, job.ID, testIntegration.ID)
	require.NoError(t, err)
	assert.Equal(t, job.SlackThreadTS, retrievedJob.SlackThreadTS)

	// Clean up
	jobsRepo.DeleteJobTx(ctx, job.ID, testIntegration.ID)
}

func TestTransactionManager_ManualTransactionRollback(t *testing.T) {
	txManager, jobsRepo, testIntegration, cleanup := setupTransactionTest(t)
	defer cleanup()

	ctx := context.Background()
	
	// Test manual transaction rollback
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

	err = jobsRepo.CreateJobTx(txCtx, job)
	require.NoError(t, err)

	// Rollback manual transaction
	err = txManager.RollbackTransaction(txCtx)
	require.NoError(t, err)

	// Job should not exist after rollback
	_, err = jobsRepo.GetJobByIDTx(ctx, job.ID, testIntegration.ID)
	require.Error(t, err, "Job should not exist after manual rollback")
}

func TestGetQueryable_WithoutTransaction(t *testing.T) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	ctx := context.Background()
	
	// GetQueryable should return the db when no transaction in context
	queryable := GetQueryable(ctx, dbConn)
	assert.Equal(t, dbConn, queryable)
}

func TestGetQueryable_WithTransaction(t *testing.T) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	ctx := context.Background()
	
	// Begin transaction
	tx, err := dbConn.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	// Create context with transaction
	txCtx := WithTransaction(ctx, tx)
	
	// GetQueryable should return the transaction when transaction in context
	queryable := GetQueryable(txCtx, dbConn)
	assert.Equal(t, tx, queryable)
}

func TestTransactionFromContext(t *testing.T) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	ctx := context.Background()
	
	// No transaction in context
	tx, ok := TransactionFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, tx)

	// Transaction in context
	realTx, err := dbConn.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer realTx.Rollback()

	txCtx := WithTransaction(ctx, realTx)
	
	tx, ok = TransactionFromContext(txCtx)
	assert.True(t, ok)
	assert.Equal(t, realTx, tx)
}