package db

import (
	"database/sql"
	"fmt"

	"ccbackend/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"
)

type PostgresJobsRepository struct {
	db     *sqlx.DB
	schema string
}

func NewPostgresJobsRepository(db *sqlx.DB, schema string) *PostgresJobsRepository {
	return &PostgresJobsRepository{db: db, schema: schema}
}

func (r *PostgresJobsRepository) CreateJob(job *models.Job) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.jobs (id, slack_thread_ts, slack_channel_id, created_at, updated_at) 
		VALUES ($1, $2, $3, NOW(), NOW()) 
		RETURNING id, slack_thread_ts, slack_channel_id, created_at, updated_at`, r.schema)

	err := r.db.QueryRowx(query, job.ID, job.SlackThreadTS, job.SlackChannelID).StructScan(job)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

func (r *PostgresJobsRepository) GetJobByID(id uuid.UUID) (*models.Job, error) {
	query := fmt.Sprintf(`
		SELECT id, slack_thread_ts, slack_channel_id, created_at, updated_at 
		FROM %s.jobs 
		WHERE id = $1`, r.schema)

	job := &models.Job{}
	err := r.db.Get(job, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

func (r *PostgresJobsRepository) GetJobBySlackThread(threadTS, channelID string) (*models.Job, error) {
	query := fmt.Sprintf(`
		SELECT id, slack_thread_ts, slack_channel_id, created_at, updated_at 
		FROM %s.jobs 
		WHERE slack_thread_ts = $1 AND slack_channel_id = $2`, r.schema)

	job := &models.Job{}
	err := r.db.Get(job, query, threadTS, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job with slack_thread_ts %s and slack_channel_id %s not found", threadTS, channelID)
		}
		return nil, fmt.Errorf("failed to get job by slack thread: %w", err)
	}

	return job, nil
}

func (r *PostgresJobsRepository) UpdateJob(job *models.Job) error {
	query := fmt.Sprintf(`
		UPDATE %s.jobs 
		SET slack_thread_ts = $2, slack_channel_id = $3, updated_at = NOW() 
		WHERE id = $1 
		RETURNING id, slack_thread_ts, slack_channel_id, created_at, updated_at`, r.schema)

	err := r.db.QueryRowx(query, job.ID, job.SlackThreadTS, job.SlackChannelID).StructScan(job)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

func (r *PostgresJobsRepository) UpdateJobTimestamp(jobID uuid.UUID) error {
	query := fmt.Sprintf(`
		UPDATE %s.jobs 
		SET updated_at = NOW() 
		WHERE id = $1`, r.schema)

	_, err := r.db.Exec(query, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	return nil
}

func (r *PostgresJobsRepository) GetIdleJobs(idleMinutes int) ([]*models.Job, error) {
	query := fmt.Sprintf(`
		SELECT id, slack_thread_ts, slack_channel_id, created_at, updated_at 
		FROM %s.jobs 
		WHERE updated_at < NOW() - INTERVAL '%d minutes'`, r.schema, idleMinutes)

	var jobs []*models.Job
	err := r.db.Select(&jobs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle jobs: %w", err)
	}

	return jobs, nil
}

func (r *PostgresJobsRepository) DeleteJob(id uuid.UUID) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.jobs 
		WHERE id = $1`, r.schema)

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job with id %s not found", id)
	}

	return nil
}

