-- Create connected_channels table for tracking Slack and Discord channels
-- with polymorphic support and default repository URL assignment

CREATE TABLE claudecontrol.connected_channels (
    id TEXT PRIMARY KEY,                           -- ULID with "cc_" prefix
    organization_id TEXT NOT NULL,                 -- Organization scoping
    channel_id TEXT NOT NULL,                      -- Slack channel ID or Discord channel ID
    channel_type TEXT NOT NULL CHECK (channel_type IN ('slack', 'discord')), -- Platform type
    default_repo_url TEXT,                         -- Repository URL from first available agent
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Foreign key to organizations table
    CONSTRAINT fk_connected_channels_organization
        FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE,

    -- Unique constraint per org to prevent duplicate channel tracking
    UNIQUE(organization_id, channel_id, channel_type)
);

-- Create indexes for efficient lookups
CREATE INDEX idx_connected_channels_org_id ON claudecontrol.connected_channels(organization_id);
CREATE INDEX idx_connected_channels_channel_lookup ON claudecontrol.connected_channels(organization_id, channel_id, channel_type);

-- Create the same table for test schema
CREATE TABLE claudecontrol_test.connected_channels (
    id TEXT PRIMARY KEY,                           -- ULID with "cc_" prefix
    organization_id TEXT NOT NULL,                 -- Organization scoping
    channel_id TEXT NOT NULL,                      -- Slack channel ID or Discord channel ID
    channel_type TEXT NOT NULL CHECK (channel_type IN ('slack', 'discord')), -- Platform type
    default_repo_url TEXT,                         -- Repository URL from first available agent
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Foreign key to organizations table
    CONSTRAINT fk_connected_channels_organization_test
        FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE,

    -- Unique constraint per org to prevent duplicate channel tracking
    UNIQUE(organization_id, channel_id, channel_type)
);

-- Create indexes for test schema
CREATE INDEX idx_connected_channels_org_id_test ON claudecontrol_test.connected_channels(organization_id);
CREATE INDEX idx_connected_channels_channel_lookup_test ON claudecontrol_test.connected_channels(organization_id, channel_id, channel_type);