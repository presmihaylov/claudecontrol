-- Add ccagent_secret_key and ccagent_secret_key_generated_at to organizations table in production schema
ALTER TABLE claudecontrol.organizations 
ADD COLUMN ccagent_secret_key VARCHAR(512) NULL,
ADD COLUMN ccagent_secret_key_generated_at TIMESTAMPTZ NULL;

-- Add ccagent_secret_key and ccagent_secret_key_generated_at to organizations table in test schema
ALTER TABLE claudecontrol_test.organizations 
ADD COLUMN ccagent_secret_key VARCHAR(512) NULL,
ADD COLUMN ccagent_secret_key_generated_at TIMESTAMPTZ NULL;