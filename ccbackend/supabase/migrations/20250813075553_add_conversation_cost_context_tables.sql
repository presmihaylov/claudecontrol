-- Add conversation cost tracking tables for PMI-981

-- Table to track conversation costs per job
CREATE TABLE IF NOT EXISTS conversation_costs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    total_input_tokens INTEGER NOT NULL DEFAULT 0,
    total_output_tokens INTEGER NOT NULL DEFAULT 0,
    estimated_cost_usd DECIMAL(10,6) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table to manage conversation context and summarization
CREATE TABLE IF NOT EXISTS conversation_context (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    full_context TEXT NOT NULL DEFAULT '',
    summarized_context TEXT,
    context_size_tokens INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Add test schema equivalents for isolated testing
CREATE TABLE IF NOT EXISTS claudecontrol_test.conversation_costs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE,
    job_id TEXT NOT NULL REFERENCES claudecontrol_test.jobs(id) ON DELETE CASCADE,
    total_input_tokens INTEGER NOT NULL DEFAULT 0,
    total_output_tokens INTEGER NOT NULL DEFAULT 0,
    estimated_cost_usd DECIMAL(10,6) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS claudecontrol_test.conversation_context (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE,
    job_id TEXT NOT NULL REFERENCES claudecontrol_test.jobs(id) ON DELETE CASCADE,
    full_context TEXT NOT NULL DEFAULT '',
    summarized_context TEXT,
    context_size_tokens INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_conversation_costs_job_id ON conversation_costs(job_id);
CREATE INDEX IF NOT EXISTS idx_conversation_costs_organization_id ON conversation_costs(organization_id);
CREATE INDEX IF NOT EXISTS idx_conversation_costs_created_at ON conversation_costs(created_at);

CREATE INDEX IF NOT EXISTS idx_conversation_context_job_id ON conversation_context(job_id);
CREATE INDEX IF NOT EXISTS idx_conversation_context_organization_id ON conversation_context(organization_id);
CREATE INDEX IF NOT EXISTS idx_conversation_context_is_active ON conversation_context(is_active);

-- Test schema indexes
CREATE INDEX IF NOT EXISTS idx_conversation_costs_job_id_test ON claudecontrol_test.conversation_costs(job_id);
CREATE INDEX IF NOT EXISTS idx_conversation_costs_organization_id_test ON claudecontrol_test.conversation_costs(organization_id);
CREATE INDEX IF NOT EXISTS idx_conversation_costs_created_at_test ON claudecontrol_test.conversation_costs(created_at);

CREATE INDEX IF NOT EXISTS idx_conversation_context_job_id_test ON claudecontrol_test.conversation_context(job_id);
CREATE INDEX IF NOT EXISTS idx_conversation_context_organization_id_test ON claudecontrol_test.conversation_context(organization_id);
CREATE INDEX IF NOT EXISTS idx_conversation_context_is_active_test ON claudecontrol_test.conversation_context(is_active);