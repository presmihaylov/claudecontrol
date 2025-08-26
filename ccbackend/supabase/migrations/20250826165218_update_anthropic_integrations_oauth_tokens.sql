-- Update anthropic_integrations table to support OAuth tokens

-- Production schema

-- Add new columns for OAuth tokens
ALTER TABLE claudecontrol.anthropic_integrations 
ADD COLUMN IF NOT EXISTS claude_code_access_token TEXT,
ADD COLUMN IF NOT EXISTS claude_code_refresh_token TEXT,
ADD COLUMN IF NOT EXISTS access_token_expires_at TIMESTAMP WITH TIME ZONE;

-- Drop the old constraint that required exactly one token type
ALTER TABLE claudecontrol.anthropic_integrations 
DROP CONSTRAINT IF EXISTS exactly_one_token;

-- Add new constraint that requires at least one authentication method
ALTER TABLE claudecontrol.anthropic_integrations 
ADD CONSTRAINT at_least_one_auth_method CHECK (
    anthropic_api_key IS NOT NULL OR 
    claude_code_oauth_token IS NOT NULL OR
    (claude_code_access_token IS NOT NULL AND claude_code_refresh_token IS NOT NULL)
);

-- Test schema

-- Add new columns for OAuth tokens
ALTER TABLE claudecontrol_test.anthropic_integrations 
ADD COLUMN IF NOT EXISTS claude_code_access_token TEXT,
ADD COLUMN IF NOT EXISTS claude_code_refresh_token TEXT,
ADD COLUMN IF NOT EXISTS access_token_expires_at TIMESTAMP WITH TIME ZONE;

-- Drop the old constraint that required exactly one token type
ALTER TABLE claudecontrol_test.anthropic_integrations 
DROP CONSTRAINT IF EXISTS exactly_one_token;

-- Add new constraint that requires at least one authentication method
ALTER TABLE claudecontrol_test.anthropic_integrations 
ADD CONSTRAINT at_least_one_auth_method CHECK (
    anthropic_api_key IS NOT NULL OR 
    claude_code_oauth_token IS NOT NULL OR
    (claude_code_access_token IS NOT NULL AND claude_code_refresh_token IS NOT NULL)
);