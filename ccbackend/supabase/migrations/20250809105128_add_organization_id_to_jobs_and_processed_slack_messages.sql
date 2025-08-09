-- Add organization_id to jobs and processed_slack_messages tables

-- Step 1: Add organization_id column to jobs table in production schema
ALTER TABLE claudecontrol.jobs 
ADD COLUMN organization_id TEXT;

-- Step 2: Add organization_id column to jobs table in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN organization_id TEXT;

-- Step 3: Populate organization_id in jobs from related slack_integrations in production schema
UPDATE claudecontrol.jobs j 
SET organization_id = si.organization_id
FROM claudecontrol.slack_integrations si 
WHERE j.slack_integration_id = si.id;

-- Step 4: Populate organization_id in jobs from related slack_integrations in test schema
UPDATE claudecontrol_test.jobs j 
SET organization_id = si.organization_id
FROM claudecontrol_test.slack_integrations si 
WHERE j.slack_integration_id = si.id;

-- Step 5: Add organization_id column to processed_slack_messages table in production schema
ALTER TABLE claudecontrol.processed_slack_messages 
ADD COLUMN organization_id TEXT;

-- Step 6: Add organization_id column to processed_slack_messages table in test schema
ALTER TABLE claudecontrol_test.processed_slack_messages 
ADD COLUMN organization_id TEXT;

-- Step 7: Populate organization_id in processed_slack_messages from related slack_integrations in production schema
UPDATE claudecontrol.processed_slack_messages psm 
SET organization_id = si.organization_id
FROM claudecontrol.slack_integrations si 
WHERE psm.slack_integration_id = si.id;

-- Step 8: Populate organization_id in processed_slack_messages from related slack_integrations in test schema
UPDATE claudecontrol_test.processed_slack_messages psm 
SET organization_id = si.organization_id
FROM claudecontrol_test.slack_integrations si 
WHERE psm.slack_integration_id = si.id;

-- Step 9: Add NOT NULL constraint to organization_id in jobs in production schema
ALTER TABLE claudecontrol.jobs 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 10: Add NOT NULL constraint to organization_id in jobs in test schema
ALTER TABLE claudecontrol_test.jobs 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 11: Add NOT NULL constraint to organization_id in processed_slack_messages in production schema
ALTER TABLE claudecontrol.processed_slack_messages 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 12: Add NOT NULL constraint to organization_id in processed_slack_messages in test schema
ALTER TABLE claudecontrol_test.processed_slack_messages 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 13: Add foreign key constraint to organizations table in jobs in production schema
ALTER TABLE claudecontrol.jobs 
ADD CONSTRAINT jobs_organization_id_fkey 
FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Step 14: Add foreign key constraint to organizations table in jobs in test schema
ALTER TABLE claudecontrol_test.jobs 
ADD CONSTRAINT jobs_organization_id_fkey_test 
FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Step 15: Add foreign key constraint to organizations table in processed_slack_messages in production schema
ALTER TABLE claudecontrol.processed_slack_messages 
ADD CONSTRAINT processed_slack_messages_organization_id_fkey 
FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Step 16: Add foreign key constraint to organizations table in processed_slack_messages in test schema
ALTER TABLE claudecontrol_test.processed_slack_messages 
ADD CONSTRAINT processed_slack_messages_organization_id_fkey_test 
FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Step 17: Create index on organization_id for jobs in production schema
CREATE INDEX idx_jobs_organization_id ON claudecontrol.jobs(organization_id);

-- Step 18: Create index on organization_id for jobs in test schema
CREATE INDEX idx_jobs_organization_id_test ON claudecontrol_test.jobs(organization_id);

-- Step 19: Create index on organization_id for processed_slack_messages in production schema
CREATE INDEX idx_processed_slack_messages_organization_id ON claudecontrol.processed_slack_messages(organization_id);

-- Step 20: Create index on organization_id for processed_slack_messages in test schema
CREATE INDEX idx_processed_slack_messages_organization_id_test ON claudecontrol_test.processed_slack_messages(organization_id);