-- Step 1: Add slack_user_id column as nullable
ALTER TABLE claudecontrol.jobs 
ADD COLUMN slack_user_id TEXT;

-- Step 2: Backfill existing rows with appropriate values
-- You can update with a fixed placeholder, join another table, or use logic as needed
UPDATE claudecontrol.jobs 
SET slack_user_id = 'unknown'
WHERE slack_user_id IS NULL;

-- Step 3: Make the column NOT NULL
ALTER TABLE claudecontrol.jobs 
ALTER COLUMN slack_user_id SET NOT NULL;

-- test schema changes
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN slack_user_id TEXT;

UPDATE claudecontrol_test.jobs 
SET slack_user_id = 'unknown'
WHERE slack_user_id IS NULL;

ALTER TABLE claudecontrol_test.jobs 
ALTER COLUMN slack_user_id SET NOT NULL;

