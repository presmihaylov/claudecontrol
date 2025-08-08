-- Migrate ccagent secret keys from slack_integrations to organizations
-- For each organization, use the most recently generated secret key

-- Production schema migration
UPDATE claudecontrol.organizations o
SET 
    ccagent_secret_key = latest_keys.ccagent_secret_key,
    ccagent_secret_key_generated_at = latest_keys.ccagent_secret_key_generated_at
FROM (
    SELECT DISTINCT ON (organization_id) 
        organization_id,
        ccagent_secret_key,
        ccagent_secret_key_generated_at
    FROM claudecontrol.slack_integrations
    WHERE ccagent_secret_key IS NOT NULL 
      AND ccagent_secret_key_generated_at IS NOT NULL
    ORDER BY organization_id, ccagent_secret_key_generated_at DESC
) AS latest_keys
WHERE o.id = latest_keys.organization_id;

-- Test schema migration  
UPDATE claudecontrol_test.organizations o
SET 
    ccagent_secret_key = latest_keys.ccagent_secret_key,
    ccagent_secret_key_generated_at = latest_keys.ccagent_secret_key_generated_at
FROM (
    SELECT DISTINCT ON (organization_id) 
        organization_id,
        ccagent_secret_key,
        ccagent_secret_key_generated_at
    FROM claudecontrol_test.slack_integrations
    WHERE ccagent_secret_key IS NOT NULL 
      AND ccagent_secret_key_generated_at IS NOT NULL
    ORDER BY organization_id, ccagent_secret_key_generated_at DESC
) AS latest_keys
WHERE o.id = latest_keys.organization_id;