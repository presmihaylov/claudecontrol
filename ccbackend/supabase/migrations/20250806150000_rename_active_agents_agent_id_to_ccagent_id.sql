-- Rename agent_id column to ccagent_id in active_agents table only
-- This column stores the ccagent's identifier, not a foreign key reference

-- Rename the column in production schema
ALTER TABLE claudecontrol.active_agents RENAME COLUMN agent_id TO ccagent_id;

-- Rename the column in test schema  
ALTER TABLE claudecontrol_test.active_agents RENAME COLUMN agent_id TO ccagent_id;

-- Update the unique constraint to use the new column name
ALTER TABLE claudecontrol.active_agents DROP CONSTRAINT IF EXISTS unique_slack_integration_agent_id;
ALTER TABLE claudecontrol.active_agents ADD CONSTRAINT unique_slack_integration_ccagent_id 
UNIQUE (slack_integration_id, ccagent_id);

ALTER TABLE claudecontrol_test.active_agents DROP CONSTRAINT IF EXISTS unique_slack_integration_agent_id;
ALTER TABLE claudecontrol_test.active_agents ADD CONSTRAINT unique_slack_integration_ccagent_id 
UNIQUE (slack_integration_id, ccagent_id);

-- Note: agent_job_assignments.agent_id is NOT renamed as it's a foreign key to active_agents.id