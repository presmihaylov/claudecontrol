-- Add slack_integration_id back to jobs table (jobs need both organization_id for tenancy and slack_integration_id for API calls)
ALTER TABLE claudecontrol.jobs 
ADD COLUMN slack_integration_id TEXT REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.jobs
ADD COLUMN slack_integration_id TEXT REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;

-- Add index for performance
CREATE INDEX idx_jobs_slack_integration_id_new ON claudecontrol.jobs(slack_integration_id);
CREATE INDEX idx_jobs_slack_integration_id_test_new ON claudecontrol_test.jobs(slack_integration_id);