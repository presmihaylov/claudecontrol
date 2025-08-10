-- Add Discord job support to the jobs table

-- Step 1: Add Discord-specific columns to jobs table in production schema
ALTER TABLE claudecontrol.jobs 
ADD COLUMN discord_message_id VARCHAR(255),
ADD COLUMN discord_thread_id VARCHAR(255),
ADD COLUMN discord_integration_id VARCHAR(255);

-- Step 2: Add Discord-specific columns to jobs table in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN discord_message_id VARCHAR(255),
ADD COLUMN discord_thread_id VARCHAR(255),
ADD COLUMN discord_integration_id VARCHAR(255);

-- Step 3: Drop existing constraint for Slack-only fields in production schema
ALTER TABLE claudecontrol.jobs 
DROP CONSTRAINT check_job_type_slack_fields;

-- Step 4: Drop existing constraint for Slack-only fields in test schema
ALTER TABLE claudecontrol_test.jobs 
DROP CONSTRAINT check_job_type_slack_fields_test;

-- Step 5: Add updated polymorphic constraint for both Slack and Discord in production schema
ALTER TABLE claudecontrol.jobs 
ADD CONSTRAINT check_job_type_fields 
CHECK (
    (job_type = 'slack' AND 
     slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND 
     slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND discord_integration_id IS NULL) OR
    (job_type = 'discord' AND 
     discord_message_id IS NOT NULL AND discord_thread_id IS NOT NULL AND 
     discord_integration_id IS NOT NULL AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL) OR
    (job_type NOT IN ('slack', 'discord'))
);

-- Step 6: Add updated polymorphic constraint for both Slack and Discord in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD CONSTRAINT check_job_type_fields_test 
CHECK (
    (job_type = 'slack' AND 
     slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND 
     slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND discord_integration_id IS NULL) OR
    (job_type = 'discord' AND 
     discord_message_id IS NOT NULL AND discord_thread_id IS NOT NULL AND 
     discord_integration_id IS NOT NULL AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL) OR
    (job_type NOT IN ('slack', 'discord'))
);

-- Step 7: Add Discord-specific composite index in production schema
CREATE INDEX idx_jobs_discord_lookup ON claudecontrol.jobs(discord_thread_id, discord_integration_id) 
WHERE job_type = 'discord';

-- Step 8: Add Discord-specific composite index in test schema
CREATE INDEX idx_jobs_discord_lookup_test ON claudecontrol_test.jobs(discord_thread_id, discord_integration_id) 
WHERE job_type = 'discord';

-- Step 9: Add foreign key constraint for Discord integration in production schema
ALTER TABLE claudecontrol.jobs 
ADD CONSTRAINT fk_jobs_discord_integration_id 
FOREIGN KEY (discord_integration_id) 
REFERENCES claudecontrol.discord_integrations(id) 
ON DELETE CASCADE;

-- Step 10: Add foreign key constraint for Discord integration in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD CONSTRAINT fk_jobs_discord_integration_id_test 
FOREIGN KEY (discord_integration_id) 
REFERENCES claudecontrol_test.discord_integrations(id) 
ON DELETE CASCADE;