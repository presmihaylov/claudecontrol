-- Migration: Change all UUID columns to VARCHAR(28) for ULID support
-- This migration assumes the database will be reset and contains no existing data

-- Drop foreign key constraints first
ALTER TABLE claudecontrol.agent_job_assignments 
    DROP CONSTRAINT agent_job_assignments_agent_id_fkey,
    DROP CONSTRAINT agent_job_assignments_job_id_fkey;

ALTER TABLE claudecontrol.processed_slack_messages 
    DROP CONSTRAINT processed_slack_messages_job_id_fkey;

ALTER TABLE claudecontrol_test.agent_job_assignments 
    DROP CONSTRAINT agent_job_assignments_agent_id_fkey,
    DROP CONSTRAINT agent_job_assignments_job_id_fkey;

ALTER TABLE claudecontrol_test.processed_slack_messages 
    DROP CONSTRAINT processed_slack_messages_job_id_fkey;

-- Update users table
ALTER TABLE claudecontrol.users 
    ALTER COLUMN id TYPE VARCHAR(28);

ALTER TABLE claudecontrol_test.users 
    ALTER COLUMN id TYPE VARCHAR(28);

-- Update slack_integrations table
ALTER TABLE claudecontrol.slack_integrations 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN user_id TYPE VARCHAR(28);

ALTER TABLE claudecontrol_test.slack_integrations 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN user_id TYPE VARCHAR(28);

-- Update active_agents table
ALTER TABLE claudecontrol.active_agents 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN agent_id TYPE VARCHAR(28),
    ALTER COLUMN slack_integration_id TYPE VARCHAR(28);

ALTER TABLE claudecontrol_test.active_agents 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN agent_id TYPE VARCHAR(28),
    ALTER COLUMN slack_integration_id TYPE VARCHAR(28);

-- Update jobs table
ALTER TABLE claudecontrol.jobs 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN slack_integration_id TYPE VARCHAR(28);

ALTER TABLE claudecontrol_test.jobs 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN slack_integration_id TYPE VARCHAR(28);

-- Update processed_slack_messages table
ALTER TABLE claudecontrol.processed_slack_messages 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN job_id TYPE VARCHAR(28),
    ALTER COLUMN slack_integration_id TYPE VARCHAR(28);

ALTER TABLE claudecontrol_test.processed_slack_messages 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN job_id TYPE VARCHAR(28),
    ALTER COLUMN slack_integration_id TYPE VARCHAR(28);

-- Update agent_job_assignments table
ALTER TABLE claudecontrol.agent_job_assignments 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN agent_id TYPE VARCHAR(28),
    ALTER COLUMN job_id TYPE VARCHAR(28),
    ALTER COLUMN slack_integration_id TYPE VARCHAR(28);

ALTER TABLE claudecontrol_test.agent_job_assignments 
    ALTER COLUMN id TYPE VARCHAR(28),
    ALTER COLUMN agent_id TYPE VARCHAR(28),
    ALTER COLUMN job_id TYPE VARCHAR(28),
    ALTER COLUMN slack_integration_id TYPE VARCHAR(28);

-- Re-add foreign key constraints
ALTER TABLE claudecontrol.agent_job_assignments 
    ADD CONSTRAINT agent_job_assignments_agent_id_fkey 
        FOREIGN KEY (agent_id) REFERENCES claudecontrol.active_agents(id) ON DELETE CASCADE,
    ADD CONSTRAINT agent_job_assignments_job_id_fkey 
        FOREIGN KEY (job_id) REFERENCES claudecontrol.jobs(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol.processed_slack_messages 
    ADD CONSTRAINT processed_slack_messages_job_id_fkey 
        FOREIGN KEY (job_id) REFERENCES claudecontrol.jobs(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.agent_job_assignments 
    ADD CONSTRAINT agent_job_assignments_agent_id_fkey 
        FOREIGN KEY (agent_id) REFERENCES claudecontrol_test.active_agents(id) ON DELETE CASCADE,
    ADD CONSTRAINT agent_job_assignments_job_id_fkey 
        FOREIGN KEY (job_id) REFERENCES claudecontrol_test.jobs(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.processed_slack_messages 
    ADD CONSTRAINT processed_slack_messages_job_id_fkey 
        FOREIGN KEY (job_id) REFERENCES claudecontrol_test.jobs(id) ON DELETE CASCADE;