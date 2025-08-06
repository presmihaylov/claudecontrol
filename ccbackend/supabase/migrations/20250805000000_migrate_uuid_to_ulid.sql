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

-- Drop all foreign key constraints that reference ID columns we're changing
ALTER TABLE claudecontrol.slack_integrations 
    DROP CONSTRAINT IF EXISTS slack_integrations_user_id_fkey;

ALTER TABLE claudecontrol.active_agents 
    DROP CONSTRAINT IF EXISTS active_agents_slack_integration_id_fkey;

ALTER TABLE claudecontrol.jobs 
    DROP CONSTRAINT IF EXISTS jobs_slack_integration_id_fkey;

ALTER TABLE claudecontrol.agent_job_assignments 
    DROP CONSTRAINT IF EXISTS agent_job_assignments_agent_id_fkey,
    DROP CONSTRAINT IF EXISTS agent_job_assignments_job_id_fkey,
    DROP CONSTRAINT IF EXISTS fk_agent_job_assignments_slack_integration;

ALTER TABLE claudecontrol.processed_slack_messages 
    DROP CONSTRAINT IF EXISTS processed_slack_messages_job_id_fkey,
    DROP CONSTRAINT IF EXISTS processed_slack_messages_slack_integration_id_fkey;

-- Test schema constraints
ALTER TABLE claudecontrol_test.slack_integrations 
    DROP CONSTRAINT IF EXISTS slack_integrations_user_id_fkey_test;

ALTER TABLE claudecontrol_test.active_agents 
    DROP CONSTRAINT IF EXISTS active_agents_slack_integration_id_fkey;

ALTER TABLE claudecontrol_test.jobs 
    DROP CONSTRAINT IF EXISTS jobs_slack_integration_id_fkey;

ALTER TABLE claudecontrol_test.agent_job_assignments 
    DROP CONSTRAINT IF EXISTS agent_job_assignments_agent_id_fkey,
    DROP CONSTRAINT IF EXISTS agent_job_assignments_job_id_fkey,
    DROP CONSTRAINT IF EXISTS fk_agent_job_assignments_slack_integration;

ALTER TABLE claudecontrol_test.processed_slack_messages 
    DROP CONSTRAINT IF EXISTS processed_slack_messages_job_id_fkey,
    DROP CONSTRAINT IF EXISTS processed_slack_messages_slack_integration_id_fkey;

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

-- Re-add all foreign key constraints
ALTER TABLE claudecontrol.slack_integrations 
    ADD CONSTRAINT slack_integrations_user_id_fkey 
        FOREIGN KEY (user_id) REFERENCES claudecontrol.users(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol.active_agents 
    ADD CONSTRAINT active_agents_slack_integration_id_fkey 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol.jobs 
    ADD CONSTRAINT jobs_slack_integration_id_fkey 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol.agent_job_assignments 
    ADD CONSTRAINT agent_job_assignments_agent_id_fkey 
        FOREIGN KEY (agent_id) REFERENCES claudecontrol.active_agents(id) ON DELETE CASCADE,
    ADD CONSTRAINT agent_job_assignments_job_id_fkey 
        FOREIGN KEY (job_id) REFERENCES claudecontrol.jobs(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_agent_job_assignments_slack_integration 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol.processed_slack_messages 
    ADD CONSTRAINT processed_slack_messages_job_id_fkey 
        FOREIGN KEY (job_id) REFERENCES claudecontrol.jobs(id) ON DELETE CASCADE,
    ADD CONSTRAINT processed_slack_messages_slack_integration_id_fkey 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

-- Test schema constraints
ALTER TABLE claudecontrol_test.slack_integrations 
    ADD CONSTRAINT slack_integrations_user_id_fkey_test 
        FOREIGN KEY (user_id) REFERENCES claudecontrol_test.users(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.active_agents 
    ADD CONSTRAINT active_agents_slack_integration_id_fkey 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.jobs 
    ADD CONSTRAINT jobs_slack_integration_id_fkey 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.agent_job_assignments 
    ADD CONSTRAINT agent_job_assignments_agent_id_fkey 
        FOREIGN KEY (agent_id) REFERENCES claudecontrol_test.active_agents(id) ON DELETE CASCADE,
    ADD CONSTRAINT agent_job_assignments_job_id_fkey 
        FOREIGN KEY (job_id) REFERENCES claudecontrol_test.jobs(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_agent_job_assignments_slack_integration 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.processed_slack_messages 
    ADD CONSTRAINT processed_slack_messages_job_id_fkey 
        FOREIGN KEY (job_id) REFERENCES claudecontrol_test.jobs(id) ON DELETE CASCADE,
    ADD CONSTRAINT processed_slack_messages_slack_integration_id_fkey 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;