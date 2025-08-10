package discord

import (
	"context"
	"fmt"
	"log"

	"ccbackend/models"
	"ccbackend/utils"
)

// ProcessAssistantMessage handles assistant messages from agents and updates Discord accordingly
func (d *DiscordUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process assistant message from client %s", clientID)

	// Validate agent exists by WebSocket connection ID (agents are organization-scoped)
	maybeAgent, err := d.agentsService.GetAgentByWSConnectionID(ctx, clientID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", clientID)
		return fmt.Errorf("no agent found for client: %s", clientID)
	}

	// Get the specific job from the payload to find the Discord thread information
	utils.AssertInvariant(payload.JobID != "", "JobID is empty in AssistantMessage payload")

	jobID := payload.JobID

	// Get job directly using organization_id (optimization)
	maybeJob, err := d.jobsService.GetJobByID(ctx, jobID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf(
			"⚠️ Job %s not found - already completed manually or by another agent, skipping assistant message",
			jobID,
		)
		return nil
	}

	job := maybeJob.MustGet()

	// Ensure job has Discord payload
	if job.DiscordPayload == nil {
		log.Printf("❌ Job %s has no Discord payload", job.ID)
		return fmt.Errorf("job has no Discord payload")
	}

	// Get Discord integration ID from job
	discordIntegrationID := job.DiscordPayload.DiscordIntegrationID

	// Send the assistant message to Discord thread
	if err := d.sendDiscordMessage(
		ctx,
		discordIntegrationID,
		job.DiscordPayload.ChannelID,
		job.DiscordPayload.ThreadID,
		payload.Message,
	); err != nil {
		return fmt.Errorf("failed to send assistant message to Discord: %w", err)
	}

	log.Printf("📋 Completed successfully - processed assistant message for job %s", job.ID)
	return nil
}

// ProcessSystemMessage handles system messages from agents and updates Discord accordingly
func (d *DiscordUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process system message from client %s", clientID)

	// Validate agent exists by WebSocket connection ID (agents are organization-scoped)
	maybeAgent, err := d.agentsService.GetAgentByWSConnectionID(ctx, clientID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", clientID)
		return fmt.Errorf("no agent found for client: %s", clientID)
	}

	// Get the specific job from the payload to find the Discord thread information
	utils.AssertInvariant(payload.JobID != "", "JobID is empty in SystemMessage payload")

	jobID := payload.JobID

	// Get job directly using organization_id (optimization)
	maybeJob, err := d.jobsService.GetJobByID(ctx, jobID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf(
			"⚠️ Job %s not found - already completed manually or by another agent, skipping system message",
			jobID,
		)
		return nil
	}

	job := maybeJob.MustGet()

	// Ensure job has Discord payload
	if job.DiscordPayload == nil {
		log.Printf("❌ Job %s has no Discord payload", job.ID)
		return fmt.Errorf("job has no Discord payload")
	}

	// Get Discord integration ID from job
	discordIntegrationID := job.DiscordPayload.DiscordIntegrationID

	// Send the system message to Discord thread
	if err := d.sendSystemMessage(
		ctx,
		discordIntegrationID,
		job.DiscordPayload.ChannelID,
		job.DiscordPayload.ThreadID,
		payload.Message,
	); err != nil {
		return fmt.Errorf("failed to send system message to Discord: %w", err)
	}

	log.Printf("📋 Completed successfully - processed system message for job %s", job.ID)
	return nil
}