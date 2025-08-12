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
	ID        string       `db:"id"`
	JobType   string       `db:"job_type"`
	OrgID     models.OrgID `db:"organization_id"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`

	// Slack fields (nullable)
	SlackThreadTS      *string `db:"slack_thread_ts"`
	SlackChannelID     *string `db:"slack_channel_id"`
	SlackUserID        *string `db:"slack_user_id"`
	SlackIntegrationID *string `db:"slack_integration_id"`

	// Discord fields (nullable)
	DiscordMessageID     *string `db:"discord_message_id"`
	DiscordChannelID     *string `db:"discord_channel_id"`
	DiscordThreadID      *string `db:"discord_thread_id"`
	DiscordUserID        *string `db:"discord_user_id"`
	DiscordIntegrationID *string `db:"discord_integration_id"`
}

// Column names for jobs table
var jobsColumns = []string{
	"id",
	"job_type",
	"slack_thread_ts",
	"slack_channel_id",
	"slack_user_id",
	"slack_integration_id",
	"discord_message_id",
	"discord_channel_id",
	"discord_thread_id",
	"discord_user_id",
	"discord_integration_id",
	"organization_id",
	"created_at",
	"updated_at",
}

func NewPostgresJobsRepository(db *sqlx.DB, schema string) *PostgresJobsRepository {
	return &PostgresJobsRepository{db: db, schema: schema}
}

// dbJobToModel converts a DBJob to models.Job
func dbJobToModel(dbJob *DBJob) (*models.Job, error) {
	job := &models.Job{
		ID:        dbJob.ID,
		JobType:   models.JobType(dbJob.JobType),
		OrgID:     dbJob.OrgID,
		CreatedAt: dbJob.CreatedAt,
		UpdatedAt: dbJob.UpdatedAt,
	}

	// Populate payload based on type with comprehensive validation
	switch job.JobType {
	case models.JobTypeSlack:
		if dbJob.SlackThreadTS == nil ||
			dbJob.SlackChannelID == nil ||
			dbJob.SlackUserID == nil ||
			dbJob.SlackIntegrationID == nil {
			return nil, fmt.Errorf("slack job missing required fields: job_id=%s", dbJob.ID)
		}
		job.SlackPayload = &models.SlackJobPayload{
			ThreadTS:      *dbJob.SlackThreadTS,
			ChannelID:     *dbJob.SlackChannelID,
			UserID:        *dbJob.SlackUserID,
			IntegrationID: *dbJob.SlackIntegrationID,
		}
	case models.JobTypeDiscord:
		if dbJob.DiscordMessageID == nil ||
			dbJob.DiscordChannelID == nil ||
			dbJob.DiscordThreadID == nil ||
			dbJob.DiscordUserID == nil ||
			dbJob.DiscordIntegrationID == nil {
			return nil, fmt.Errorf("discord job missing required fields: job_id=%s", dbJob.ID)
		}
		job.DiscordPayload = &models.DiscordJobPayload{
			MessageID:     *dbJob.DiscordMessageID,
			ChannelID:     *dbJob.DiscordChannelID,
			ThreadID:      *dbJob.DiscordThreadID,
			UserID:        *dbJob.DiscordUserID,
			IntegrationID: *dbJob.DiscordIntegrationID,
		}
	default:
		return nil, fmt.Errorf("unsupported job type: %s for job_id=%s", job.JobType, dbJob.ID)
	}

	return job, nil
}

// modelToDBJob converts a models.Job to DBJob
func modelToDBJob(job *models.Job) (*DBJob, error) {
	// Validate that job type matches payload presence
	if job.JobType == models.JobTypeSlack && job.SlackPayload == nil {
		return nil, fmt.Errorf("slack job type requires SlackPayload to be populated")
	}
	if job.JobType == models.JobTypeDiscord && job.DiscordPayload == nil {
		return nil, fmt.Errorf("discord job type requires DiscordPayload to be populated")
	}

	dbJob := &DBJob{
		ID:        job.ID,
		JobType:   string(job.JobType),
		OrgID:     job.OrgID,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}

	// Set Slack fields if payload exists
	if job.SlackPayload != nil {
		dbJob.SlackThreadTS = &job.SlackPayload.ThreadTS
		dbJob.SlackChannelID = &job.SlackPayload.ChannelID
		dbJob.SlackUserID = &job.SlackPayload.UserID
		dbJob.SlackIntegrationID = &job.SlackPayload.IntegrationID
	}

	// Set Discord fields if payload exists
	if job.DiscordPayload != nil {
		dbJob.DiscordMessageID = &job.DiscordPayload.MessageID
		dbJob.DiscordChannelID = &job.DiscordPayload.ChannelID
		dbJob.DiscordThreadID = &job.DiscordPayload.ThreadID
		dbJob.DiscordUserID = &job.DiscordPayload.UserID
		dbJob.DiscordIntegrationID = &job.DiscordPayload.IntegrationID
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
		"discord_message_id",
		"discord_channel_id",
		"discord_thread_id",
		"discord_user_id",
		"discord_integration_id",
		"organization_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(jobsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.jobs (%s) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	var returnedDBJob DBJob
	err = db.QueryRowxContext(ctx, query,
		dbJob.ID, dbJob.JobType, dbJob.SlackThreadTS, dbJob.SlackChannelID,
		dbJob.SlackUserID, dbJob.SlackIntegrationID, dbJob.DiscordMessageID,
		dbJob.DiscordChannelID, dbJob.DiscordThreadID, dbJob.DiscordUserID,
		dbJob.DiscordIntegrationID, dbJob.OrgID).
		StructScan(&returnedDBJob)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	// Update the original job with returned values
	convertedJob, err := dbJobToModel(&returnedDBJob)
	if err != nil {
		return fmt.Errorf("failed to convert created job: %w", err)
	}
	*job = *convertedJob
	return nil
}

func (r *PostgresJobsRepository) GetJobByID(
	ctx context.Context,
	id string,
	organizationID models.OrgID,
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

	convertedJob, err := dbJobToModel(&dbJob)
	if err != nil {
		return mo.None[*models.Job](), fmt.Errorf("failed to convert job: %w", err)
	}
	return mo.Some(convertedJob), nil
}

func (r *PostgresJobsRepository) GetJobBySlackThread(
	ctx context.Context,
	threadTS, channelID, slackIntegrationID string,
	organizationID models.OrgID,
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

	convertedJob, err := dbJobToModel(&dbJob)
	if err != nil {
		return mo.None[*models.Job](), fmt.Errorf("failed to convert job: %w", err)
	}
	return mo.Some(convertedJob), nil
}

func (r *PostgresJobsRepository) GetJobByDiscordThread(
	ctx context.Context,
	threadID, discordIntegrationID string,
	organizationID models.OrgID,
) (mo.Option[*models.Job], error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(jobsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.jobs 
		WHERE discord_thread_id = $1 AND discord_integration_id = $2 AND organization_id = $3`, columnsStr, r.schema)

	var dbJob DBJob
	err := db.GetContext(ctx, &dbJob, query, threadID, discordIntegrationID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.Job](), nil
		}
		return mo.None[*models.Job](), fmt.Errorf("failed to get job by discord thread: %w", err)
	}

	convertedJob, err := dbJobToModel(&dbJob)
	if err != nil {
		return mo.None[*models.Job](), fmt.Errorf("failed to convert job: %w", err)
	}
	return mo.Some(convertedJob), nil
}

func (r *PostgresJobsRepository) UpdateJobTimestamp(
	ctx context.Context,
	jobID string,
	organizationID models.OrgID,
) error {
	db := dbtx.GetTransactional(ctx, r.db)
	query := fmt.Sprintf(`
		UPDATE %s.jobs 
		SET updated_at = NOW() 
		WHERE id = $1 AND organization_id = $2`, r.schema)

	_, err := db.ExecContext(ctx, query, jobID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	return nil
}

func (r *PostgresJobsRepository) GetJobs(
	ctx context.Context,
	organizationID models.OrgID,
) ([]*models.Job, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	columnsStr := strings.Join(jobsColumns, ", ")

	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.jobs 
		WHERE organization_id = $1
		ORDER BY created_at ASC`, columnsStr, r.schema)

	var dbJobs []DBJob
	err := db.SelectContext(ctx, &dbJobs, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}

	// Convert DBJobs to models.Job
	jobs := make([]*models.Job, 0, len(dbJobs))
	for _, dbJob := range dbJobs {
		convertedJob, err := dbJobToModel(&dbJob)
		if err != nil {
			return nil, fmt.Errorf("failed to convert job: %w", err)
		}
		jobs = append(jobs, convertedJob)
	}

	return jobs, nil
}

func (r *PostgresJobsRepository) DeleteJob(
	ctx context.Context,
	id string,
	organizationID models.OrgID,
) (bool, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	query := fmt.Sprintf(`
		DELETE FROM %s.jobs 
		WHERE id = $1 AND organization_id = $2`, r.schema)

	result, err := db.ExecContext(ctx, query, id, organizationID)
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
	organizationID models.OrgID,
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
	jobType models.JobType,
	integrationID string,
	organizationID models.OrgID,
) ([]*models.Job, error) {
	db := dbtx.GetTransactional(ctx, r.db)
	// Build column list with j. prefix for table alias
	var aliasedColumns []string
	for _, col := range jobsColumns {
		aliasedColumns = append(aliasedColumns, "j."+col)
	}
	columnsStr := strings.Join(aliasedColumns, ", ")

	var query string
	switch jobType {
	case models.JobTypeSlack:
		query = fmt.Sprintf(`
			SELECT DISTINCT %s 
			FROM %s.jobs j
			INNER JOIN %s.processed_slack_messages psm ON j.id = psm.job_id
			WHERE j.job_type = $1
			AND j.slack_integration_id = $2 
			AND j.organization_id = $3
			AND psm.status = 'QUEUED'
			ORDER BY j.created_at ASC`, columnsStr, r.schema, r.schema)
	case models.JobTypeDiscord:
		query = fmt.Sprintf(`
			SELECT DISTINCT %s 
			FROM %s.jobs j
			INNER JOIN %s.processed_discord_messages pdm ON j.id = pdm.job_id
			WHERE j.job_type = $1
			AND j.discord_integration_id = $2 
			AND j.organization_id = $3
			AND pdm.status = 'QUEUED'
			ORDER BY j.created_at ASC`, columnsStr, r.schema, r.schema)
	default:
		return nil, fmt.Errorf("unsupported job type: %s", jobType)
	}

	var dbJobs []DBJob
	err := db.SelectContext(ctx, &dbJobs, query, jobType, integrationID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs with queued messages: %w", err)
	}

	// Convert DBJobs to models.Job
	jobs := make([]*models.Job, 0, len(dbJobs))
	for _, dbJob := range dbJobs {
		convertedJob, err := dbJobToModel(&dbJob)
		if err != nil {
			return nil, fmt.Errorf("failed to convert job with queued messages: %w", err)
		}
		jobs = append(jobs, convertedJob)
	}

	return jobs, nil
}
