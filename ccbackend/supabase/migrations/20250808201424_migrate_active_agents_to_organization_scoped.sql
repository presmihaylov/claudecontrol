-- Migrate active_agents and agent_job_assignments from slack_integration_id to organization_id

-- Step 1: Add organization_id column to active_agents table in production schema
ALTER TABLE claudecontrol.active_agents 
ADD COLUMN organization_id TEXT;

-- Step 2: Add organization_id column to active_agents table in test schema
ALTER TABLE claudecontrol_test.active_agents 
ADD COLUMN organization_id TEXT;

-- Step 3: Populate organization_id in active_agents from related slack_integrations in production schema
UPDATE claudecontrol.active_agents aa 
SET organization_id = si.organization_id
FROM claudecontrol.slack_integrations si 
WHERE aa.slack_integration_id = si.id;

-- Step 4: Populate organization_id in active_agents from related slack_integrations in test schema
UPDATE claudecontrol_test.active_agents aa 
SET organization_id = si.organization_id
FROM claudecontrol_test.slack_integrations si 
WHERE aa.slack_integration_id = si.id;

-- Step 5: Add organization_id column to agent_job_assignments table in production schema
ALTER TABLE claudecontrol.agent_job_assignments 
ADD COLUMN organization_id TEXT;

-- Step 6: Add organization_id column to agent_job_assignments table in test schema
ALTER TABLE claudecontrol_test.agent_job_assignments 
ADD COLUMN organization_id TEXT;

-- Step 7: Populate organization_id in agent_job_assignments from related slack_integrations in production schema
UPDATE claudecontrol.agent_job_assignments aja 
SET organization_id = si.organization_id
FROM claudecontrol.slack_integrations si 
WHERE aja.slack_integration_id = si.id;

-- Step 8: Populate organization_id in agent_job_assignments from related slack_integrations in test schema
UPDATE claudecontrol_test.agent_job_assignments aja 
SET organization_id = si.organization_id
FROM claudecontrol_test.slack_integrations si 
WHERE aja.slack_integration_id = si.id;

-- Step 9: Add NOT NULL constraint to organization_id in active_agents in production schema
ALTER TABLE claudecontrol.active_agents 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 10: Add NOT NULL constraint to organization_id in active_agents in test schema
ALTER TABLE claudecontrol_test.active_agents 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 11: Add NOT NULL constraint to organization_id in agent_job_assignments in production schema
ALTER TABLE claudecontrol.agent_job_assignments 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 12: Add NOT NULL constraint to organization_id in agent_job_assignments in test schema
ALTER TABLE claudecontrol_test.agent_job_assignments 
ALTER COLUMN organization_id SET NOT NULL;

-- Step 13: Add foreign key constraint to organizations table in active_agents in production schema
ALTER TABLE claudecontrol.active_agents 
ADD CONSTRAINT active_agents_organization_id_fkey 
FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Step 14: Add foreign key constraint to organizations table in active_agents in test schema
ALTER TABLE claudecontrol_test.active_agents 
ADD CONSTRAINT active_agents_organization_id_fkey_test 
FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Step 15: Add foreign key constraint to organizations table in agent_job_assignments in production schema
ALTER TABLE claudecontrol.agent_job_assignments 
ADD CONSTRAINT agent_job_assignments_organization_id_fkey 
FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE;

-- Step 16: Add foreign key constraint to organizations table in agent_job_assignments in test schema
ALTER TABLE claudecontrol_test.agent_job_assignments 
ADD CONSTRAINT agent_job_assignments_organization_id_fkey_test 
FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE;

-- Step 17: Drop the unique constraint on (slack_integration_id, ccagent_id) in active_agents production schema
-- First, find and drop the existing unique constraint
DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    SELECT conname INTO constraint_name
    FROM pg_constraint 
    WHERE conrelid = 'claudecontrol.active_agents'::regclass 
    AND contype = 'u'
    AND array_length(conkey, 1) = 2;
    
    IF constraint_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE claudecontrol.active_agents DROP CONSTRAINT ' || constraint_name;
    END IF;
END $$;

-- Step 18: Drop the unique constraint on (slack_integration_id, ccagent_id) in active_agents test schema
DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    SELECT conname INTO constraint_name
    FROM pg_constraint 
    WHERE conrelid = 'claudecontrol_test.active_agents'::regclass 
    AND contype = 'u'
    AND array_length(conkey, 1) = 2;
    
    IF constraint_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE claudecontrol_test.active_agents DROP CONSTRAINT ' || constraint_name;
    END IF;
END $$;

-- Step 19: Add new unique constraint on (organization_id, ccagent_id) in active_agents production schema
ALTER TABLE claudecontrol.active_agents 
ADD CONSTRAINT active_agents_organization_id_ccagent_id_unique 
UNIQUE (organization_id, ccagent_id);

-- Step 20: Add new unique constraint on (organization_id, ccagent_id) in active_agents test schema
ALTER TABLE claudecontrol_test.active_agents 
ADD CONSTRAINT active_agents_organization_id_ccagent_id_unique_test 
UNIQUE (organization_id, ccagent_id);

-- Step 21: Drop slack_integration_id foreign key constraint from active_agents in production schema
ALTER TABLE claudecontrol.active_agents 
DROP CONSTRAINT IF EXISTS active_agents_slack_integration_id_fkey;

-- Step 22: Drop slack_integration_id foreign key constraint from active_agents in test schema
ALTER TABLE claudecontrol_test.active_agents 
DROP CONSTRAINT IF EXISTS active_agents_slack_integration_id_fkey;

-- Step 23: Drop slack_integration_id foreign key constraint from agent_job_assignments in production schema
ALTER TABLE claudecontrol.agent_job_assignments 
DROP CONSTRAINT IF EXISTS agent_job_assignments_slack_integration_id_fkey;

-- Step 24: Drop slack_integration_id foreign key constraint from agent_job_assignments in test schema
ALTER TABLE claudecontrol_test.agent_job_assignments 
DROP CONSTRAINT IF EXISTS agent_job_assignments_slack_integration_id_fkey;

-- Step 25: Drop index on slack_integration_id from active_agents in production schema
DROP INDEX IF EXISTS claudecontrol.idx_active_agents_slack_integration_id;

-- Step 26: Drop index on slack_integration_id from active_agents in test schema
DROP INDEX IF EXISTS claudecontrol_test.idx_active_agents_slack_integration_id;

-- Step 27: Drop index on slack_integration_id from agent_job_assignments in production schema
DROP INDEX IF EXISTS claudecontrol.idx_agent_job_assignments_slack_integration_id;

-- Step 28: Drop index on slack_integration_id from agent_job_assignments in test schema
DROP INDEX IF EXISTS claudecontrol_test.idx_agent_job_assignments_slack_integration_id;

-- Step 29: Drop slack_integration_id column from active_agents in production schema
ALTER TABLE claudecontrol.active_agents 
DROP COLUMN slack_integration_id;

-- Step 30: Drop slack_integration_id column from active_agents in test schema
ALTER TABLE claudecontrol_test.active_agents 
DROP COLUMN slack_integration_id;

-- Step 31: Drop slack_integration_id column from agent_job_assignments in production schema
ALTER TABLE claudecontrol.agent_job_assignments 
DROP COLUMN slack_integration_id;

-- Step 32: Drop slack_integration_id column from agent_job_assignments in test schema
ALTER TABLE claudecontrol_test.agent_job_assignments 
DROP COLUMN slack_integration_id;

-- Step 33: Create index on organization_id for active_agents in production schema
CREATE INDEX idx_active_agents_organization_id ON claudecontrol.active_agents(organization_id);

-- Step 34: Create index on organization_id for active_agents in test schema
CREATE INDEX idx_active_agents_organization_id_test ON claudecontrol_test.active_agents(organization_id);

-- Step 35: Create index on organization_id for agent_job_assignments in production schema
CREATE INDEX idx_agent_job_assignments_organization_id ON claudecontrol.agent_job_assignments(organization_id);

-- Step 36: Create index on organization_id for agent_job_assignments in test schema
CREATE INDEX idx_agent_job_assignments_organization_id_test ON claudecontrol_test.agent_job_assignments(organization_id);