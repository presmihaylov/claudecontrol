-- Add Discord support to jobs table

-- Step 1: Add Discord columns to jobs table in production schema
ALTER TABLE claudecontrol.jobs 
ADD COLUMN discord_message_id VARCHAR(255),
ADD COLUMN discord_thread_id VARCHAR(255),
ADD COLUMN discord_user_id VARCHAR(255),
ADD COLUMN discord_integration_id CHAR(26);

-- Step 2: Add Discord columns to jobs table in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN discord_message_id VARCHAR(255),
ADD COLUMN discord_thread_id VARCHAR(255),
ADD COLUMN discord_user_id VARCHAR(255),
ADD COLUMN discord_integration_id CHAR(26);

-- Step 3: Update constraint to handle Discord fields in production schema
ALTER TABLE claudecontrol.jobs 
DROP CONSTRAINT check_job_type_slack_fields;

ALTER TABLE claudecontrol.jobs 
ADD CONSTRAINT check_job_type_payload_fields 
CHECK (
    (job_type = 'slack' AND 
     slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND 
     slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND 
     discord_user_id IS NULL AND discord_integration_id IS NULL) OR
    (job_type = 'discord' AND 
     discord_message_id IS NOT NULL AND discord_thread_id IS NOT NULL AND 
     discord_user_id IS NOT NULL AND discord_integration_id IS NOT NULL AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL) OR
    (job_type NOT IN ('slack', 'discord') AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND 
     discord_user_id IS NULL AND discord_integration_id IS NULL)
);

-- Step 4: Update constraint to handle Discord fields in test schema
ALTER TABLE claudecontrol_test.jobs 
DROP CONSTRAINT check_job_type_slack_fields_test;

ALTER TABLE claudecontrol_test.jobs 
ADD CONSTRAINT check_job_type_payload_fields_test 
CHECK (
    (job_type = 'slack' AND 
     slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND 
     slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND 
     discord_user_id IS NULL AND discord_integration_id IS NULL) OR
    (job_type = 'discord' AND 
     discord_message_id IS NOT NULL AND discord_thread_id IS NOT NULL AND 
     discord_user_id IS NOT NULL AND discord_integration_id IS NOT NULL AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL) OR
    (job_type NOT IN ('slack', 'discord') AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND 
     discord_user_id IS NULL AND discord_integration_id IS NULL)
);

-- Step 5: Create composite index for Discord-specific queries in production schema
CREATE INDEX idx_jobs_discord_lookup ON claudecontrol.jobs(discord_thread_id, discord_integration_id) 
WHERE job_type = 'discord';

-- Step 6: Create composite index for Discord-specific queries in test schema
CREATE INDEX idx_jobs_discord_lookup_test ON claudecontrol_test.jobs(discord_thread_id, discord_integration_id) 
WHERE job_type = 'discord';

-- Step 7: Add foreign key constraint for Discord integration in production schema
ALTER TABLE claudecontrol.jobs 
ADD CONSTRAINT fk_jobs_discord_integration_id 
FOREIGN KEY (discord_integration_id) 
REFERENCES claudecontrol.discord_integrations(id) 
ON DELETE CASCADE;

-- Step 8: Add foreign key constraint for Discord integration in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD CONSTRAINT fk_jobs_discord_integration_id_test 
FOREIGN KEY (discord_integration_id) 
REFERENCES claudecontrol_test.discord_integrations(id) 
ON DELETE CASCADE;