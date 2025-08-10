-- Create processed_discord_messages table for tracking Discord message processing

-- Step 1: Create processed_discord_messages table in production schema
CREATE TABLE claudecontrol.processed_discord_messages (
    id VARCHAR(32) PRIMARY KEY,
    job_id VARCHAR(32) NOT NULL,
    discord_message_id VARCHAR(255) NOT NULL,
    discord_thread_id VARCHAR(255) NOT NULL,
    text_content TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    discord_integration_id VARCHAR(32) NOT NULL,
    organization_id VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Step 2: Create processed_discord_messages table in test schema
CREATE TABLE claudecontrol_test.processed_discord_messages (
    id VARCHAR(32) PRIMARY KEY,
    job_id VARCHAR(32) NOT NULL,
    discord_message_id VARCHAR(255) NOT NULL,
    discord_thread_id VARCHAR(255) NOT NULL,
    text_content TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    discord_integration_id VARCHAR(32) NOT NULL,
    organization_id VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Step 3: Add foreign key constraints in production schema
ALTER TABLE claudecontrol.processed_discord_messages
ADD CONSTRAINT fk_processed_discord_messages_job_id
FOREIGN KEY (job_id) REFERENCES claudecontrol.jobs(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol.processed_discord_messages
ADD CONSTRAINT fk_processed_discord_messages_discord_integration_id
FOREIGN KEY (discord_integration_id) REFERENCES claudecontrol.discord_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol.processed_discord_messages
ADD CONSTRAINT fk_processed_discord_messages_organization_id
FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Step 4: Add foreign key constraints in test schema
ALTER TABLE claudecontrol_test.processed_discord_messages
ADD CONSTRAINT fk_processed_discord_messages_job_id_test
FOREIGN KEY (job_id) REFERENCES claudecontrol_test.jobs(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.processed_discord_messages
ADD CONSTRAINT fk_processed_discord_messages_discord_integration_id_test
FOREIGN KEY (discord_integration_id) REFERENCES claudecontrol_test.discord_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.processed_discord_messages
ADD CONSTRAINT fk_processed_discord_messages_organization_id_test
FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Step 5: Add check constraints for status values in production schema
ALTER TABLE claudecontrol.processed_discord_messages
ADD CONSTRAINT check_processed_discord_message_status
CHECK (status IN ('QUEUED', 'IN_PROGRESS', 'COMPLETED'));

-- Step 6: Add check constraints for status values in test schema
ALTER TABLE claudecontrol_test.processed_discord_messages
ADD CONSTRAINT check_processed_discord_message_status_test
CHECK (status IN ('QUEUED', 'IN_PROGRESS', 'COMPLETED'));

-- Step 7: Create indexes for performance in production schema
CREATE INDEX idx_processed_discord_messages_job_id ON claudecontrol.processed_discord_messages(job_id);
CREATE INDEX idx_processed_discord_messages_status ON claudecontrol.processed_discord_messages(status);
CREATE INDEX idx_processed_discord_messages_discord_integration_id ON claudecontrol.processed_discord_messages(discord_integration_id);
CREATE INDEX idx_processed_discord_messages_organization_id ON claudecontrol.processed_discord_messages(organization_id);

-- Step 8: Create indexes for performance in test schema
CREATE INDEX idx_processed_discord_messages_job_id_test ON claudecontrol_test.processed_discord_messages(job_id);
CREATE INDEX idx_processed_discord_messages_status_test ON claudecontrol_test.processed_discord_messages(status);
CREATE INDEX idx_processed_discord_messages_discord_integration_id_test ON claudecontrol_test.processed_discord_messages(discord_integration_id);
CREATE INDEX idx_processed_discord_messages_organization_id_test ON claudecontrol_test.processed_discord_messages(organization_id);

-- Step 9: Create composite indexes for common queries in production schema
CREATE INDEX idx_processed_discord_messages_job_status ON claudecontrol.processed_discord_messages(job_id, status);
CREATE INDEX idx_processed_discord_messages_org_integration ON claudecontrol.processed_discord_messages(organization_id, discord_integration_id);

-- Step 10: Create composite indexes for common queries in test schema
CREATE INDEX idx_processed_discord_messages_job_status_test ON claudecontrol_test.processed_discord_messages(job_id, status);
CREATE INDEX idx_processed_discord_messages_org_integration_test ON claudecontrol_test.processed_discord_messages(organization_id, discord_integration_id);