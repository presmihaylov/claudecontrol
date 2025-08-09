-- Add polymorphic job support with job_type column

-- Step 1: Add job_type column to jobs table in production schema (no default value)
ALTER TABLE claudecontrol.jobs 
ADD COLUMN job_type VARCHAR(50) NOT NULL;

-- Step 2: Update all existing records to have job_type='slack' in production schema
UPDATE claudecontrol.jobs 
SET job_type = 'slack' 
WHERE job_type IS NULL OR job_type = '';

-- Step 3: Add job_type column to jobs table in test schema (no default value)
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN job_type VARCHAR(50) NOT NULL;

-- Step 4: Update all existing records to have job_type='slack' in test schema
UPDATE claudecontrol_test.jobs 
SET job_type = 'slack' 
WHERE job_type IS NULL OR job_type = '';

-- Step 5: Add constraint to ensure Slack fields are only populated when job_type='slack' in production schema
ALTER TABLE claudecontrol.jobs 
ADD CONSTRAINT check_job_type_slack_fields 
CHECK (
    (job_type = 'slack' AND slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL) OR
    (job_type != 'slack' AND slack_thread_ts IS NULL AND slack_channel_id IS NULL AND slack_user_id IS NULL AND slack_integration_id IS NULL)
);

-- Step 6: Add constraint to ensure Slack fields are only populated when job_type='slack' in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD CONSTRAINT check_job_type_slack_fields_test 
CHECK (
    (job_type = 'slack' AND slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL) OR
    (job_type != 'slack' AND slack_thread_ts IS NULL AND slack_channel_id IS NULL AND slack_user_id IS NULL AND slack_integration_id IS NULL)
);

-- Step 7: Create index on job_type for performance in production schema
CREATE INDEX idx_jobs_job_type ON claudecontrol.jobs(job_type);

-- Step 8: Create index on job_type for performance in test schema
CREATE INDEX idx_jobs_job_type_test ON claudecontrol_test.jobs(job_type);

-- Step 9: Create composite index for Slack-specific queries in production schema
CREATE INDEX idx_jobs_slack_lookup ON claudecontrol.jobs(slack_thread_ts, slack_channel_id, slack_integration_id) 
WHERE job_type = 'slack';

-- Step 10: Create composite index for Slack-specific queries in test schema
CREATE INDEX idx_jobs_slack_lookup_test ON claudecontrol_test.jobs(slack_thread_ts, slack_channel_id, slack_integration_id) 
WHERE job_type = 'slack';