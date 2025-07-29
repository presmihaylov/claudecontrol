-- Drop assigned_job_id column from active_agents table in production schema
ALTER TABLE claudecontrol.active_agents DROP COLUMN assigned_job_id;

-- Drop assigned_job_id column from active_agents table in test schema
ALTER TABLE claudecontrol_test.active_agents DROP COLUMN assigned_job_id;