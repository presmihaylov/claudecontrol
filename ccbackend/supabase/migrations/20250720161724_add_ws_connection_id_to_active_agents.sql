-- Add ws_connection_id field to active_agents table
ALTER TABLE claudecontrol.active_agents 
ADD COLUMN ws_connection_id VARCHAR(255) NOT NULL;

-- Add ws_connection_id field to test schema as well
ALTER TABLE claudecontrol_test.active_agents 
ADD COLUMN ws_connection_id VARCHAR(255) NOT NULL;