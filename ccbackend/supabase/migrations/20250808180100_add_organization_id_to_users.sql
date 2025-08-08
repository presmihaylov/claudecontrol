-- Add organization_id column to users table in production schema
ALTER TABLE claudecontrol.users 
ADD COLUMN organization_id VARCHAR(26),
ADD CONSTRAINT users_organization_id_fkey 
FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Add organization_id column to users table in test schema
ALTER TABLE claudecontrol_test.users 
ADD COLUMN organization_id VARCHAR(26),
ADD CONSTRAINT users_organization_id_fkey_test 
FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;