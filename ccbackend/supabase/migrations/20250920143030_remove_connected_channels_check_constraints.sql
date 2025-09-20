-- Remove CHECK constraints from connected_channels table
-- Validation should happen at runtime in the application, not at database level

-- Remove CHECK constraint from production schema
ALTER TABLE claudecontrol.connected_channels
DROP CONSTRAINT connected_channels_platform_check;

-- Remove CHECK constraint from test schema
ALTER TABLE claudecontrol_test.connected_channels
DROP CONSTRAINT connected_channels_platform_check_test;