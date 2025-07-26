-- Create users table in production schema
CREATE TABLE claudecontrol.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_provider VARCHAR(50) NOT NULL,
    auth_provider_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT users_auth_provider_id_unique UNIQUE (auth_provider, auth_provider_id)
);

-- Create users table in test schema
CREATE TABLE claudecontrol_test.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_provider VARCHAR(50) NOT NULL,
    auth_provider_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT users_auth_provider_id_unique_test UNIQUE (auth_provider, auth_provider_id)
);