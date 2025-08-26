-- Create processed Discord messages table

-- Step 1: Create processed_discord_messages table in production schema
CREATE TABLE claudecontrol.processed_discord_messages (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    discord_message_id TEXT NOT NULL,
    discord_thread_id TEXT NOT NULL,
    text_content TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('QUEUED', 'IN_PROGRESS', 'COMPLETED')),
    discord_integration_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Foreign key constraints
    CONSTRAINT fk_processed_discord_messages_job_id 
        FOREIGN KEY (job_id) 
        REFERENCES claudecontrol.jobs(id) 
        ON DELETE CASCADE,
    
    CONSTRAINT fk_processed_discord_messages_discord_integration_id 
        FOREIGN KEY (discord_integration_id) 
        REFERENCES claudecontrol.discord_integrations(id) 
        ON DELETE CASCADE,
    
    CONSTRAINT fk_processed_discord_messages_organization_id 
        FOREIGN KEY (organization_id) 
        REFERENCES claudecontrol.organizations(id) 
        ON DELETE CASCADE
);

-- Step 2: Create processed_discord_messages table in test schema
CREATE TABLE claudecontrol_test.processed_discord_messages (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    discord_message_id TEXT NOT NULL,
    discord_thread_id TEXT NOT NULL,
    text_content TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('QUEUED', 'IN_PROGRESS', 'COMPLETED')),
    discord_integration_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Foreign key constraints
    CONSTRAINT fk_processed_discord_messages_job_id_test 
        FOREIGN KEY (job_id) 
        REFERENCES claudecontrol_test.jobs(id) 
        ON DELETE CASCADE,
    
    CONSTRAINT fk_processed_discord_messages_discord_integration_id_test 
        FOREIGN KEY (discord_integration_id) 
        REFERENCES claudecontrol_test.discord_integrations(id) 
        ON DELETE CASCADE,
    
    CONSTRAINT fk_processed_discord_messages_organization_id_test 
        FOREIGN KEY (organization_id) 
        REFERENCES claudecontrol_test.organizations(id) 
        ON DELETE CASCADE
);

-- Step 3: Create indexes for production schema
CREATE INDEX idx_processed_discord_messages_job_id ON claudecontrol.processed_discord_messages(job_id);
CREATE INDEX idx_processed_discord_messages_status ON claudecontrol.processed_discord_messages(status);
CREATE INDEX idx_processed_discord_messages_discord_integration_id ON claudecontrol.processed_discord_messages(discord_integration_id);
CREATE INDEX idx_processed_discord_messages_organization_id ON claudecontrol.processed_discord_messages(organization_id);
CREATE INDEX idx_processed_discord_messages_updated_at ON claudecontrol.processed_discord_messages(updated_at);

-- Create composite index for common queries
CREATE INDEX idx_processed_discord_messages_job_status ON claudecontrol.processed_discord_messages(job_id, status);
CREATE INDEX idx_processed_discord_messages_integration_org ON claudecontrol.processed_discord_messages(discord_integration_id, organization_id);

-- Step 4: Create indexes for test schema
CREATE INDEX idx_processed_discord_messages_job_id_test ON claudecontrol_test.processed_discord_messages(job_id);
CREATE INDEX idx_processed_discord_messages_status_test ON claudecontrol_test.processed_discord_messages(status);
CREATE INDEX idx_processed_discord_messages_discord_integration_id_test ON claudecontrol_test.processed_discord_messages(discord_integration_id);
CREATE INDEX idx_processed_discord_messages_organization_id_test ON claudecontrol_test.processed_discord_messages(organization_id);
CREATE INDEX idx_processed_discord_messages_updated_at_test ON claudecontrol_test.processed_discord_messages(updated_at);

-- Create composite index for common queries in test schema
CREATE INDEX idx_processed_discord_messages_job_status_test ON claudecontrol_test.processed_discord_messages(job_id, status);
CREATE INDEX idx_processed_discord_messages_integration_org_test ON claudecontrol_test.processed_discord_messages(discord_integration_id, organization_id);

