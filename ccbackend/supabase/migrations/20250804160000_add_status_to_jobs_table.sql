-- Add status column to jobs table for production
ALTER TABLE claudecontrol.jobs 
ADD COLUMN status TEXT NOT NULL DEFAULT 'ACTIVE';

-- Add status column to jobs table for test schema
ALTER TABLE claudecontrol_test.jobs 
ADD COLUMN status TEXT NOT NULL DEFAULT 'ACTIVE';

-- Add index on status column for performance
CREATE INDEX idx_jobs_status ON claudecontrol.jobs(status);
CREATE INDEX idx_jobs_status_test ON claudecontrol_test.jobs(status);

-- Update any existing jobs to have ACTIVE status (this should be a no-op since we set DEFAULT)
UPDATE claudecontrol.jobs SET status = 'ACTIVE' WHERE status IS NULL;
UPDATE claudecontrol_test.jobs SET status = 'ACTIVE' WHERE status IS NULL;