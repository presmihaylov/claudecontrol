package core

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sort"

	"ccbackend/models"
)

// ProcessJobComplete handles job completion from an agent
func (s *CoreUseCase) ProcessJobComplete(
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
	if err := s.validateJobBelongsToAgent(ctx, agent.ID, jobID, organizationID); err != nil {
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
		if err := s.jobsService.DeleteJob(ctx, jobID, slackIntegrationID, organizationID); err != nil {
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

// validateJobBelongsToAgent checks if a job is assigned to the specified agent
func (s *CoreUseCase) validateJobBelongsToAgent(
	ctx context.Context,
	agentID, jobID string,
	organizationID string,
) error {
	agentJobs, err := s.agentsService.GetActiveAgentJobAssignments(ctx, agentID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get jobs for agent: %w", err)
	}
	if slices.Contains(agentJobs, jobID) {
		log.Printf("✅ Agent %s is assigned to job %s", agentID, jobID)
		return nil
	}

	log.Printf("❌ Agent %s is not assigned to job %s", agentID, jobID)
	return fmt.Errorf("agent %s is not assigned to job %s", agentID, jobID)
}

// getOrAssignAgentForJob gets an existing agent assignment or assigns a new agent to a job
func (s *CoreUseCase) getOrAssignAgentForJob(
	ctx context.Context,
	job *models.Job,
	threadTS, organizationID string,
) (string, error) {
	// Check if this job is already assigned to an agent
	maybeExistingAgent, err := s.agentsService.GetAgentByJobID(ctx, job.ID, organizationID)
	if err != nil {
		// Some error occurred
		log.Printf("❌ Failed to check for existing agent assignment: %v", err)
		return "", fmt.Errorf("failed to check for existing agent assignment: %w", err)
	}

	if !maybeExistingAgent.IsPresent() {
		// Job not assigned to any agent yet - need to assign to an available agent
		return s.assignJobToAvailableAgent(ctx, job, threadTS, organizationID)
	}

	existingAgent := maybeExistingAgent.MustGet()

	// Job is already assigned to an agent - verify it still has an active connection
	connectedClientIDs := s.wsClient.GetClientIDs()
	if s.agentsService.CheckAgentHasActiveConnection(existingAgent, connectedClientIDs) {
		log.Printf(
			"🔄 Job %s already assigned to agent %s with active connection, routing message to existing agent",
			job.ID,
			existingAgent.ID,
		)
		return existingAgent.WSConnectionID, nil
	}

	// Existing agent doesn't have active connection - return error to signal no available agents
	log.Printf("⚠️ Job %s assigned to agent %s but no active WebSocket connection found", job.ID, existingAgent.ID)
	return "", fmt.Errorf("no active agents available for job assignment")
}

// assignJobToAvailableAgent attempts to assign a job to the least loaded available agent
// Returns the WebSocket client ID if successful, empty string if no agents available, or error on failure
func (s *CoreUseCase) assignJobToAvailableAgent(
	ctx context.Context,
	job *models.Job,
	threadTS, organizationID string,
) (string, error) {
	log.Printf("📝 Job %s not yet assigned, looking for any active agent", job.ID)

	clientID, assigned, err := s.tryAssignJobToAgent(ctx, job.ID, organizationID)
	if err != nil {
		return "", err
	}

	if !assigned {
		log.Printf("⚠️ No agents have active WebSocket connections")
		return "", fmt.Errorf("no agents with active WebSocket connections available for job assignment")
	}

	log.Printf("✅ Assigned job %s to agent for slack thread %s (agent can handle multiple jobs)", job.ID, threadTS)
	return clientID, nil
}

// tryAssignJobToAgent is a reusable function that attempts to assign a job to the least loaded available agent
// Returns (clientID, wasAssigned, error) where:
// - clientID: WebSocket connection ID of assigned agent (empty if not assigned)
// - wasAssigned: true if job was successfully assigned to an agent, false if no agents available
// - error: any error that occurred during the assignment process
func (s *CoreUseCase) tryAssignJobToAgent(
	ctx context.Context,
	jobID string,
	organizationID string,
) (string, bool, error) {
	// First check if this job is already assigned to an agent
	maybeExistingAgent, err := s.agentsService.GetAgentByJobID(ctx, jobID, organizationID)
	if err != nil {
		return "", false, fmt.Errorf("failed to check for existing agent assignment: %w", err)
	}

	if maybeExistingAgent.IsPresent() {
		existingAgent := maybeExistingAgent.MustGet()
		// Job is already assigned - check if agent still has active connection
		connectedClientIDs := s.wsClient.GetClientIDs()
		if s.agentsService.CheckAgentHasActiveConnection(existingAgent, connectedClientIDs) {
			log.Printf("🔄 Job %s already assigned to agent %s with active connection", jobID, existingAgent.ID)
			return existingAgent.WSConnectionID, true, nil
		}
		// Agent no longer has active connection - job remains assigned but can't process
		log.Printf("⚠️ Job %s assigned to agent %s but no active connection", jobID, existingAgent.ID)
		return "", false, nil
	}

	// Job not assigned - proceed with assignment
	// Get active WebSocket connections first
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("🔍 Found %d connected WebSocket clients", len(connectedClientIDs))

	// Get only agents with active connections using centralized service method
	connectedAgents, err := s.agentsService.GetConnectedActiveAgents(ctx, organizationID, connectedClientIDs)
	if err != nil {
		log.Printf("❌ Failed to get connected active agents: %v", err)
		return "", false, fmt.Errorf("failed to get connected active agents: %w", err)
	}

	if len(connectedAgents) == 0 {
		log.Printf("⚠️ No agents have active WebSocket connections")
		return "", false, nil
	}

	// Sort agents by load (number of assigned jobs) to select the least loaded agent
	sortedAgents, err := s.sortAgentsByLoad(ctx, connectedAgents, organizationID)
	if err != nil {
		log.Printf("❌ Failed to sort agents by load: %v", err)
		return "", false, fmt.Errorf("failed to sort agents by load: %w", err)
	}

	selectedAgent := sortedAgents[0].agent
	log.Printf("🎯 Selected agent %s with %d active jobs (least loaded)", selectedAgent.ID, sortedAgents[0].load)

	// Assign the job to the selected agent (agents can now handle multiple jobs simultaneously)
	if err := s.agentsService.AssignAgentToJob(ctx, selectedAgent.ID, jobID, organizationID); err != nil {
		log.Printf("❌ Failed to assign job %s to agent %s: %v", jobID, selectedAgent.ID, err)
		return "", false, fmt.Errorf("failed to assign job to agent: %w", err)
	}

	log.Printf("✅ Assigned job %s to agent %s", jobID, selectedAgent.ID)
	return selectedAgent.WSConnectionID, true, nil
}

// cleanupFailedJob handles the cleanup of a failed job including Slack notifications and database cleanup
// This is used both when an agent encounters an error and when an agent is disconnected
func (s *CoreUseCase) cleanupFailedJob(
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
		if err := s.jobsService.DeleteJob(ctx, job.ID, slackIntegrationID, organizationID); err != nil {
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

type agentWithLoad struct {
	agent *models.ActiveAgent
	load  int
}

// sortAgentsByLoad sorts agents by their current job load (ascending - least loaded first)
func (s *CoreUseCase) sortAgentsByLoad(
	ctx context.Context,
	agents []*models.ActiveAgent,
	organizationID string,
) ([]agentWithLoad, error) {
	agentsWithLoad := make([]agentWithLoad, 0, len(agents))

	for _, agent := range agents {
		// Get job IDs assigned to this agent
		jobIDs, err := s.agentsService.GetActiveAgentJobAssignments(ctx, agent.ID, organizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get job assignments for agent %s: %w", agent.ID, err)
		}

		jobCount := len(jobIDs)

		agentsWithLoad = append(agentsWithLoad, agentWithLoad{agent: agent, load: jobCount})
	}

	// Sort by load (ascending - least loaded first)
	sort.Slice(agentsWithLoad, func(i, j int) bool {
		return agentsWithLoad[i].load < agentsWithLoad[j].load
	})

	return agentsWithLoad, nil
}
