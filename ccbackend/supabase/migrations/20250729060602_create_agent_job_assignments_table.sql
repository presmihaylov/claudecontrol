-- Create agent_job_assignments junction table for production
CREATE TABLE claudecontrol.agent_job_assignments (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL REFERENCES claudecontrol.active_agents(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES claudecontrol.jobs(id) ON DELETE CASCADE,
    slack_integration_id UUID NOT NULL,
    assigned_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    UNIQUE(agent_id, job_id)
);

-- Create agent_job_assignments junction table for test schema
CREATE TABLE claudecontrol_test.agent_job_assignments (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL REFERENCES claudecontrol_test.active_agents(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES claudecontrol_test.jobs(id) ON DELETE CASCADE,
    slack_integration_id UUID NOT NULL,
    assigned_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    UNIQUE(agent_id, job_id)
);

-- Create indexes for performance
CREATE INDEX idx_agent_job_assignments_agent_id ON claudecontrol.agent_job_assignments(agent_id);
CREATE INDEX idx_agent_job_assignments_job_id ON claudecontrol.agent_job_assignments(job_id);

CREATE INDEX idx_agent_job_assignments_agent_id_test ON claudecontrol_test.agent_job_assignments(agent_id);
CREATE INDEX idx_agent_job_assignments_job_id_test ON claudecontrol_test.agent_job_assignments(job_id);