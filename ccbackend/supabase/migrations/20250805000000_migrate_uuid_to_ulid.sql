-- Migration: Change all UUID columns to TEXT for ULID support
-- This migration will clean up existing data and convert schema for ULID identifiers

-- Delete data from child tables only (preserve users and slack_integrations)
DELETE FROM claudecontrol.processed_slack_messages;
DELETE FROM claudecontrol.agent_job_assignments;
DELETE FROM claudecontrol.active_agents;
DELETE FROM claudecontrol.jobs;

DELETE FROM claudecontrol_test.processed_slack_messages;
DELETE FROM claudecontrol_test.agent_job_assignments;
DELETE FROM claudecontrol_test.active_agents;
DELETE FROM claudecontrol_test.jobs;

-- Drop foreign key constraints
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
    ALTER COLUMN id TYPE TEXT;

ALTER TABLE claudecontrol_test.users 
    ALTER COLUMN id TYPE TEXT;

-- Update slack_integrations table
ALTER TABLE claudecontrol.slack_integrations 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN user_id TYPE TEXT;

ALTER TABLE claudecontrol_test.slack_integrations 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN user_id TYPE TEXT;

-- Update active_agents table
ALTER TABLE claudecontrol.active_agents 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN agent_id TYPE TEXT,
    ALTER COLUMN slack_integration_id TYPE TEXT;

ALTER TABLE claudecontrol_test.active_agents 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN agent_id TYPE TEXT,
    ALTER COLUMN slack_integration_id TYPE TEXT;

-- Update jobs table
ALTER TABLE claudecontrol.jobs 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN slack_integration_id TYPE TEXT;

ALTER TABLE claudecontrol_test.jobs 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN slack_integration_id TYPE TEXT;

-- Update processed_slack_messages table
ALTER TABLE claudecontrol.processed_slack_messages 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN job_id TYPE TEXT,
    ALTER COLUMN slack_integration_id TYPE TEXT;

ALTER TABLE claudecontrol_test.processed_slack_messages 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN job_id TYPE TEXT,
    ALTER COLUMN slack_integration_id TYPE TEXT;

-- Update agent_job_assignments table
ALTER TABLE claudecontrol.agent_job_assignments 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN agent_id TYPE TEXT,
    ALTER COLUMN job_id TYPE TEXT,
    ALTER COLUMN slack_integration_id TYPE TEXT;

ALTER TABLE claudecontrol_test.agent_job_assignments 
    ALTER COLUMN id TYPE TEXT,
    ALTER COLUMN agent_id TYPE TEXT,
    ALTER COLUMN job_id TYPE TEXT,
    ALTER COLUMN slack_integration_id TYPE TEXT;

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