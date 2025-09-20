-- Update all existing NULL repo_url values to fallback repository URL
UPDATE claudecontrol.active_agents
SET repo_url = 'github.com/unknown/repository'
WHERE repo_url IS NULL;

-- Add NOT NULL constraint to repo_url column
ALTER TABLE claudecontrol.active_agents
ALTER COLUMN repo_url SET NOT NULL;

-- Also update test schema
UPDATE claudecontrol_test.active_agents
SET repo_url = 'github.com/unknown/repository'
WHERE repo_url IS NULL;

-- Add NOT NULL constraint to test schema
ALTER TABLE claudecontrol_test.active_agents
ALTER COLUMN repo_url SET NOT NULL;