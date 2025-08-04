-- Add slack_user_id column to jobs table for production
ALTER TABLE claudecontrol.jobs 
ADD COLUMN slack_user_id TEXT NOT NULL;

-- Add slack_user_id column to jobs table for test schema  
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN slack_user_id TEXT NOT NULL;