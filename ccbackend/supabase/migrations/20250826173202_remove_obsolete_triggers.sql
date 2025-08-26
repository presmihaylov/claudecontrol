-- Remove trigger functions and triggers for processed_discord_messages

-- Step 1: Drop trigger for production schema
DROP TRIGGER IF EXISTS trigger_update_processed_discord_messages_updated_at ON claudecontrol.processed_discord_messages;

-- Step 2: Drop function for production schema
DROP FUNCTION IF EXISTS claudecontrol.update_processed_discord_messages_updated_at();

-- Step 3: Drop trigger for test schema
DROP TRIGGER IF EXISTS trigger_update_processed_discord_messages_updated_at_test ON claudecontrol_test.processed_discord_messages;

-- Step 4: Drop function for test schema
DROP FUNCTION IF EXISTS claudecontrol_test.update_processed_discord_messages_updated_at();
