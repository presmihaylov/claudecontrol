-- Migrate slack_integrations from user-scoped to organization-scoped

-- Add organization_id column to slack_integrations table in production schema
ALTER TABLE claudecontrol.slack_integrations 
ADD COLUMN organization_id TEXT;

-- Add organization_id column to slack_integrations table in test schema
ALTER TABLE claudecontrol_test.slack_integrations 
ADD COLUMN organization_id TEXT;

-- Migrate existing data: populate organization_id from user's organization_id in production schema
UPDATE claudecontrol.slack_integrations si 
SET organization_id = u.organization_id
FROM claudecontrol.users u 
WHERE si.user_id = u.id;

-- Migrate existing data: populate organization_id from user's organization_id in test schema
UPDATE claudecontrol_test.slack_integrations si 
SET organization_id = u.organization_id
FROM claudecontrol_test.users u 
WHERE si.user_id = u.id;

-- Add NOT NULL constraint to organization_id in production schema
ALTER TABLE claudecontrol.slack_integrations 
ALTER COLUMN organization_id SET NOT NULL;

-- Add NOT NULL constraint to organization_id in test schema
ALTER TABLE claudecontrol_test.slack_integrations 
ALTER COLUMN organization_id SET NOT NULL;

-- Add foreign key constraint to organizations table in production schema
ALTER TABLE claudecontrol.slack_integrations 
ADD CONSTRAINT slack_integrations_organization_id_fkey 
FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Add foreign key constraint to organizations table in test schema
ALTER TABLE claudecontrol_test.slack_integrations 
ADD CONSTRAINT slack_integrations_organization_id_fkey_test 
FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Drop user_id foreign key constraint in production schema
ALTER TABLE claudecontrol.slack_integrations 
DROP CONSTRAINT slack_integrations_user_id_fkey;

-- Drop user_id foreign key constraint in test schema
ALTER TABLE claudecontrol_test.slack_integrations 
DROP CONSTRAINT slack_integrations_user_id_fkey_test;

-- Drop user_id column from production schema
ALTER TABLE claudecontrol.slack_integrations 
DROP COLUMN user_id;

-- Drop user_id column from test schema
ALTER TABLE claudecontrol_test.slack_integrations 
DROP COLUMN user_id;