package slack

import (
	"context"
	"fmt"
	"log"

	"ccbackend/models"
)

// ProcessJobComplete handles job completion from an agent
func (s *SlackUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	organizationID string,
) error {
	log.Printf(
		"📋 Starting to process job complete from client %s: JobID: %s, Reason: %s",
		clientID,
		payload.JobID,
		payload.Reason,
	)

	// Validate JobID is not empty
	if payload.JobID == "" {
		log.Printf("❌ Empty JobID from client %s", clientID)
		return fmt.Errorf("JobID cannot be empty")
	}

	jobID := payload.JobID

	// Get job directly using organization_id (optimization)
	maybeJob, err := s.jobsService.GetJobByID(ctx, jobID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - already completed manually or by another agent, skipping", jobID)
		return nil
	}

	job := maybeJob.MustGet()
	if job.SlackPayload == nil {
		log.Printf("❌ Job %s has no Slack payload", jobID)
		return fmt.Errorf("job has no Slack payload")
	}
	slackIntegrationID := job.SlackPayload.IntegrationID

	// Get the agent by WebSocket connection ID to verify ownership (agents are organization-scoped)
	maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, clientID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", clientID)
		return fmt.Errorf("no agent found for client: %s", clientID)
	}
	agent := maybeAgent.MustGet()

	// Validate that this agent is actually assigned to this job
	if err := s.agentsUseCase.ValidateJobBelongsToAgent(ctx, agent.ID, jobID, organizationID); err != nil {
		log.Printf("❌ Agent %s not assigned to job %s: %v", agent.ID, jobID, err)
		return fmt.Errorf("agent not assigned to job: %w", err)
	}

	// Set white_check_mark emoji on the top-level message to indicate job completion
	if err := s.updateSlackMessageReaction(ctx, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, "white_check_mark", slackIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update top-level message reaction for completed job %s: %v", jobID, err)
		// Don't return error - this is not critical to job completion
	}

	// Perform database operations within transaction
	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Unassign the agent from the job
		if err := s.agentsService.UnassignAgentFromJob(ctx, agent.ID, jobID, organizationID); err != nil {
			log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, jobID, err)
			return fmt.Errorf("failed to unassign agent from job: %w", err)
		}
		log.Printf("✅ Unassigned agent %s from completed job %s", agent.ID, jobID)

		// Delete the job and its associated processed messages
		if err := s.jobsService.DeleteJob(ctx, jobID, organizationID); err != nil {
			log.Printf("❌ Failed to delete completed job %s: %v", jobID, err)
			return fmt.Errorf("failed to delete completed job: %w", err)
		}
		log.Printf("🗑️ Deleted completed job %s", jobID)

		return nil
	}); err != nil {
		return fmt.Errorf("failed to complete job processing in transaction: %w", err)
	}

	// Send completion message to Slack thread with reason
	if err := s.sendSystemMessage(ctx, slackIntegrationID, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, payload.Reason); err != nil {
		log.Printf("❌ Failed to send completion message to Slack thread %s: %v", job.SlackPayload.ThreadTS, err)
		return fmt.Errorf("failed to send completion message to Slack: %w", err)
	}

	log.Printf("📤 Sent completion message to Slack thread %s: %s", job.SlackPayload.ThreadTS, payload.Reason)
	log.Printf("📋 Completed successfully - processed job complete for job %s", jobID)
	return nil
}

// CleanupFailedSlackJob handles the cleanup of a failed Slack job including Slack notifications and database cleanup
// This is exported so core use case can call it when deregistering agents
func (s *SlackUseCase) CleanupFailedSlackJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	failureMessage string,
) error {
	if job.SlackPayload == nil {
		log.Printf("❌ Job %s has no Slack payload", job.ID)
		return fmt.Errorf("job has no Slack payload")
	}
	slackIntegrationID := job.SlackPayload.IntegrationID
	organizationID := job.OrganizationID

	// Send failure message to Slack thread
	if err := s.sendSlackMessage(ctx, slackIntegrationID, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, failureMessage); err != nil {
		log.Printf("❌ Failed to send failure message to Slack thread %s: %v", job.SlackPayload.ThreadTS, err)
		// Continue with cleanup even if Slack message fails
	}

	// Update the top-level message emoji to :x:
	if err := s.updateSlackMessageReaction(ctx, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, "x", slackIntegrationID); err != nil {
		log.Printf("❌ Failed to update slack message reaction to :x: for failed job %s: %v", job.ID, err)
		// Continue with cleanup even if reaction update fails
	}

	// Perform database operations within transaction
	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// If agent ID is provided, unassign agent from job
		if agentID != "" {
			if err := s.agentsService.UnassignAgentFromJob(ctx, agentID, job.ID, organizationID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from job %s: %v", agentID, job.ID, err)
				return fmt.Errorf("failed to unassign agent from job: %w", err)
			}
			log.Printf("🔗 Unassigned agent %s from job %s", agentID, job.ID)
		}

		// Delete the job (use the job's slack integration and organization from the job)
		if err := s.jobsService.DeleteJob(ctx, job.ID, organizationID); err != nil {
			log.Printf("❌ Failed to delete job %s: %v", job.ID, err)
			return fmt.Errorf("failed to delete job: %w", err)
		}
		log.Printf("🗑️ Deleted job %s", job.ID)

		return nil
	}); err != nil {
		return fmt.Errorf("failed to cleanup job %s in transaction: %w", job.ID, err)
	}

	return nil
}
