-- Data migration for existing users to organizations
-- Create an organization for each existing user and assign them to it

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

-- Production schema: Create organizations for existing users and assign them
DO $$
DECLARE
    user_record RECORD;
    org_id TEXT;
BEGIN
    -- Loop through all existing users without organizations
    FOR user_record IN 
        SELECT id, created_at 
        FROM claudecontrol.users 
        WHERE organization_id IS NULL OR organization_id = ''
    LOOP
        -- Generate organization ULID using user's creation timestamp
        org_id := generate_ulid_from_timestamp(user_record.created_at, 'org_');
        
        -- Insert the organization
        INSERT INTO claudecontrol.organizations (id, created_at, updated_at)
        VALUES (org_id, user_record.created_at, NOW());
        
        -- Update the user with their organization_id
        UPDATE claudecontrol.users 
        SET organization_id = org_id, updated_at = NOW()
        WHERE id = user_record.id;
        
        RAISE NOTICE 'Created organization % for user %', org_id, user_record.id;
    END LOOP;
END $$;

-- Test schema: Create organizations for existing users and assign them
DO $$
DECLARE
    user_record RECORD;
    org_id TEXT;
BEGIN
    -- Loop through all existing users without organizations
    FOR user_record IN 
        SELECT id, created_at 
        FROM claudecontrol_test.users 
        WHERE organization_id IS NULL OR organization_id = ''
    LOOP
        -- Generate organization ULID using user's creation timestamp
        org_id := generate_ulid_from_timestamp(user_record.created_at, 'org_');
        
        -- Insert the organization
        INSERT INTO claudecontrol_test.organizations (id, created_at, updated_at)
        VALUES (org_id, user_record.created_at, NOW());
        
        -- Update the user with their organization_id
        UPDATE claudecontrol_test.users 
        SET organization_id = org_id, updated_at = NOW()
        WHERE id = user_record.id;
        
        RAISE NOTICE 'Created organization % for user %', org_id, user_record.id;
    END LOOP;
END $$;

-- Clean up the temporary function
DROP FUNCTION generate_ulid_from_timestamp(TIMESTAMP WITH TIME ZONE, TEXT);