-- Add slack_integration_id column to active_agents table
ALTER TABLE claudecontrol.active_agents 
ADD COLUMN slack_integration_id UUID NOT NULL 
REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

-- Add index for performance on active_agents
CREATE INDEX idx_active_agents_slack_integration_id ON claudecontrol.active_agents(slack_integration_id);

-- Add slack_integration_id column to jobs table
ALTER TABLE claudecontrol.jobs 
ADD COLUMN slack_integration_id UUID NOT NULL 
REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

-- Add index for performance on jobs
CREATE INDEX idx_jobs_slack_integration_id ON claudecontrol.jobs(slack_integration_id);

-- Add slack_integration_id column to processed_slack_messages table
ALTER TABLE claudecontrol.processed_slack_messages 
ADD COLUMN slack_integration_id UUID NOT NULL 
REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

-- Add index for performance on processed_slack_messages
CREATE INDEX idx_processed_slack_messages_slack_integration_id ON claudecontrol.processed_slack_messages(slack_integration_id);

-- Add same changes to test schema
-- Clear test data if any exists
DELETE FROM claudecontrol_test.processed_slack_messages WHERE 1=1;
DELETE FROM claudecontrol_test.active_agents WHERE 1=1;  
DELETE FROM claudecontrol_test.jobs WHERE 1=1;

ALTER TABLE claudecontrol_test.active_agents 
ADD COLUMN slack_integration_id UUID NOT NULL 
REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;

CREATE INDEX idx_active_agents_slack_integration_id ON claudecontrol_test.active_agents(slack_integration_id);

ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN slack_integration_id UUID NOT NULL 
REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;

CREATE INDEX idx_jobs_slack_integration_id ON claudecontrol_test.jobs(slack_integration_id);

ALTER TABLE claudecontrol_test.processed_slack_messages 
ADD COLUMN slack_integration_id UUID NOT NULL 
REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;

CREATE INDEX idx_processed_slack_messages_slack_integration_id ON claudecontrol_test.processed_slack_messages(slack_integration_id);