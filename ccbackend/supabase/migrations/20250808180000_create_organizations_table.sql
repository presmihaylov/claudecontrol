-- Create organizations table in production schema
CREATE TABLE claudecontrol.organizations (
    id VARCHAR(26) PRIMARY KEY DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create organizations table in test schema  
CREATE TABLE claudecontrol_test.organizations (
    id VARCHAR(26) PRIMARY KEY DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);