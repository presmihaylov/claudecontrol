package conversationcontext

import (
	"context"
	"fmt"
	"log"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/samber/mo"

	"ccbackend/appctx"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

// Token limits for different Claude models
const (
	ClaudeHaikuMaxTokens  = 200000   // 200K tokens
	ClaudeSonnetMaxTokens = 200000   // 200K tokens
	ClaudeOpusMaxTokens   = 200000   // 200K tokens
	DefaultMaxTokens      = 200000   // Default to Sonnet limit
	SummarizationThreshold = 160000   // 80% of max tokens, trigger summarization
)

type ConversationContextServiceImpl struct {
	conversationContextRepo *db.PostgresConversationContextRepository
	tokenCounter           *core.TokenCounter
	defaultModel           anthropic.Model
}

func NewConversationContextService(repo *db.PostgresConversationContextRepository, tokenCounter *core.TokenCounter) *ConversationContextServiceImpl {
	return &ConversationContextServiceImpl{
		conversationContextRepo: repo,
		tokenCounter:           tokenCounter,
		defaultModel:           core.GetDefaultModel(),
	}
}

func (s *ConversationContextServiceImpl) AppendMessage(ctx context.Context, jobID string, message string) error {
	log.Printf("📋 Starting to append message to job %s context", jobID)

	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		return fmt.Errorf("organization not found in context")
	}

	// Get existing context
	existingContext, err := s.conversationContextRepo.GetConversationContextByJobID(ctx, org.ID, jobID)
	if err != nil {
		return fmt.Errorf("failed to get existing context: %w", err)
	}

	if existingContext != nil {
		// Append to existing context
		newContext := existingContext.FullContext + "\n\n" + message
		
		// Count tokens using proper token counter
		tokenCount, err := s.tokenCounter.CountTokens(ctx, newContext, s.defaultModel)
		if err != nil {
			log.Printf("Warning: failed to count tokens for context update: %v", err)
			// Use estimation as fallback
			tokenCount = s.tokenCounter.EstimateTokens(newContext)
		}

		_, err = s.conversationContextRepo.UpdateConversationContext(ctx, org.ID, jobID, newContext, tokenCount)
		if err != nil {
			return fmt.Errorf("failed to update conversation context: %w", err)
		}
	} else {
		// Count tokens for new message
		tokenCount, err := s.tokenCounter.CountTokens(ctx, message, s.defaultModel)
		if err != nil {
			log.Printf("Warning: failed to count tokens for new context: %v", err)
			tokenCount = s.tokenCounter.EstimateTokens(message)
		}

		// Create new context
		_, err = s.CreateContextForJob(ctx, jobID, message, tokenCount)
		if err != nil {
			return fmt.Errorf("failed to create new context: %w", err)
		}
	}

	log.Printf("📋 Completed successfully - appended message to context for job: %s", jobID)
	return nil
}

func (s *ConversationContextServiceImpl) GetCurrentContext(ctx context.Context, jobID string) (mo.Option[*models.ConversationContext], error) {
	log.Printf("📋 Starting to get current context for job: %s", jobID)

	if jobID == "" {
		return mo.None[*models.ConversationContext](), fmt.Errorf("job ID cannot be empty")
	}

	org := appctx.GetOrganization(ctx)
	if org == nil {
		return mo.None[*models.ConversationContext](), fmt.Errorf("organization not found in context")
	}

	context, err := s.conversationContextRepo.GetConversationContextByJobID(ctx, org.ID, jobID)
	if err != nil {
		return mo.None[*models.ConversationContext](), fmt.Errorf("failed to get current context: %w", err)
	}

	if context == nil {
		log.Printf("📋 Completed successfully - no context record found for job: %s", jobID)
		return mo.None[*models.ConversationContext](), nil
	}

	log.Printf("📋 Completed successfully - retrieved context for job %s (%d tokens)", jobID, context.ContextSizeTokens)
	return mo.Some(context), nil
}

func (s *ConversationContextServiceImpl) SummarizeIfNeeded(ctx context.Context, jobID string, maxTokens int) error {
	log.Printf("📋 Starting to check if summarization needed for job %s (max tokens: %d)", jobID, maxTokens)

	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		return fmt.Errorf("organization not found in context")
	}

	context, err := s.conversationContextRepo.GetConversationContextByJobID(ctx, org.ID, jobID)
	if err != nil {
		return fmt.Errorf("failed to get context for summarization check: %w", err)
	}

	if context == nil {
		log.Printf("📋 Completed successfully - no context found to summarize for job: %s", jobID)
		return nil
	}

	// Use token counter to check if summarization is needed
	shouldSummarize := s.tokenCounter.ShouldTriggerSummarization(context.ContextSizeTokens, s.defaultModel)
	if !shouldSummarize {
		log.Printf("📋 Completed successfully - no summarization needed for job %s (%d tokens)", 
			jobID, context.ContextSizeTokens)
		return nil
	}

	// TODO: Implement actual Claude API summarization in Phase 5
	// For now, just create a placeholder summary
	summaryPlaceholder := fmt.Sprintf("[SUMMARY PLACEHOLDER] This conversation has %d tokens and needs summarization. Context: %s...", 
		context.ContextSizeTokens, 
		context.FullContext[:min(200, len(context.FullContext))])

	// Keep most recent part of context and add summary
	recentContextStart := len(context.FullContext) / 2 // Keep last 50% of context
	if recentContextStart >= len(context.FullContext) {
		recentContextStart = 0
	}
	
	summarizedContext := summaryPlaceholder + "\n\n[RECENT CONTEXT]\n" + context.FullContext[recentContextStart:]
	
	// Count tokens for the summarized context
	newTokenCount, err := s.tokenCounter.CountTokens(ctx, summarizedContext, s.defaultModel)
	if err != nil {
		log.Printf("Warning: failed to count tokens for summarized context: %v", err)
		newTokenCount = s.tokenCounter.EstimateTokens(summarizedContext)
	}

	_, err = s.conversationContextRepo.UpdateConversationContextWithSummary(ctx, org.ID, jobID, 
		context.FullContext, summaryPlaceholder, newTokenCount)
	if err != nil {
		return fmt.Errorf("failed to update context with summary: %w", err)
	}

	log.Printf("📋 Completed successfully - summarized context for job %s (reduced to %d tokens)", jobID, newTokenCount)
	return nil
}

func (s *ConversationContextServiceImpl) GetLeftoverContext(ctx context.Context, jobID string) (string, error) {
	log.Printf("📋 Starting to get leftover context for job: %s", jobID)

	if jobID == "" {
		return "", fmt.Errorf("job ID cannot be empty")
	}

	org := appctx.GetOrganization(ctx)
	if org == nil {
		return "", fmt.Errorf("organization not found in context")
	}

	context, err := s.conversationContextRepo.GetConversationContextByJobID(ctx, org.ID, jobID)
	if err != nil {
		return "", fmt.Errorf("failed to get leftover context: %w", err)
	}

	if context == nil || !context.HasSummarizedContext() {
		log.Printf("📋 Completed successfully - no leftover context for job: %s", jobID)
		return "", nil
	}

	log.Printf("📋 Completed successfully - retrieved leftover context for job: %s", jobID)
	return *context.SummarizedContext, nil
}

func (s *ConversationContextServiceImpl) UpdateContext(ctx context.Context, jobID string, fullContext string, tokenCount int) error {
	log.Printf("📋 Starting to update context for job %s (%d tokens)", jobID, tokenCount)

	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if tokenCount < 0 {
		return fmt.Errorf("token count cannot be negative")
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		return fmt.Errorf("organization not found in context")
	}

	_, err := s.conversationContextRepo.UpdateConversationContext(ctx, org.ID, jobID, fullContext, tokenCount)
	if err != nil {
		return fmt.Errorf("failed to update context: %w", err)
	}

	log.Printf("📋 Completed successfully - updated context for job: %s", jobID)
	return nil
}

func (s *ConversationContextServiceImpl) CreateContextForJob(ctx context.Context, jobID string, initialMessage string, tokenCount int) (*models.ConversationContext, error) {
	log.Printf("📋 Starting to create context for job %s with initial message (%d tokens)", jobID, tokenCount)

	if jobID == "" {
		return nil, fmt.Errorf("job ID cannot be empty")
	}
	if tokenCount < 0 {
		return nil, fmt.Errorf("token count cannot be negative")
	}

	org := appctx.GetOrganization(ctx)
	if org == nil {
		return nil, fmt.Errorf("organization not found in context")
	}

	context := &models.ConversationContext{
		ID:                core.NewID("ctx"),
		OrganizationID:    org.ID,
		JobID:             jobID,
		FullContext:       initialMessage,
		SummarizedContext: nil,
		ContextSizeTokens: tokenCount,
		IsActive:          true,
	}

	if err := s.conversationContextRepo.CreateConversationContext(ctx, context); err != nil {
		return nil, fmt.Errorf("failed to create context for job: %w", err)
	}

	log.Printf("📋 Completed successfully - created context for job %s with ID: %s", jobID, context.ID)
	return context, nil
}

func (s *ConversationContextServiceImpl) DeleteJobContext(ctx context.Context, jobID string) error {
	log.Printf("📋 Starting to delete context for job: %s", jobID)

	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		return fmt.Errorf("organization not found in context")
	}

	if err := s.conversationContextRepo.DeleteConversationContextByJobID(ctx, org.ID, jobID); err != nil {
		return fmt.Errorf("failed to delete job context: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted context for job: %s", jobID)
	return nil
}

// Helper function for min (Go doesn't have a built-in min for int)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}