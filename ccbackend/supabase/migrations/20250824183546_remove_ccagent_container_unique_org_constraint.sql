-- Remove unique constraint on organization_id to allow multiple CCAgent container integrations per organization

-- Production schema
DROP INDEX IF EXISTS claudecontrol.idx_ccagent_container_integrations_org_unique;

-- Test schema  
DROP INDEX IF EXISTS claudecontrol_test.idx_ccagent_container_integrations_org_unique_test;