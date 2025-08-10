-- Create discord_integrations table in production schema
CREATE TABLE claudecontrol.discord_integrations (
    id TEXT PRIMARY KEY,
    discord_guild_id TEXT NOT NULL,
    discord_guild_name TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT discord_integrations_discord_guild_id_unique UNIQUE (discord_guild_id),
    CONSTRAINT discord_integrations_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES claudecontrol.organizations(id) ON DELETE CASCADE
);

-- Create discord_integrations table in test schema
CREATE TABLE claudecontrol_test.discord_integrations (
    id TEXT PRIMARY KEY,
    discord_guild_id TEXT NOT NULL,
    discord_guild_name TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT discord_integrations_discord_guild_id_unique_test UNIQUE (discord_guild_id),
    CONSTRAINT discord_integrations_organization_id_fkey_test FOREIGN KEY (organization_id) REFERENCES claudecontrol_test.organizations(id) ON DELETE CASCADE
);