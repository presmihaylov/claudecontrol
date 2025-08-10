-- Add Discord channel_id and guild_id fields to jobs table

-- Step 1: Add new Discord columns to jobs table in production schema
ALTER TABLE claudecontrol.jobs 
ADD COLUMN discord_channel_id TEXT,
ADD COLUMN discord_guild_id TEXT;

-- Step 2: Add new Discord columns to jobs table in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN discord_channel_id TEXT,
ADD COLUMN discord_guild_id TEXT;

-- Step 3: Update constraint to include new Discord fields in production schema
ALTER TABLE claudecontrol.jobs 
DROP CONSTRAINT check_job_type_payload_fields;

ALTER TABLE claudecontrol.jobs 
ADD CONSTRAINT check_job_type_payload_fields 
CHECK (
    (job_type = 'slack' AND 
     slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND 
     slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND 
     discord_channel_id IS NULL AND discord_guild_id IS NULL AND
     discord_user_id IS NULL AND discord_integration_id IS NULL) OR
    (job_type = 'discord' AND 
     discord_message_id IS NOT NULL AND discord_thread_id IS NOT NULL AND 
     discord_channel_id IS NOT NULL AND discord_guild_id IS NOT NULL AND
     discord_user_id IS NOT NULL AND discord_integration_id IS NOT NULL AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL) OR
    (job_type NOT IN ('slack', 'discord') AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND 
     discord_channel_id IS NULL AND discord_guild_id IS NULL AND
     discord_user_id IS NULL AND discord_integration_id IS NULL)
);

-- Step 4: Update constraint to include new Discord fields in test schema
ALTER TABLE claudecontrol_test.jobs 
DROP CONSTRAINT check_job_type_payload_fields_test;

ALTER TABLE claudecontrol_test.jobs 
ADD CONSTRAINT check_job_type_payload_fields_test 
CHECK (
    (job_type = 'slack' AND 
     slack_thread_ts IS NOT NULL AND slack_channel_id IS NOT NULL AND 
     slack_user_id IS NOT NULL AND slack_integration_id IS NOT NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND 
     discord_channel_id IS NULL AND discord_guild_id IS NULL AND
     discord_user_id IS NULL AND discord_integration_id IS NULL) OR
    (job_type = 'discord' AND 
     discord_message_id IS NOT NULL AND discord_thread_id IS NOT NULL AND 
     discord_channel_id IS NOT NULL AND discord_guild_id IS NOT NULL AND
     discord_user_id IS NOT NULL AND discord_integration_id IS NOT NULL AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL) OR
    (job_type NOT IN ('slack', 'discord') AND
     slack_thread_ts IS NULL AND slack_channel_id IS NULL AND 
     slack_user_id IS NULL AND slack_integration_id IS NULL AND
     discord_message_id IS NULL AND discord_thread_id IS NULL AND 
     discord_channel_id IS NULL AND discord_guild_id IS NULL AND
     discord_user_id IS NULL AND discord_integration_id IS NULL)
);