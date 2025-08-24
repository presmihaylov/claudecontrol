-- Create anthropic_integrations table for storing Anthropic API keys and Claude OAuth tokens

-- Production schema
CREATE TABLE IF NOT EXISTS claudecontrol.anthropic_integrations (
    id TEXT PRIMARY KEY,
    anthropic_api_key TEXT,
    claude_code_oauth_token TEXT,
    organization_id TEXT NOT NULL REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    -- Ensure exactly one token type is set
    CONSTRAINT exactly_one_token CHECK (
        (anthropic_api_key IS NOT NULL AND claude_code_oauth_token IS NULL) OR
        (anthropic_api_key IS NULL AND claude_code_oauth_token IS NOT NULL)
    )
);

-- Create index on organization_id for faster queries
CREATE INDEX idx_anthropic_integrations_organization_id ON claudecontrol.anthropic_integrations(organization_id);

-- Test schema
CREATE TABLE IF NOT EXISTS claudecontrol_test.anthropic_integrations (
    id TEXT PRIMARY KEY,
    anthropic_api_key TEXT,
    claude_code_oauth_token TEXT,
    organization_id TEXT NOT NULL REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    -- Ensure exactly one token type is set
    CONSTRAINT exactly_one_token CHECK (
        (anthropic_api_key IS NOT NULL AND claude_code_oauth_token IS NULL) OR
        (anthropic_api_key IS NULL AND claude_code_oauth_token IS NOT NULL)
    )
);

-- Create index on organization_id for faster queries (test schema)
CREATE INDEX idx_anthropic_integrations_organization_id_test ON claudecontrol_test.anthropic_integrations(organization_id);