-- Add organization_id column to users table in production schema
ALTER TABLE claudecontrol.users 
ADD COLUMN organization_id TEXT REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Add organization_id column to users table in test schema  
ALTER TABLE claudecontrol_test.users
ADD COLUMN organization_id TEXT REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Add index for performance on users organization_id
CREATE INDEX idx_users_organization_id ON claudecontrol.users(organization_id);
CREATE INDEX idx_users_organization_id_test ON claudecontrol_test.users(organization_id);