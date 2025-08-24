-- Create github_integrations table for storing GitHub App installations

-- Production schema
CREATE TABLE IF NOT EXISTS claudecontrol.github_integrations (
    id TEXT PRIMARY KEY,
    github_installation_id TEXT NOT NULL UNIQUE,
    github_access_token TEXT NOT NULL,
    organization_id TEXT NOT NULL REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on organization_id for faster queries
CREATE INDEX idx_github_integrations_organization_id ON claudecontrol.github_integrations(organization_id);

-- Create index on github_installation_id for uniqueness and faster lookups
CREATE INDEX idx_github_integrations_installation_id ON claudecontrol.github_integrations(github_installation_id);

-- Test schema
CREATE TABLE IF NOT EXISTS claudecontrol_test.github_integrations (
    id TEXT PRIMARY KEY,
    github_installation_id TEXT NOT NULL UNIQUE,
    github_access_token TEXT NOT NULL,
    organization_id TEXT NOT NULL REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on organization_id for faster queries (test schema)
CREATE INDEX idx_github_integrations_organization_id_test ON claudecontrol_test.github_integrations(organization_id);

-- Create index on github_installation_id for uniqueness and faster lookups (test schema)
CREATE INDEX idx_github_integrations_installation_id_test ON claudecontrol_test.github_integrations(github_installation_id);