-- Add cc_agent_system_secret_key to organizations table in production schema
ALTER TABLE claudecontrol.organizations 
ADD COLUMN cc_agent_system_secret_key TEXT NOT NULL DEFAULT '';

-- Add cc_agent_system_secret_key to organizations table in test schema
ALTER TABLE claudecontrol_test.organizations 
ADD COLUMN cc_agent_system_secret_key TEXT NOT NULL DEFAULT '';

-- Update all existing organizations to have a generated system secret key
-- This ensures all existing orgs get a key when the migration runs
UPDATE claudecontrol.organizations 
SET cc_agent_system_secret_key = 'sys_' || encode(gen_random_bytes(32), 'base64')
WHERE cc_agent_system_secret_key = '';

UPDATE claudecontrol_test.organizations 
SET cc_agent_system_secret_key = 'sys_' || encode(gen_random_bytes(32), 'base64')
WHERE cc_agent_system_secret_key = '';

-- Remove the default constraint now that we've populated existing records
ALTER TABLE claudecontrol.organizations 
ALTER COLUMN cc_agent_system_secret_key DROP DEFAULT;

ALTER TABLE claudecontrol_test.organizations 
ALTER COLUMN cc_agent_system_secret_key DROP DEFAULT;