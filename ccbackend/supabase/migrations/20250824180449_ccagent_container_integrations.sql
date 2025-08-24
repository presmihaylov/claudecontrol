-- Create ccagent_container_integrations table for managing CCAgent container configurations

-- Production schema
CREATE TABLE IF NOT EXISTS claudecontrol.ccagent_container_integrations (
    id TEXT PRIMARY KEY,
    instances_count INTEGER NOT NULL DEFAULT 1 CHECK (instances_count >= 1 AND instances_count <= 10),
    repo_url TEXT NOT NULL,
    organization_id TEXT NOT NULL REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index for organization lookups
CREATE INDEX IF NOT EXISTS idx_ccagent_container_integrations_organization_id 
    ON claudecontrol.ccagent_container_integrations(organization_id);

-- Create unique constraint to ensure only one integration per organization
CREATE UNIQUE INDEX IF NOT EXISTS idx_ccagent_container_integrations_org_unique 
    ON claudecontrol.ccagent_container_integrations(organization_id);

-- Test schema
CREATE TABLE IF NOT EXISTS claudecontrol_test.ccagent_container_integrations (
    id TEXT PRIMARY KEY,
    instances_count INTEGER NOT NULL DEFAULT 1 CHECK (instances_count >= 1 AND instances_count <= 10),
    repo_url TEXT NOT NULL,
    organization_id TEXT NOT NULL REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index for test schema
CREATE INDEX IF NOT EXISTS idx_ccagent_container_integrations_organization_id_test 
    ON claudecontrol_test.ccagent_container_integrations(organization_id);

-- Create unique constraint for test schema
CREATE UNIQUE INDEX IF NOT EXISTS idx_ccagent_container_integrations_org_unique_test 
    ON claudecontrol_test.ccagent_container_integrations(organization_id);