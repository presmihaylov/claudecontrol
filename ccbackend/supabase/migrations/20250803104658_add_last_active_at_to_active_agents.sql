-- Add last_active_at column to active_agents table for healthcheck tracking
ALTER TABLE claudecontrol.active_agents 
ADD COLUMN last_active_at TIMESTAMPTZ DEFAULT NOW() NOT NULL;

-- Also add to test schema
ALTER TABLE claudecontrol_test.active_agents 
ADD COLUMN last_active_at TIMESTAMPTZ DEFAULT NOW() NOT NULL;