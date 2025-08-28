-- Add ssh_host field to ccagent_container_integrations table

-- Production schema
ALTER TABLE claudecontrol.ccagent_container_integrations 
ADD COLUMN ssh_host TEXT;

-- Test schema
ALTER TABLE claudecontrol_test.ccagent_container_integrations 
ADD COLUMN ssh_host TEXT;