-- Add ccagent_secret_key and ccagent_secret_key_generated_at to slack_integrations table in production schema
ALTER TABLE claudecontrol.slack_integrations 
ADD COLUMN ccagent_secret_key VARCHAR(512) NULL,
ADD COLUMN ccagent_secret_key_generated_at TIMESTAMPTZ NULL;

-- Add ccagent_secret_key and ccagent_secret_key_generated_at to slack_integrations table in test schema
ALTER TABLE claudecontrol_test.slack_integrations 
ADD COLUMN ccagent_secret_key VARCHAR(512) NULL,
ADD COLUMN ccagent_secret_key_generated_at TIMESTAMPTZ NULL;