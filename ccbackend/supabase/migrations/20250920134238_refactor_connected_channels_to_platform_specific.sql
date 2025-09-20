-- Refactor connected_channels table to use platform-specific fields instead of generic channel_id

-- Add new platform-specific columns to claudecontrol schema
ALTER TABLE claudecontrol.connected_channels
ADD COLUMN slack_team_id TEXT NULL,
ADD COLUMN slack_channel_id TEXT NULL,
ADD COLUMN discord_guild_id TEXT NULL,
ADD COLUMN discord_channel_id TEXT NULL;

-- Migrate existing data based on channel_type
UPDATE claudecontrol.connected_channels
SET slack_channel_id = channel_id
WHERE channel_type = 'slack';

UPDATE claudecontrol.connected_channels
SET discord_channel_id = channel_id
WHERE channel_type = 'discord';

-- Drop the old unique constraint
ALTER TABLE claudecontrol.connected_channels
DROP CONSTRAINT connected_channels_organization_id_channel_id_channel_type_key;

-- Drop the old columns
ALTER TABLE claudecontrol.connected_channels
DROP COLUMN channel_id,
DROP COLUMN channel_type;

-- Add new unique constraints for each platform
ALTER TABLE claudecontrol.connected_channels
ADD CONSTRAINT connected_channels_slack_unique
UNIQUE (organization_id, slack_team_id, slack_channel_id);

ALTER TABLE claudecontrol.connected_channels
ADD CONSTRAINT connected_channels_discord_unique
UNIQUE (organization_id, discord_guild_id, discord_channel_id);

-- Add check constraint to ensure only one platform's fields are populated
ALTER TABLE claudecontrol.connected_channels
ADD CONSTRAINT connected_channels_platform_check
CHECK (
    (slack_team_id IS NOT NULL AND slack_channel_id IS NOT NULL AND discord_guild_id IS NULL AND discord_channel_id IS NULL) OR
    (slack_team_id IS NULL AND slack_channel_id IS NULL AND discord_guild_id IS NOT NULL AND discord_channel_id IS NOT NULL)
);

-- Update indexes
DROP INDEX IF EXISTS claudecontrol.idx_connected_channels_channel_lookup;
CREATE INDEX idx_connected_channels_slack_lookup ON claudecontrol.connected_channels(organization_id, slack_team_id, slack_channel_id) WHERE slack_team_id IS NOT NULL;
CREATE INDEX idx_connected_channels_discord_lookup ON claudecontrol.connected_channels(organization_id, discord_guild_id, discord_channel_id) WHERE discord_guild_id IS NOT NULL;

-- Do the same for test schema
ALTER TABLE claudecontrol_test.connected_channels
ADD COLUMN slack_team_id TEXT NULL,
ADD COLUMN slack_channel_id TEXT NULL,
ADD COLUMN discord_guild_id TEXT NULL,
ADD COLUMN discord_channel_id TEXT NULL;

-- Migrate existing test data
UPDATE claudecontrol_test.connected_channels
SET slack_channel_id = channel_id
WHERE channel_type = 'slack';

UPDATE claudecontrol_test.connected_channels
SET discord_channel_id = channel_id
WHERE channel_type = 'discord';

-- Drop the old unique constraint for test schema
ALTER TABLE claudecontrol_test.connected_channels
DROP CONSTRAINT connected_channels_organization_id_channel_id_channel_type_key;

-- Drop the old columns for test schema
ALTER TABLE claudecontrol_test.connected_channels
DROP COLUMN channel_id,
DROP COLUMN channel_type;

-- Add new unique constraints for test schema
ALTER TABLE claudecontrol_test.connected_channels
ADD CONSTRAINT connected_channels_slack_unique_test
UNIQUE (organization_id, slack_team_id, slack_channel_id);

ALTER TABLE claudecontrol_test.connected_channels
ADD CONSTRAINT connected_channels_discord_unique_test
UNIQUE (organization_id, discord_guild_id, discord_channel_id);

-- Add check constraint for test schema
ALTER TABLE claudecontrol_test.connected_channels
ADD CONSTRAINT connected_channels_platform_check_test
CHECK (
    (slack_team_id IS NOT NULL AND slack_channel_id IS NOT NULL AND discord_guild_id IS NULL AND discord_channel_id IS NULL) OR
    (slack_team_id IS NULL AND slack_channel_id IS NULL AND discord_guild_id IS NOT NULL AND discord_channel_id IS NOT NULL)
);

-- Update indexes for test schema
DROP INDEX IF EXISTS claudecontrol_test.idx_connected_channels_channel_lookup_test;
CREATE INDEX idx_connected_channels_slack_lookup_test ON claudecontrol_test.connected_channels(organization_id, slack_team_id, slack_channel_id) WHERE slack_team_id IS NOT NULL;
CREATE INDEX idx_connected_channels_discord_lookup_test ON claudecontrol_test.connected_channels(organization_id, discord_guild_id, discord_channel_id) WHERE discord_guild_id IS NOT NULL;