-- Migrate existing ccagent secret keys from slack_integrations to organizations
-- This migration copies the first valid secret key found for each organization

-- Production schema migration
UPDATE claudecontrol.organizations o
SET 
    ccagent_secret_key = si.ccagent_secret_key,
    ccagent_secret_key_generated_at = si.ccagent_secret_key_generated_at
FROM claudecontrol.slack_integrations si
WHERE o.id = si.organization_id 
    AND si.ccagent_secret_key IS NOT NULL
    AND o.ccagent_secret_key IS NULL;

-- Test schema migration  
UPDATE claudecontrol_test.organizations o
SET 
    ccagent_secret_key = si.ccagent_secret_key,
    ccagent_secret_key_generated_at = si.ccagent_secret_key_generated_at
FROM claudecontrol_test.slack_integrations si
WHERE o.id = si.organization_id 
    AND si.ccagent_secret_key IS NOT NULL
    AND o.ccagent_secret_key IS NULL;