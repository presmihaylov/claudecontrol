-- Migration: Convert existing UUID data to ULID format for users and slack_integrations
-- This migration converts existing UUID data to proper ULID format with prefixes

-- Create a function to generate ULID-like IDs from timestamps
CREATE OR REPLACE FUNCTION generate_ulid_from_timestamp(ts TIMESTAMP WITH TIME ZONE, prefix TEXT)
RETURNS TEXT AS $$
DECLARE
    -- ULID timestamp part (first 10 characters)
    timestamp_part TEXT;
    -- Random part (remaining 16 characters)
    random_part TEXT;
    -- Characters used in ULID (Crockford's Base32)
    ulid_chars TEXT := '0123456789ABCDEFGHJKMNPQRSTVWXYZ';
    result TEXT;
    i INTEGER;
BEGIN
    -- Convert timestamp to milliseconds since epoch
    timestamp_part := LPAD(
        UPPER(TO_HEX(EXTRACT(EPOCH FROM ts)::BIGINT * 1000)), 
        10, '0'
    );
    
    -- Generate 16 random characters from ULID character set
    random_part := '';
    FOR i IN 1..16 LOOP
        random_part := random_part || SUBSTR(ulid_chars, 1 + (RANDOM() * 31)::INTEGER, 1);
    END LOOP;
    
    -- Combine prefix, timestamp (first 10 chars), and random part (16 chars) = 26 chars + prefix
    result := prefix || SUBSTR(timestamp_part, 1, 10) || random_part;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Step 1: Add temporary columns for new ULIDs
ALTER TABLE claudecontrol.users ADD COLUMN new_id TEXT;
ALTER TABLE claudecontrol.slack_integrations ADD COLUMN new_id TEXT, ADD COLUMN new_user_id TEXT;

ALTER TABLE claudecontrol_test.users ADD COLUMN new_id TEXT;
ALTER TABLE claudecontrol_test.slack_integrations ADD COLUMN new_id TEXT, ADD COLUMN new_user_id TEXT;

-- Step 2: Generate ULIDs for existing records using their creation timestamps
UPDATE claudecontrol.users 
SET new_id = generate_ulid_from_timestamp(created_at, 'u_');

UPDATE claudecontrol.slack_integrations 
SET new_id = generate_ulid_from_timestamp(created_at, 'si_');

-- Update slack_integrations with new user_id references
UPDATE claudecontrol.slack_integrations 
SET new_user_id = (SELECT new_id FROM claudecontrol.users WHERE users.id = slack_integrations.user_id);

-- Same for test schema
UPDATE claudecontrol_test.users 
SET new_id = generate_ulid_from_timestamp(created_at, 'u_');

UPDATE claudecontrol_test.slack_integrations 
SET new_id = generate_ulid_from_timestamp(created_at, 'si_');

UPDATE claudecontrol_test.slack_integrations 
SET new_user_id = (SELECT new_id FROM claudecontrol_test.users WHERE users.id = slack_integrations.user_id);

-- Step 3: Drop ALL foreign key constraints that reference primary keys we need to change
ALTER TABLE claudecontrol.slack_integrations DROP CONSTRAINT slack_integrations_user_id_fkey;
ALTER TABLE claudecontrol.active_agents DROP CONSTRAINT active_agents_slack_integration_id_fkey;
ALTER TABLE claudecontrol.jobs DROP CONSTRAINT jobs_slack_integration_id_fkey;
ALTER TABLE claudecontrol.agent_job_assignments DROP CONSTRAINT fk_agent_job_assignments_slack_integration;
ALTER TABLE claudecontrol.processed_slack_messages DROP CONSTRAINT processed_slack_messages_slack_integration_id_fkey;

ALTER TABLE claudecontrol_test.slack_integrations DROP CONSTRAINT slack_integrations_user_id_fkey_test;
ALTER TABLE claudecontrol_test.active_agents DROP CONSTRAINT active_agents_slack_integration_id_fkey;
ALTER TABLE claudecontrol_test.jobs DROP CONSTRAINT jobs_slack_integration_id_fkey;
ALTER TABLE claudecontrol_test.agent_job_assignments DROP CONSTRAINT fk_agent_job_assignments_slack_integration;
ALTER TABLE claudecontrol_test.processed_slack_messages DROP CONSTRAINT processed_slack_messages_slack_integration_id_fkey;

-- Step 4: Drop primary key constraints
ALTER TABLE claudecontrol.users DROP CONSTRAINT users_pkey;
ALTER TABLE claudecontrol.slack_integrations DROP CONSTRAINT slack_integrations_pkey;

ALTER TABLE claudecontrol_test.users DROP CONSTRAINT users_pkey;
ALTER TABLE claudecontrol_test.slack_integrations DROP CONSTRAINT slack_integrations_pkey;

-- Step 5: Swap the columns (production schema)
ALTER TABLE claudecontrol.users DROP COLUMN id;
ALTER TABLE claudecontrol.users RENAME COLUMN new_id TO id;

ALTER TABLE claudecontrol.slack_integrations DROP COLUMN user_id;
ALTER TABLE claudecontrol.slack_integrations RENAME COLUMN new_user_id TO user_id;
ALTER TABLE claudecontrol.slack_integrations DROP COLUMN id;
ALTER TABLE claudecontrol.slack_integrations RENAME COLUMN new_id TO id;

-- Swap the columns (test schema)
ALTER TABLE claudecontrol_test.users DROP COLUMN id;
ALTER TABLE claudecontrol_test.users RENAME COLUMN new_id TO id;

ALTER TABLE claudecontrol_test.slack_integrations DROP COLUMN user_id;
ALTER TABLE claudecontrol_test.slack_integrations RENAME COLUMN new_user_id TO user_id;
ALTER TABLE claudecontrol_test.slack_integrations DROP COLUMN id;
ALTER TABLE claudecontrol_test.slack_integrations RENAME COLUMN new_id TO id;

-- Step 6: Restore primary key constraints
ALTER TABLE claudecontrol.users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
ALTER TABLE claudecontrol.slack_integrations ADD CONSTRAINT slack_integrations_pkey PRIMARY KEY (id);

ALTER TABLE claudecontrol_test.users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
ALTER TABLE claudecontrol_test.slack_integrations ADD CONSTRAINT slack_integrations_pkey PRIMARY KEY (id);

-- Step 7: Restore ALL foreign key constraints
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
    ADD CONSTRAINT fk_agent_job_assignments_slack_integration 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol.processed_slack_messages 
    ADD CONSTRAINT processed_slack_messages_slack_integration_id_fkey 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol.slack_integrations(id) ON DELETE CASCADE;

-- Test schema foreign keys
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
    ADD CONSTRAINT fk_agent_job_assignments_slack_integration 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;

ALTER TABLE claudecontrol_test.processed_slack_messages 
    ADD CONSTRAINT processed_slack_messages_slack_integration_id_fkey 
        FOREIGN KEY (slack_integration_id) REFERENCES claudecontrol_test.slack_integrations(id) ON DELETE CASCADE;

-- Step 8: Clean up the temporary function
DROP FUNCTION generate_ulid_from_timestamp(TIMESTAMP WITH TIME ZONE, TEXT);