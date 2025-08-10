package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/samber/mo"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	dbtx "ccbackend/db/tx"
	"ccbackend/models"
)

type PostgresJobsRepository struct {
	db     *sqlx.DB
	schema string
}

// DBJob represents the database schema for jobs table
type DBJob struct {
	ID             string    `db:"id"`
	JobType        string    `db:"job_type"`
	OrganizationID string    `db:"organization_id"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`

	// Slack fields (nullable)
	SlackThreadTS      *string `db:"slack_thread_ts"`
	SlackChannelID     *string `db:"slack_channel_id"`
	SlackUserID        *string `db:"slack_user_id"`
	SlackIntegrationID *string `db:"slack_integration_id"`
}

// Column names for jobs table
var jobsColumns = []string{
	"id",
	"job_type",
	"slack_thread_ts",
	"slack_channel_id",
	"slack_user_id",
	"slack_integration_id",
	"organization_id",
	"created_at",
	"updated_at",
}

func NewPostgresJobsRepository(db *sqlx.DB, schema string) *PostgresJobsRepository {
	return &PostgresJobsRepository{db: db, schema: schema}
}

// dbJobToModel converts a DBJob to models.Job
func dbJobToModel(dbJob *DBJob) *models.Job {
	job := &models.Job{
		ID:             dbJob.ID,
		JobType:        models.JobType(dbJob.JobType),
		OrganizationID: dbJob.OrganizationID,
		CreatedAt:      dbJob.CreatedAt,
		UpdatedAt:      dbJob.UpdatedAt,
	}

	// Populate payload based on type with comprehensive nil checking
	if job.JobType == models.JobTypeSlack &&
		dbJob.SlackThreadTS != nil &&
		dbJob.SlackChannelID != nil &&
		dbJob.SlackUserID != nil &&
		dbJob.SlackIntegrationID != nil {
		job.SlackPayload = &models.SlackJobPayload{
			ThreadTS:      *dbJob.SlackThreadTS,
			ChannelID:     *dbJob.SlackChannelID,
			UserID:        *dbJob.SlackUserID,
			IntegrationID: *dbJob.SlackIntegrationID,
		}
	}

	return job
}

// modelToDBJob converts a models.Job to DBJob
func modelToDBJob(job *models.Job) (*DBJob, error) {
	// Validate that job type matches payload presence
	if job.JobType == models.JobTypeSlack && job.SlackPayload == nil {
		return nil, fmt.Errorf("slack job type requires SlackPayload to be populated")
	}

	dbJob := &DBJob{
		ID:             job.ID,
		JobType:        string(job.JobType),
		OrganizationID: job.OrganizationID,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
	}

	// Set Slack fields if payload exists
	if job.SlackPayload != nil {
		dbJob.SlackThreadTS = &job.SlackPayload.ThreadTS
		dbJob.SlackChannelID = &job.SlackPayload.ChannelID
		dbJob.SlackUserID = &job.SlackPayload.UserID
		dbJob.SlackIntegrationID = &job.SlackPayload.IntegrationID
	}

	return dbJob, nil
}

func (r *PostgresJobsRepository) CreateJob(ctx context.Context, job *models.Job) error {
	db := dbtx.GetTransactional(ctx, r.db)
	dbJob, err := modelToDBJob(job)
	if err != nil {
		return fmt.Errorf("failed to convert job to db model: %w", err)
	}

	insertColumns := []string{
		"id",
		"job_type",
		"slack_thread_ts",
		"slack_channel_id",
		"slack_user_id",
		"slack_integration_id",
		"organization_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(jobsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.jobs (%s) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	var returnedDBJob DBJob
	err = db.QueryRowxContext(ctx, query,
		dbJob.ID, dbJob.JobType, dbJob.SlackThreadTS, dbJob.SlackChannelID,
		dbJob.SlackUserID, dbJob.SlackIntegrationID, dbJob.OrganizationID).
		StructScan(&returnedDBJob)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	// Update the original job with returned values
	*job = *dbJobToModel(&returnedDBJob)
	return nil
}

func (r *PostgresJobsRepository) GetJobByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.Job], error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(jobsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.jobs 
		WHERE id = $1 AND organization_id = $2`, columnsStr, r.schema)

	var dbJob DBJob
	err := db.GetContext(ctx, &dbJob, query, id, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.Job](), nil
		}
		return mo.None[*models.Job](), fmt.Errorf("failed to get job: %w", err)
	}

	return mo.Some(dbJobToModel(&dbJob)), nil
}

func (r *PostgresJobsRepository) GetJobBySlackThread(
	ctx context.Context,
	threadTS, channelID, slackIntegrationID, organizationID string,
) (mo.Option[*models.Job], error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(jobsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.jobs 
		WHERE slack_thread_ts = $1 AND slack_channel_id = $2 AND slack_integration_id = $3 AND organization_id = $4`, columnsStr, r.schema)

	var dbJob DBJob
	err := db.GetContext(ctx, &dbJob, query, threadTS, channelID, slackIntegrationID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.Job](), nil
		}
		return mo.None[*models.Job](), fmt.Errorf("failed to get job by slack thread: %w", err)
	}

	return mo.Some(dbJobToModel(&dbJob)), nil
}

func (r *PostgresJobsRepository) UpdateJobTimestamp(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) error {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		UPDATE %s.jobs 
		SET updated_at = NOW() 
		WHERE id = $1 AND slack_integration_id = $2 AND organization_id = $3`, r.schema)

	_, err := db.ExecContext(ctx, query, jobID, slackIntegrationID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	return nil
}

func (r *PostgresJobsRepository) GetIdleJobs(
	ctx context.Context,
	idleMinutes int,
	organizationID string,
) ([]*models.Job, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	// Build column list with j. prefix for table alias
	var aliasedColumns []string
	for _, col := range jobsColumns {
		aliasedColumns = append(aliasedColumns, "j."+col)
	}
	columnsStr := strings.Join(aliasedColumns, ", ")

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.jobs j
		WHERE j.organization_id = $1
		AND NOT EXISTS (
			-- No messages that are not COMPLETED
			SELECT 1 FROM %s.processed_slack_messages psm 
			WHERE psm.job_id = j.id 
			AND psm.status != 'COMPLETED'
		)
		AND (
			-- Either no messages at all (use job updated_at)
			(NOT EXISTS (SELECT 1 FROM %s.processed_slack_messages psm WHERE psm.job_id = j.id)
			 AND j.updated_at < NOW() - INTERVAL '%d minutes')
			OR
			-- Or last COMPLETED message is older than threshold
			(SELECT MAX(psm.updated_at) FROM %s.processed_slack_messages psm 
			 WHERE psm.job_id = j.id AND psm.status = 'COMPLETED') < NOW() - INTERVAL '%d minutes'
		)`, columnsStr, r.schema, r.schema, r.schema, idleMinutes, r.schema, idleMinutes)

	var dbJobs []DBJob
	err := db.SelectContext(ctx, &dbJobs, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle jobs: %w", err)
	}

	// Convert DBJobs to models.Job
	jobs := make([]*models.Job, len(dbJobs))
	for i, dbJob := range dbJobs {
		jobs[i] = dbJobToModel(&dbJob)
	}

	return jobs, nil
}

func (r *PostgresJobsRepository) DeleteJob(
	ctx context.Context,
	id string,
	slackIntegrationID string,
	organizationID string,
) (bool, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		DELETE FROM %s.jobs 
		WHERE id = $1 AND slack_integration_id = $2 AND organization_id = $3`, r.schema)

	result, err := db.ExecContext(ctx, query, id, slackIntegrationID, organizationID)
	if err != nil {
		return false, fmt.Errorf("failed to delete job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// TESTS_UpdateJobUpdatedAt updates the updated_at timestamp of a job for testing purposes
func (r *PostgresJobsRepository) TESTS_UpdateJobUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
	organizationID string,
) (bool, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		UPDATE %s.jobs 
		SET updated_at = $2 
		WHERE id = $1 AND slack_integration_id = $3 AND organization_id = $4`, r.schema)

	result, err := db.ExecContext(ctx, query, id, updatedAt, slackIntegrationID, organizationID)
	if err != nil {
		return false, fmt.Errorf("failed to update job updated_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// GetJobsWithQueuedMessages returns jobs that have at least one message in QUEUED status
func (r *PostgresJobsRepository) GetJobsWithQueuedMessages(
	ctx context.Context,
	slackIntegrationID string,
	organizationID string,
) ([]*models.Job, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	// Build column list with j. prefix for table alias
	var aliasedColumns []string
	for _, col := range jobsColumns {
		aliasedColumns = append(aliasedColumns, "j."+col)
	}
	columnsStr := strings.Join(aliasedColumns, ", ")

	query := fmt.Sprintf(`
		SELECT DISTINCT %s 
		FROM %s.jobs j
		INNER JOIN %s.processed_slack_messages psm ON j.id = psm.job_id
		WHERE j.slack_integration_id = $1 
		AND j.organization_id = $2
		AND psm.status = 'QUEUED'
		ORDER BY j.created_at ASC`, columnsStr, r.schema, r.schema)

	var dbJobs []DBJob
	err := db.SelectContext(ctx, &dbJobs, query, slackIntegrationID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs with queued messages: %w", err)
	}

	// Convert DBJobs to models.Job
	jobs := make([]*models.Job, len(dbJobs))
	for i, dbJob := range dbJobs {
		jobs[i] = dbJobToModel(&dbJob)
	}

	return jobs, nil
}

// GetJobsByOrganizationID retrieves all jobs for a specific organization
func (r *PostgresJobsRepository) GetJobsByOrganizationID(
	ctx context.Context,
	organizationID string,
) ([]*models.Job, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.jobs
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`, strings.Join(jobsColumns, ", "), r.schema)

	var dbJobs []DBJob
	err := db.SelectContext(ctx, &dbJobs, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs by organization: %w", err)
	}

	// Convert DBJobs to models.Job
	jobs := make([]*models.Job, len(dbJobs))
	for i, dbJob := range dbJobs {
		jobs[i] = dbJobToModel(&dbJob)
	}

	return jobs, nil
}
