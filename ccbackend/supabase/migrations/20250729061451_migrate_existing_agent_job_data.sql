-- Migrate existing assigned_job_id data to junction table for production schema
INSERT INTO claudecontrol.agent_job_assignments (id, agent_id, job_id, slack_integration_id, assigned_at)
SELECT 
    gen_random_uuid() as id,
    id as agent_id,
    assigned_job_id as job_id,
    slack_integration_id,
    created_at as assigned_at
FROM claudecontrol.active_agents 
WHERE assigned_job_id IS NOT NULL;

-- Migrate existing assigned_job_id data to junction table for test schema
INSERT INTO claudecontrol_test.agent_job_assignments (id, agent_id, job_id, slack_integration_id, assigned_at)
SELECT 
    gen_random_uuid() as id,
    id as agent_id,
    assigned_job_id as job_id,
    slack_integration_id,
    created_at as assigned_at
FROM claudecontrol_test.active_agents 
WHERE assigned_job_id IS NOT NULL;