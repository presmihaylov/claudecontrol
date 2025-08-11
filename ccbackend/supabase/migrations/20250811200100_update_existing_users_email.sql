-- Update existing users with NULL email to have 'unknown' value in production schema
UPDATE claudecontrol.users SET email = 'unknown' WHERE email IS NULL;

-- Update existing users with NULL email to have 'unknown' value in test schema  
UPDATE claudecontrol_test.users SET email = 'unknown' WHERE email IS NULL;