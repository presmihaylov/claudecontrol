-- Add agent_id column to active_agents table as UUID type
ALTER TABLE claudecontrol.active_agents 
ADD COLUMN agent_id UUID;

-- Create unique constraint on (slack_integration_id, agent_id)
-- This allows only one active agent per agent_id per slack integration
ALTER TABLE claudecontrol.active_agents 
ADD CONSTRAINT unique_slack_integration_agent_id 
UNIQUE (slack_integration_id, agent_id);

-- Add the same changes to test schema
ALTER TABLE claudecontrol_test.active_agents 
ADD COLUMN agent_id UUID;

ALTER TABLE claudecontrol_test.active_agents 
ADD CONSTRAINT unique_slack_integration_agent_id 
UNIQUE (slack_integration_id, agent_id);