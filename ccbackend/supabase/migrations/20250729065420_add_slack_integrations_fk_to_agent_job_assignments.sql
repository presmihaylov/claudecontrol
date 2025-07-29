-- Add foreign key constraint to slack_integrations table for agent_job_assignments

-- Add foreign key constraint for production schema
ALTER TABLE claudecontrol.agent_job_assignments 
ADD CONSTRAINT fk_agent_job_assignments_slack_integration 
FOREIGN KEY (slack_integration_id) 
REFERENCES claudecontrol.slack_integrations(id) 
ON DELETE CASCADE;

-- Add foreign key constraint for test schema
ALTER TABLE claudecontrol_test.agent_job_assignments 
ADD CONSTRAINT fk_agent_job_assignments_slack_integration 
FOREIGN KEY (slack_integration_id) 
REFERENCES claudecontrol_test.slack_integrations(id) 
ON DELETE CASCADE;