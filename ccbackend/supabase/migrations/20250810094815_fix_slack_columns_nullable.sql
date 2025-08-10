-- Fix Slack columns to be nullable for polymorphic job support
-- This migration addresses NOT NULL constraints that prevent Discord job creation

-- Step 1: Remove NOT NULL constraints from Slack columns in production schema
ALTER TABLE claudecontrol.jobs 
ALTER COLUMN slack_thread_ts DROP NOT NULL,
ALTER COLUMN slack_channel_id DROP NOT NULL,
ALTER COLUMN slack_user_id DROP NOT NULL,
ALTER COLUMN slack_integration_id DROP NOT NULL;

-- Step 2: Remove NOT NULL constraints from Slack columns in test schema
ALTER TABLE claudecontrol_test.jobs 
ALTER COLUMN slack_thread_ts DROP NOT NULL,
ALTER COLUMN slack_channel_id DROP NOT NULL,
ALTER COLUMN slack_user_id DROP NOT NULL,
ALTER COLUMN slack_integration_id DROP NOT NULL;