-- Add text_content column to processed_slack_messages table for production
ALTER TABLE claudecontrol.processed_slack_messages 
ADD COLUMN text_content TEXT NOT NULL DEFAULT '';

-- Add text_content column to processed_slack_messages table for test schema
ALTER TABLE claudecontrol_test.processed_slack_messages 
ADD COLUMN text_content TEXT NOT NULL DEFAULT '';