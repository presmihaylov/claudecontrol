-- Create test schema with same structure as prod
CREATE SCHEMA IF NOT EXISTS claudecontrol_test;

-- Create active_agents table in test schema
CREATE TABLE claudecontrol_test.active_agents (
    id UUID PRIMARY KEY,
    assigned_job_id UUID NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);