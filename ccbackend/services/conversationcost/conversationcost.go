package conversationcost

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"
	"github.com/shopspring/decimal"

	"ccbackend/appctx"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

// Claude API pricing per 1K tokens (approximate as of 2024)
const (
	ClaudeHaikuInputCostPer1K  = 0.00025  // $0.25 per 1M tokens
	ClaudeHaikuOutputCostPer1K = 0.00125  // $1.25 per 1M tokens
	ClaudeSonnetInputCostPer1K = 0.003    // $3.00 per 1M tokens  
	ClaudeSonnetOutputCostPer1K = 0.015   // $15.00 per 1M tokens
	ClaudeOpusInputCostPer1K   = 0.015    // $15.00 per 1M tokens
	ClaudeOpusOutputCostPer1K  = 0.075    // $75.00 per 1M tokens
)

type ConversationCostServiceImpl struct {
	conversationCostRepo *db.PostgresConversationCostRepository
}

func NewConversationCostService(repo *db.PostgresConversationCostRepository) *ConversationCostServiceImpl {
	return &ConversationCostServiceImpl{
		conversationCostRepo: repo,
	}
}

func (s *ConversationCostServiceImpl) TrackUsage(ctx context.Context, jobID string, inputTokens, outputTokens int) error {
	log.Printf("📋 Starting to track usage for job %s: input=%d, output=%d tokens", jobID, inputTokens, outputTokens)

	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if inputTokens < 0 || outputTokens < 0 {
		return fmt.Errorf("token counts cannot be negative")
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		return fmt.Errorf("organization not found in context")
	}

	// Check if cost record already exists
	existingCost, err := s.conversationCostRepo.GetConversationCostByJobID(ctx, org.ID, jobID)
	if err != nil {
		return fmt.Errorf("failed to check existing cost record: %w", err)
	}

	estimatedCost := s.EstimateCost(inputTokens, outputTokens)

	if existingCost != nil {
		// Update existing record by adding to current totals
		newInputTokens := existingCost.TotalInputTokens + inputTokens
		newOutputTokens := existingCost.TotalOutputTokens + outputTokens
		newEstimatedCost := existingCost.EstimatedCostUSD.Add(estimatedCost)

		_, err = s.conversationCostRepo.UpdateConversationCost(ctx, org.ID, jobID, 
			newInputTokens, newOutputTokens, newEstimatedCost)
		if err != nil {
			return fmt.Errorf("failed to update conversation cost: %w", err)
		}
	} else {
		// Create new cost record
		cost := &models.ConversationCost{
			ID:                core.NewID("cc"),
			OrganizationID:    org.ID,
			JobID:             jobID,
			TotalInputTokens:  inputTokens,
			TotalOutputTokens: outputTokens,
			EstimatedCostUSD:  estimatedCost,
		}

		if err := s.conversationCostRepo.CreateConversationCost(ctx, cost); err != nil {
			return fmt.Errorf("failed to create conversation cost: %w", err)
		}
	}

	log.Printf("📋 Completed successfully - tracked usage for job %s, cost: $%s", jobID, estimatedCost.String())
	return nil
}

func (s *ConversationCostServiceImpl) GetJobCosts(ctx context.Context, jobID string) (mo.Option[*models.ConversationCost], error) {
	log.Printf("📋 Starting to get job costs for job: %s", jobID)

	if jobID == "" {
		return mo.None[*models.ConversationCost](), fmt.Errorf("job ID cannot be empty")
	}

	org := appctx.GetOrganization(ctx)
	if org == nil {
		return mo.None[*models.ConversationCost](), fmt.Errorf("organization not found in context")
	}

	cost, err := s.conversationCostRepo.GetConversationCostByJobID(ctx, org.ID, jobID)
	if err != nil {
		return mo.None[*models.ConversationCost](), fmt.Errorf("failed to get job costs: %w", err)
	}

	if cost == nil {
		log.Printf("📋 Completed successfully - no cost record found for job: %s", jobID)
		return mo.None[*models.ConversationCost](), nil
	}

	log.Printf("📋 Completed successfully - retrieved cost record for job %s: $%s", jobID, cost.EstimatedCostUSD.String())
	return mo.Some(cost), nil
}

func (s *ConversationCostServiceImpl) UpdateJobCosts(ctx context.Context, jobID string, inputTokens, outputTokens int) error {
	log.Printf("📋 Starting to update job costs for job %s: input=%d, output=%d tokens", jobID, inputTokens, outputTokens)

	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if inputTokens < 0 || outputTokens < 0 {
		return fmt.Errorf("token counts cannot be negative")
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		return fmt.Errorf("organization not found in context")
	}

	estimatedCost := s.EstimateCost(inputTokens, outputTokens)

	_, err := s.conversationCostRepo.UpdateConversationCost(ctx, org.ID, jobID, 
		inputTokens, outputTokens, estimatedCost)
	if err != nil {
		return fmt.Errorf("failed to update conversation cost: %w", err)
	}

	log.Printf("📋 Completed successfully - updated cost for job %s: $%s", jobID, estimatedCost.String())
	return nil
}

func (s *ConversationCostServiceImpl) EstimateCost(inputTokens, outputTokens int) decimal.Decimal {
	// Using Claude Sonnet pricing as default middle-tier option
	inputCost := decimal.NewFromFloat(float64(inputTokens) * ClaudeSonnetInputCostPer1K / 1000)
	outputCost := decimal.NewFromFloat(float64(outputTokens) * ClaudeSonnetOutputCostPer1K / 1000)
	return inputCost.Add(outputCost)
}

func (s *ConversationCostServiceImpl) GetOrganizationCosts(ctx context.Context) ([]*models.ConversationCost, error) {
	log.Printf("📋 Starting to get organization costs")

	org := appctx.GetOrganization(ctx)
	if org == nil {
		return nil, fmt.Errorf("organization not found in context")
	}

	costs, err := s.conversationCostRepo.GetConversationCostsByOrganizationID(ctx, org.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization costs: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d cost records for organization", len(costs))
	return costs, nil
}

func (s *ConversationCostServiceImpl) DeleteJobCosts(ctx context.Context, jobID string) error {
	log.Printf("📋 Starting to delete job costs for job: %s", jobID)

	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		return fmt.Errorf("organization not found in context")
	}

	if err := s.conversationCostRepo.DeleteConversationCostByJobID(ctx, org.ID, jobID); err != nil {
		return fmt.Errorf("failed to delete job costs: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted cost record for job: %s", jobID)
	return nil
}