-- Rename and reorganize anthropic_integrations OAuth columns
-- This migration:
-- 1. Removes the original claude_code_oauth_token column (authorization code)
-- 2. Renames claude_code_access_token to claude_code_oauth_token  
-- 3. Renames claude_code_refresh_token to claude_code_oauth_refresh_token
-- 4. Renames access_token_expires_at to claude_code_oauth_token_expires_at
-- 5. Updates the constraint to use new column names

-- Production schema
BEGIN;

-- Step 1: Add new columns with final names
ALTER TABLE claudecontrol.anthropic_integrations 
ADD COLUMN IF NOT EXISTS claude_code_oauth_refresh_token TEXT,
ADD COLUMN IF NOT EXISTS claude_code_oauth_token_expires_at TIMESTAMP WITH TIME ZONE;

-- Step 2: Copy data from old columns to new columns
-- Copy access_token data to the oauth_token column (overwriting the authorization code)
-- Copy refresh_token data to the new refresh_token column  
-- Copy expiry data to the new expiry column
UPDATE claudecontrol.anthropic_integrations 
SET claude_code_oauth_token = claude_code_access_token,
    claude_code_oauth_refresh_token = claude_code_refresh_token,
    claude_code_oauth_token_expires_at = access_token_expires_at;

-- Step 3: Drop old constraint
ALTER TABLE claudecontrol.anthropic_integrations 
DROP CONSTRAINT IF EXISTS at_least_one_auth_method;

-- Step 4: Drop old columns
ALTER TABLE claudecontrol.anthropic_integrations 
DROP COLUMN IF EXISTS claude_code_access_token,
DROP COLUMN IF EXISTS claude_code_refresh_token,
DROP COLUMN IF EXISTS access_token_expires_at;

-- Step 5: Add new constraint with updated column names
ALTER TABLE claudecontrol.anthropic_integrations 
ADD CONSTRAINT at_least_one_auth_method CHECK (
    anthropic_api_key IS NOT NULL OR 
    (claude_code_oauth_token IS NOT NULL AND claude_code_oauth_refresh_token IS NOT NULL)
);

COMMIT;

-- Test schema
BEGIN;

-- Step 1: Add new columns with final names
ALTER TABLE claudecontrol_test.anthropic_integrations 
ADD COLUMN IF NOT EXISTS claude_code_oauth_refresh_token TEXT,
ADD COLUMN IF NOT EXISTS claude_code_oauth_token_expires_at TIMESTAMP WITH TIME ZONE;

-- Step 2: Copy data from old columns to new columns
UPDATE claudecontrol_test.anthropic_integrations 
SET claude_code_oauth_token = claude_code_access_token,
    claude_code_oauth_refresh_token = claude_code_refresh_token,
    claude_code_oauth_token_expires_at = access_token_expires_at;

-- Step 3: Drop old constraint
ALTER TABLE claudecontrol_test.anthropic_integrations 
DROP CONSTRAINT IF EXISTS at_least_one_auth_method;

-- Step 4: Drop old columns
ALTER TABLE claudecontrol_test.anthropic_integrations 
DROP COLUMN IF EXISTS claude_code_access_token,
DROP COLUMN IF EXISTS claude_code_refresh_token,
DROP COLUMN IF EXISTS access_token_expires_at;

-- Step 5: Add new constraint with updated column names
ALTER TABLE claudecontrol_test.anthropic_integrations 
ADD CONSTRAINT at_least_one_auth_method CHECK (
    anthropic_api_key IS NOT NULL OR 
    (claude_code_oauth_token IS NOT NULL AND claude_code_oauth_refresh_token IS NOT NULL)
);

COMMIT;