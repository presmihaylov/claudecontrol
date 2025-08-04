#!/bin/bash

cd ccbackend

# Fix agents_test.go
sed -i '' 's/CreateJob("test\.thread\.assigned", "C1234567890", slackIntegrationID)/CreateJob("test.thread.assigned", "C1234567890", "user1", slackIntegrationID)/g' services/agents_test.go
sed -i '' 's/CreateJob("test\.thread\.getbyid", "C1234567890", slackIntegrationID)/CreateJob("test.thread.getbyid", "C1234567890", "user1", slackIntegrationID)/g' services/agents_test.go
sed -i '' 's/CreateJob("test\.thread\.available", "C1234567890", slackIntegrationID)/CreateJob("test.thread.available", "C1234567890", "user1", slackIntegrationID)/g' services/agents_test.go
sed -i '' 's/CreateJob("test\.thread\.busy1", "C1111111111", slackIntegrationID)/CreateJob("test.thread.busy1", "C1111111111", "user1", slackIntegrationID)/g' services/agents_test.go
sed -i '' 's/CreateJob("test\.thread\.busy2", "C2222222222", slackIntegrationID)/CreateJob("test.thread.busy2", "C2222222222", "user1", slackIntegrationID)/g' services/agents_test.go

# Fix jobs_test.go - add user parameter as the third argument
sed -i '' 's/service\.CreateJob(slackThreadTS, slackChannelID, slackIntegrationID)/service.CreateJob(slackThreadTS, slackChannelID, "testuser", slackIntegrationID)/g' services/jobs_test.go

# Fix other patterns in jobs_test.go
sed -i '' 's/service\.CreateJob("", "C1234567890", slackIntegrationID)/service.CreateJob("", "C1234567890", "testuser", slackIntegrationID)/g' services/jobs_test.go
sed -i '' 's/service\.CreateJob("test\.thread\.456", "", slackIntegrationID)/service.CreateJob("test.thread.456", "", "testuser", slackIntegrationID)/g' services/jobs_test.go
sed -i '' 's/service\.CreateJob("test\.thread\.456", "C1234567890", "")/service.CreateJob("test.thread.456", "C1234567890", "testuser", "")/g' services/jobs_test.go
sed -i '' 's/service\.CreateJob("test\.thread\.789", "C9876543210", slackIntegrationID)/service.CreateJob("test.thread.789", "C9876543210", "testuser", slackIntegrationID)/g' services/jobs_test.go

# Fix patterns with jobsService prefix
sed -i '' 's/jobsService\.CreateJob(\([^,]*\), \([^,]*\), slackIntegrationID)/jobsService.CreateJob(\1, \2, "testuser", slackIntegrationID)/g' services/jobs_test.go
sed -i '' 's/jobsService\.CreateJob(\([^,]*\), \([^,]*\), slackIntegrationID2)/jobsService.CreateJob(\1, \2, "testuser", slackIntegrationID2)/g' services/jobs_test.go