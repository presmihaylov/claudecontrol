-- Add repo_url column to active_agents table for repository tracking
ALTER TABLE claudecontrol.active_agents
ADD COLUMN repo_url TEXT NULL;

-- Also add to test schema
ALTER TABLE claudecontrol_test.active_agents
ADD COLUMN repo_url TEXT NULL;