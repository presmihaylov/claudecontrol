-- Add organization_id column to slack_integrations table in production schema
ALTER TABLE claudecontrol.slack_integrations 
ADD COLUMN organization_id TEXT REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Add organization_id column to slack_integrations table in test schema
ALTER TABLE claudecontrol_test.slack_integrations
ADD COLUMN organization_id TEXT REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Add index for performance on slack_integrations organization_id
CREATE INDEX idx_slack_integrations_organization_id ON claudecontrol.slack_integrations(organization_id);
CREATE INDEX idx_slack_integrations_organization_id_test ON claudecontrol_test.slack_integrations(organization_id);