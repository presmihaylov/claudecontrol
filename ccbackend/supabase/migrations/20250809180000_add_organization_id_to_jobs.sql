-- Migration: Add organization_id to jobs table and related tables

-- Step 1: Add organization_id column to jobs table in production schema
ALTER TABLE claudecontrol.jobs 
ADD COLUMN organization_id TEXT;

-- Step 2: Add organization_id column to jobs table in test schema  
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN organization_id TEXT;

-- Step 3: Populate organization_id from slack_integrations in production schema
UPDATE claudecontrol.jobs j 
SET organization_id = si.organization_id
FROM claudecontrol.slack_integrations si 
WHERE j.slack_integration_id = si.id;

-- Step 4: Populate organization_id from slack_integrations in test schema
UPDATE claudecontrol_test.jobs j 
SET organization_id = si.organization_id
FROM claudecontrol_test.slack_integrations si 
WHERE j.slack_integration_id = si.id;

-- Step 5: Add NOT NULL constraint in production schema
ALTER TABLE claudecontrol.jobs 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 6: Add NOT NULL constraint in test schema
ALTER TABLE claudecontrol_test.jobs 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 7: Add foreign key constraint in production schema
ALTER TABLE claudecontrol.jobs 
ADD CONSTRAINT jobs_organization_id_fkey 
FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Step 8: Add foreign key constraint in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD CONSTRAINT jobs_organization_id_fkey_test 
FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Step 9: Create performance index in production schema
CREATE INDEX idx_jobs_organization_id ON claudecontrol.jobs(organization_id);

-- Step 10: Create performance index in test schema
CREATE INDEX idx_jobs_organization_id ON claudecontrol_test.jobs(organization_id);

-- Step 11: Create composite indexes for common query patterns in production schema
CREATE INDEX idx_jobs_org_thread ON claudecontrol.jobs(organization_id, slack_thread_ts, slack_channel_id);
CREATE INDEX idx_jobs_org_updated ON claudecontrol.jobs(organization_id, updated_at);

-- Step 12: Create composite indexes for common query patterns in test schema
CREATE INDEX idx_jobs_org_thread ON claudecontrol_test.jobs(organization_id, slack_thread_ts, slack_channel_id);
CREATE INDEX idx_jobs_org_updated ON claudecontrol_test.jobs(organization_id, updated_at);

-- Step 13: Add organization_id column to processed_slack_messages table in production schema (if not already present)
-- Note: This may already exist from previous migrations, using conditional logic
DO $$ 
BEGIN 
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'claudecontrol' 
        AND table_name = 'processed_slack_messages' 
        AND column_name = 'organization_id'
    ) THEN
        ALTER TABLE claudecontrol.processed_slack_messages 
        ADD COLUMN organization_id TEXT;
        
        -- Populate from jobs relationship
        UPDATE claudecontrol.processed_slack_messages psm 
        SET organization_id = j.organization_id
        FROM claudecontrol.jobs j 
        WHERE psm.job_id = j.id;
        
        -- Add NOT NULL constraint
        ALTER TABLE claudecontrol.processed_slack_messages 
        ALTER COLUMN organization_id SET NOT NULL;
        
        -- Add foreign key constraint
        ALTER TABLE claudecontrol.processed_slack_messages 
        ADD CONSTRAINT processed_slack_messages_organization_id_fkey 
        FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;
        
        -- Create index
        CREATE INDEX idx_processed_slack_messages_organization_id ON claudecontrol.processed_slack_messages(organization_id);
    END IF;
END $$;

-- Step 14: Add organization_id column to processed_slack_messages table in test schema (if not already present)
DO $$ 
BEGIN 
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'claudecontrol_test' 
        AND table_name = 'processed_slack_messages' 
        AND column_name = 'organization_id'
    ) THEN
        ALTER TABLE claudecontrol_test.processed_slack_messages 
        ADD COLUMN organization_id TEXT;
        
        -- Populate from jobs relationship
        UPDATE claudecontrol_test.processed_slack_messages psm 
        SET organization_id = j.organization_id
        FROM claudecontrol_test.jobs j 
        WHERE psm.job_id = j.id;
        
        -- Add NOT NULL constraint
        ALTER TABLE claudecontrol_test.processed_slack_messages 
        ALTER COLUMN organization_id SET NOT NULL;
        
        -- Add foreign key constraint
        ALTER TABLE claudecontrol_test.processed_slack_messages 
        ADD CONSTRAINT processed_slack_messages_organization_id_fkey_test 
        FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;
        
        -- Create index
        CREATE INDEX idx_processed_slack_messages_organization_id ON claudecontrol_test.processed_slack_messages(organization_id);
    END IF;
END $$;