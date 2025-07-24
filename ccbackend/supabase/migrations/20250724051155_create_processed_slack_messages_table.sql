-- Create processed_slack_messages table for production
CREATE TABLE claudecontrol.processed_slack_messages (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES claudecontrol.jobs(id) ON DELETE CASCADE,
    slack_channel_id TEXT NOT NULL,
    slack_ts TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('QUEUED', 'IN_PROGRESS', 'COMPLETED')),
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

-- Create processed_slack_messages table for test schema
CREATE TABLE claudecontrol_test.processed_slack_messages (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES claudecontrol_test.jobs(id) ON DELETE CASCADE,
    slack_channel_id TEXT NOT NULL,
    slack_ts TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('QUEUED', 'IN_PROGRESS', 'COMPLETED')),
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);