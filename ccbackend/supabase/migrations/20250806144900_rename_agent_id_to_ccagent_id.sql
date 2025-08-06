-- Rename agent_id column to ccagent_id in active_agents table
ALTER TABLE claudecontrol.active_agents RENAME COLUMN agent_id TO ccagent_id;
ALTER TABLE claudecontrol_test.active_agents RENAME COLUMN agent_id TO ccagent_id;

-- Rename agent_id column to ccagent_id in agent_job_assignments table  
ALTER TABLE claudecontrol.agent_job_assignments RENAME COLUMN agent_id TO ccagent_id;
ALTER TABLE claudecontrol_test.agent_job_assignments RENAME COLUMN agent_id TO ccagent_id;

-- Update unique constraint name for better clarity (optional)
ALTER TABLE claudecontrol.active_agents DROP CONSTRAINT IF EXISTS unique_slack_integration_agent_id;
ALTER TABLE claudecontrol.active_agents ADD CONSTRAINT unique_slack_integration_ccagent_id 
UNIQUE (slack_integration_id, ccagent_id);

ALTER TABLE claudecontrol_test.active_agents DROP CONSTRAINT IF EXISTS unique_slack_integration_agent_id;
ALTER TABLE claudecontrol_test.active_agents ADD CONSTRAINT unique_slack_integration_ccagent_id 
UNIQUE (slack_integration_id, ccagent_id);

-- Update foreign key constraint names for better clarity (optional)
ALTER TABLE claudecontrol.agent_job_assignments DROP CONSTRAINT IF EXISTS agent_job_assignments_agent_id_fkey;
ALTER TABLE claudecontrol.agent_job_assignments ADD CONSTRAINT agent_job_assignments_ccagent_id_fkey 
FOREIGN KEY (ccagent_id) REFERENCES claudecontrol.active_agents(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.agent_job_assignments DROP CONSTRAINT IF EXISTS agent_job_assignments_agent_id_fkey;
ALTER TABLE claudecontrol_test.agent_job_assignments ADD CONSTRAINT agent_job_assignments_ccagent_id_fkey 
FOREIGN KEY (ccagent_id) REFERENCES claudecontrol_test.active_agents(id) ON DELETE CASCADE;

-- Update conflict handling in queries from (agent_id, job_id) to (ccagent_id, job_id)
-- Note: This is handled in the application code, no DB changes needed for conflict clauses

-- Update index names for better clarity
DROP INDEX IF EXISTS claudecontrol.idx_agent_job_assignments_agent_id;
CREATE INDEX idx_agent_job_assignments_ccagent_id ON claudecontrol.agent_job_assignments(ccagent_id);

DROP INDEX IF EXISTS claudecontrol_test.idx_agent_job_assignments_agent_id_test;
CREATE INDEX idx_agent_job_assignments_ccagent_id_test ON claudecontrol_test.agent_job_assignments(ccagent_id);