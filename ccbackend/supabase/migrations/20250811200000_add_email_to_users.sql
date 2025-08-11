-- Add email column to users table in production schema
ALTER TABLE claudecontrol.users ADD COLUMN email TEXT;

-- Add email column to users table in test schema
ALTER TABLE claudecontrol_test.users ADD COLUMN email TEXT;