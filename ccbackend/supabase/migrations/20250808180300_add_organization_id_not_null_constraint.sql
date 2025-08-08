-- Add NOT NULL constraint to organization_id after data migration
-- This should be applied only after all existing users have organizations

-- Add NOT NULL constraint to users table in production schema
ALTER TABLE claudecontrol.users 
ALTER COLUMN organization_id SET NOT NULL;

-- Add NOT NULL constraint to users table in test schema  
ALTER TABLE claudecontrol_test.users 
ALTER COLUMN organization_id SET NOT NULL;