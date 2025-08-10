package core

import (
	"testing"
	"time"

	"ccbackend/models"
)

// Helper function to create test data
func createTestAgent(id, wsConnID, orgID string) *models.ActiveAgent {
	return &models.ActiveAgent{
		ID:             id,
		WSConnectionID: wsConnID,
		OrganizationID: orgID,
		LastActiveAt:   time.Now(),
	}
}

func createTestJob(id, jobType, slackIntegrationID, orgID string) *models.Job {
	job := &models.Job{
		ID:             id,
		JobType:        models.JobType(jobType),
		OrganizationID: orgID,
	}
	
	if jobType == string(models.JobTypeSlack) && slackIntegrationID != "" {
		job.SlackPayload = &models.SlackJobPayload{
			IntegrationID: slackIntegrationID,
		}
	}
	
	return job
}