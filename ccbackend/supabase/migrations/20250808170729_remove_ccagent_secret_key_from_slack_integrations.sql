-- Remove ccagent_secret_key and ccagent_secret_key_generated_at from slack_integrations table

-- Remove from production schema
ALTER TABLE claudecontrol.slack_integrations 
DROP COLUMN ccagent_secret_key,
DROP COLUMN ccagent_secret_key_generated_at;

-- Remove from test schema
ALTER TABLE claudecontrol_test.slack_integrations 
DROP COLUMN ccagent_secret_key,
DROP COLUMN ccagent_secret_key_generated_at;