-- Remove the check constraint that requires discord_channel_id for Discord jobs
-- This allows existing Discord jobs to continue working without the channel_id field

-- Step 1: Drop constraint from production schema
ALTER TABLE claudecontrol.jobs 
DROP CONSTRAINT IF EXISTS check_job_type_payload_fields;

-- Step 2: Drop constraint from test schema  
ALTER TABLE claudecontrol_test.jobs 
DROP CONSTRAINT IF EXISTS check_job_type_payload_fields_test;