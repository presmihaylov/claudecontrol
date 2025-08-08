-- Migrate agents from slack_integration_id to organization_id scoping
-- This migration removes the arbitrary link between agents and slack integrations
-- and makes agents organization-scoped, enabling multi-slack integration support

-- Production Schema Migration

-- 1. Clear all active agents and job assignments (data can be dropped)
DELETE FROM claudecontrol.agent_job_assignments WHERE 1=1;
DELETE FROM claudecontrol.active_agents WHERE 1=1;

-- 2. Drop constraints and indexes related to slack_integration_id
ALTER TABLE claudecontrol.active_agents 
    DROP CONSTRAINT IF EXISTS unique_slack_integration_ccagent_id;
ALTER TABLE claudecontrol.active_agents 
    DROP CONSTRAINT IF EXISTS active_agents_slack_integration_id_fkey;
DROP INDEX IF EXISTS idx_active_agents_slack_integration_id;

-- 3. Remove slack_integration_id from active_agents
ALTER TABLE claudecontrol.active_agents 
    DROP COLUMN slack_integration_id;

-- 4. Add organization_id to active_agents
ALTER TABLE claudecontrol.active_agents 
    ADD COLUMN organization_id TEXT NOT NULL 
    REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- 5. Create new unique constraint and index
ALTER TABLE claudecontrol.active_agents 
    ADD CONSTRAINT unique_organization_ccagent_id 
    UNIQUE (organization_id, ccagent_id);
CREATE INDEX idx_active_agents_organization_id ON claudecontrol.active_agents(organization_id);

-- 6. Remove slack_integration_id from agent_job_assignments (jobs are already org-scoped)
ALTER TABLE claudecontrol.agent_job_assignments 
    DROP COLUMN slack_integration_id;

-- Test Schema Migration (identical changes)

-- 1. Clear all active agents and job assignments  
DELETE FROM claudecontrol_test.agent_job_assignments WHERE 1=1;
DELETE FROM claudecontrol_test.active_agents WHERE 1=1;

-- 2. Drop constraints and indexes
ALTER TABLE claudecontrol_test.active_agents 
    DROP CONSTRAINT IF EXISTS unique_slack_integration_ccagent_id;
ALTER TABLE claudecontrol_test.active_agents 
    DROP CONSTRAINT IF EXISTS active_agents_slack_integration_id_fkey;
DROP INDEX IF EXISTS idx_active_agents_slack_integration_id;

-- 3. Remove slack_integration_id
ALTER TABLE claudecontrol_test.active_agents 
    DROP COLUMN slack_integration_id;

-- 4. Add organization_id  
ALTER TABLE claudecontrol_test.active_agents 
    ADD COLUMN organization_id TEXT NOT NULL 
    REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- 5. Create new constraint and index
ALTER TABLE claudecontrol_test.active_agents 
    ADD CONSTRAINT unique_organization_ccagent_id 
    UNIQUE (organization_id, ccagent_id);
CREATE INDEX idx_active_agents_organization_id ON claudecontrol_test.active_agents(organization_id);

-- 6. Remove slack_integration_id from agent_job_assignments
ALTER TABLE claudecontrol_test.agent_job_assignments 
    DROP COLUMN slack_integration_id;