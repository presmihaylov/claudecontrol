-- Add settings table for organization-scoped key-value storage
-- This migration creates the settings table with type-specific value columns

-- Production schema
BEGIN;

CREATE TABLE claudecontrol.settings (
    id TEXT PRIMARY KEY NOT NULL,
    organization_id TEXT NOT NULL,
    scope_type TEXT NOT NULL DEFAULT 'org',
    scope_id TEXT NOT NULL DEFAULT '',
    key TEXT NOT NULL,
    value_boolean BOOLEAN,
    value_string TEXT,
    value_stringarr TEXT[],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Foreign key constraint
    CONSTRAINT fk_settings_organization 
        FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE,
    
    -- Unique constraint to prevent duplicate settings
    CONSTRAINT uk_settings_org_scope_key 
        UNIQUE (organization_id, scope_type, scope_id, key),
    
    -- Check constraint: exactly one value column must be non-null
    CONSTRAINT ck_settings_single_value 
        CHECK (
            (value_boolean IS NOT NULL AND value_string IS NULL AND value_stringarr IS NULL) OR
            (value_boolean IS NULL AND value_string IS NOT NULL AND value_stringarr IS NULL) OR
            (value_boolean IS NULL AND value_string IS NULL AND value_stringarr IS NOT NULL)
        )
);

-- Create indexes for efficient lookups
CREATE INDEX idx_settings_organization_key ON claudecontrol.settings (organization_id, key);
CREATE INDEX idx_settings_org_scope ON claudecontrol.settings (organization_id, scope_type, scope_id);

-- Create trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION claudecontrol.update_settings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_settings_updated_at
    BEFORE UPDATE ON claudecontrol.settings
    FOR EACH ROW
    EXECUTE FUNCTION claudecontrol.update_settings_updated_at();

COMMIT;

-- Test schema
BEGIN;

CREATE TABLE claudecontrol_test.settings (
    id TEXT PRIMARY KEY NOT NULL,
    organization_id TEXT NOT NULL,
    scope_type TEXT NOT NULL DEFAULT 'org',
    scope_id TEXT NOT NULL DEFAULT '',
    key TEXT NOT NULL,
    value_boolean BOOLEAN,
    value_string TEXT,
    value_stringarr TEXT[],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Foreign key constraint
    CONSTRAINT fk_settings_organization 
        FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE,
    
    -- Unique constraint to prevent duplicate settings
    CONSTRAINT uk_settings_org_scope_key 
        UNIQUE (organization_id, scope_type, scope_id, key),
    
    -- Check constraint: exactly one value column must be non-null
    CONSTRAINT ck_settings_single_value 
        CHECK (
            (value_boolean IS NOT NULL AND value_string IS NULL AND value_stringarr IS NULL) OR
            (value_boolean IS NULL AND value_string IS NOT NULL AND value_stringarr IS NULL) OR
            (value_boolean IS NULL AND value_string IS NULL AND value_stringarr IS NOT NULL)
        )
);

-- Create indexes for efficient lookups
CREATE INDEX idx_settings_organization_key ON claudecontrol_test.settings (organization_id, key);
CREATE INDEX idx_settings_org_scope ON claudecontrol_test.settings (organization_id, scope_type, scope_id);

-- Create trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION claudecontrol_test.update_settings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_settings_updated_at
    BEFORE UPDATE ON claudecontrol_test.settings
    FOR EACH ROW
    EXECUTE FUNCTION claudecontrol_test.update_settings_updated_at();

COMMIT;