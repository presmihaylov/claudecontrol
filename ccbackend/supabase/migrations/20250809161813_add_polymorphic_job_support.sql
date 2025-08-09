-- Add polymorphic job support with job_type column

-- Step 1: Add job_type column to jobs table in production schema
ALTER TABLE claudecontrol.jobs 
ADD COLUMN job_type VARCHAR(50) DEFAULT 'slack' NOT NULL;

-- Step 2: Add job_type column to jobs table in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN job_type VARCHAR(50) DEFAULT 'slack' NOT NULL;

-- Step 3: Add constraint to ensure Slack fields are only populated when job_type='slack' in production schema
ALTER TABLE claudecontrol.jobs 
ADD CONSTRAINT check_job_type_slack_fields 
CHECK (
    (job_type = 'slack' AND slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL) OR
    (job_type != 'slack' AND slack_thread_ts IS NULL AND slack_channel_id IS NULL AND slack_user_id IS NULL AND slack_integration_id IS NULL)
);

-- Step 4: Add constraint to ensure Slack fields are only populated when job_type='slack' in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD CONSTRAINT check_job_type_slack_fields_test 
CHECK (
    (job_type = 'slack' AND slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL) OR
    (job_type != 'slack' AND slack_thread_ts IS NULL AND slack_channel_id IS NULL AND slack_user_id IS NULL AND slack_integration_id IS NULL)
);

-- Step 5: Create index on job_type for performance in production schema
CREATE INDEX idx_jobs_job_type ON claudecontrol.jobs(job_type);

-- Step 6: Create index on job_type for performance in test schema
CREATE INDEX idx_jobs_job_type_test ON claudecontrol_test.jobs(job_type);

-- Step 7: Create composite index for Slack-specific queries in production schema
CREATE INDEX idx_jobs_slack_lookup ON claudecontrol.jobs(slack_thread_ts, slack_channel_id, slack_integration_id) 
WHERE job_type = 'slack';

-- Step 8: Create composite index for Slack-specific queries in test schema
CREATE INDEX idx_jobs_slack_lookup_test ON claudecontrol_test.jobs(slack_thread_ts, slack_channel_id, slack_integration_id) 
WHERE job_type = 'slack';