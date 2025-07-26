-- Create slack_integrations table in production schema
CREATE TABLE claudecontrol.slack_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slack_team_id VARCHAR(255) NOT NULL,
    slack_auth_token VARCHAR(512) NOT NULL,
    slack_team_name VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT slack_integrations_slack_team_id_unique UNIQUE (slack_team_id),
    CONSTRAINT slack_integrations_user_id_fkey FOREIGN KEY (user_id) REFERENCES claudecontrol.users(id) ON DELETE CASCADE
);

-- Create slack_integrations table in test schema
CREATE TABLE claudecontrol_test.slack_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slack_team_id VARCHAR(255) NOT NULL,
    slack_auth_token VARCHAR(512) NOT NULL,
    slack_team_name VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT slack_integrations_slack_team_id_unique_test UNIQUE (slack_team_id),
    CONSTRAINT slack_integrations_user_id_fkey_test FOREIGN KEY (user_id) REFERENCES claudecontrol_test.users(id) ON DELETE CASCADE
);