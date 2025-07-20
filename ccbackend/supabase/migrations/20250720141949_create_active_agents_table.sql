-- Create active_agents table
CREATE TABLE claudecontrol.active_agents (
    id UUID PRIMARY KEY,
    assigned_job_id UUID NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);