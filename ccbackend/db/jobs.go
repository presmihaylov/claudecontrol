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

// Column names for jobs table
var jobsColumns = []string{
	"id",
	"slack_thread_ts",
	"slack_channel_id",
	"slack_user_id",
	"slack_integration_id",
	"created_at",
	"updated_at",
}

func NewPostgresJobsRepository(db *sqlx.DB, schema string) *PostgresJobsRepository {
	return &PostgresJobsRepository{db: db, schema: schema}
}

func (r *PostgresJobsRepository) CreateJob(ctx context.Context, job *models.Job) error {
	db := dbtx.GetTransactional(ctx, r.db)
	insertColumns := []string{
		"id",
		"slack_thread_ts",
		"slack_channel_id",
		"slack_user_id",
		"slack_integration_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(jobsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.jobs (%s) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := db.QueryRowxContext(ctx, query, job.ID, job.SlackThreadTS, job.SlackChannelID, job.SlackUserID, job.OrganizationID).
		StructScan(job)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

func (r *PostgresJobsRepository) GetJobByID(
	ctx context.Context,
	id string,
	slackIntegrationID string,
) (mo.Option[*models.Job], error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(jobsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.jobs 
		WHERE id = $1 AND slack_integration_id = $2`, columnsStr, r.schema)

	job := &models.Job{}
	err := db.GetContext(ctx, job, query, id, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.Job](), nil
		}
		return mo.None[*models.Job](), fmt.Errorf("failed to get job: %w", err)
	}

	return mo.Some(job), nil
}

func (r *PostgresJobsRepository) GetJobBySlackThread(
	ctx context.Context,
	threadTS, channelID, slackIntegrationID string,
) (mo.Option[*models.Job], error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(jobsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.jobs 
		WHERE slack_thread_ts = $1 AND slack_channel_id = $2 AND slack_integration_id = $3`, columnsStr, r.schema)

	job := &models.Job{}
	err := db.GetContext(ctx, job, query, threadTS, channelID, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.Job](), nil
		}
		return mo.None[*models.Job](), fmt.Errorf("failed to get job by slack thread: %w", err)
	}

	return mo.Some(job), nil
}

func (r *PostgresJobsRepository) UpdateJob(ctx context.Context, job *models.Job) error {
	db := dbtx.GetTransactional(ctx, r.db)
	returningStr := strings.Join(jobsColumns, ", ")
	query := fmt.Sprintf(`
		UPDATE %s.jobs 
		SET slack_thread_ts = $2, slack_channel_id = $3, slack_user_id = $4, updated_at = NOW() 
		WHERE id = $1 AND slack_integration_id = $5
		RETURNING %s`, r.schema, returningStr)

	err := db.QueryRowxContext(ctx, query, job.ID, job.SlackThreadTS, job.SlackChannelID, job.SlackUserID, job.OrganizationID).
		StructScan(job)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

func (r *PostgresJobsRepository) UpdateJobTimestamp(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
) error {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		UPDATE %s.jobs 
		SET updated_at = NOW() 
		WHERE id = $1 AND slack_integration_id = $2`, r.schema)

	_, err := db.ExecContext(ctx, query, jobID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	return nil
}

func (r *PostgresJobsRepository) GetIdleJobs(ctx context.Context, idleMinutes int) ([]*models.Job, error) {
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
		WHERE NOT EXISTS (
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

	var jobs []*models.Job
	err := db.SelectContext(ctx, &jobs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle jobs: %w", err)
	}

	return jobs, nil
}

func (r *PostgresJobsRepository) DeleteJob(ctx context.Context, id string, slackIntegrationID string) (bool, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		DELETE FROM %s.jobs 
		WHERE id = $1 AND slack_integration_id = $2`, r.schema)

	result, err := db.ExecContext(ctx, query, id, slackIntegrationID)
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
) (bool, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		UPDATE %s.jobs 
		SET updated_at = $2 
		WHERE id = $1 AND slack_integration_id = $3`, r.schema)

	result, err := db.ExecContext(ctx, query, id, updatedAt, slackIntegrationID)
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
		AND psm.status = 'QUEUED'
		ORDER BY j.created_at ASC`, columnsStr, r.schema, r.schema)

	var jobs []*models.Job
	err := db.SelectContext(ctx, &jobs, query, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs with queued messages: %w", err)
	}

	return jobs, nil
}
