-- Drop ccagent_secret_key and ccagent_secret_key_generated_at from slack_integrations table
-- These fields have been migrated to the organizations table

-- Drop columns from production schema
ALTER TABLE claudecontrol.slack_integrations 
DROP COLUMN ccagent_secret_key,
DROP COLUMN ccagent_secret_key_generated_at;

-- Drop columns from test schema
ALTER TABLE claudecontrol_test.slack_integrations 
DROP COLUMN ccagent_secret_key,
DROP COLUMN ccagent_secret_key_generated_at;