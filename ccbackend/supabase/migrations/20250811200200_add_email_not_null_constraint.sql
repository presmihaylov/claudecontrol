-- Add NOT NULL constraint to email column in production schema
ALTER TABLE claudecontrol.users ALTER COLUMN email SET NOT NULL;

-- Add NOT NULL constraint to email column in test schema
ALTER TABLE claudecontrol_test.users ALTER COLUMN email SET NOT NULL;