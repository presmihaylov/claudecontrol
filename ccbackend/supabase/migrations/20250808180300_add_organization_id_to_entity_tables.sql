-- Add organization_id column to active_agents table
ALTER TABLE claudecontrol.active_agents 
ADD COLUMN organization_id TEXT REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.active_agents 
ADD COLUMN organization_id TEXT REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Add organization_id column to jobs table
ALTER TABLE claudecontrol.jobs 
ADD COLUMN organization_id TEXT REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.jobs
ADD COLUMN organization_id TEXT REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Add organization_id column to processed_slack_messages table
ALTER TABLE claudecontrol.processed_slack_messages 
ADD COLUMN organization_id TEXT REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.processed_slack_messages
ADD COLUMN organization_id TEXT REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Add organization_id column to agent_job_assignments table
ALTER TABLE claudecontrol.agent_job_assignments 
ADD COLUMN organization_id TEXT REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.agent_job_assignments
ADD COLUMN organization_id TEXT REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Add indexes for performance
CREATE INDEX idx_active_agents_organization_id ON claudecontrol.active_agents(organization_id);
CREATE INDEX idx_active_agents_organization_id_test ON claudecontrol_test.active_agents(organization_id);

CREATE INDEX idx_jobs_organization_id ON claudecontrol.jobs(organization_id);
CREATE INDEX idx_jobs_organization_id_test ON claudecontrol_test.jobs(organization_id);

CREATE INDEX idx_processed_slack_messages_organization_id ON claudecontrol.processed_slack_messages(organization_id);
CREATE INDEX idx_processed_slack_messages_organization_id_test ON claudecontrol_test.processed_slack_messages(organization_id);

CREATE INDEX idx_agent_job_assignments_organization_id ON claudecontrol.agent_job_assignments(organization_id);
CREATE INDEX idx_agent_job_assignments_organization_id_test ON claudecontrol_test.agent_job_assignments(organization_id);