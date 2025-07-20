-- Create jobs table for production
CREATE TABLE claudecontrol.jobs (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    slack_thread_ts TEXT NOT NULL,
    slack_channel_id TEXT NOT NULL
);

-- Create jobs table for test schema
CREATE TABLE claudecontrol_test.jobs (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    slack_thread_ts TEXT NOT NULL,
    slack_channel_id TEXT NOT NULL
);