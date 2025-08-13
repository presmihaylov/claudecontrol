package models

import (
	"github.com/shopspring/decimal"
)

// Message types
const (
	MessageTypeStartConversation = "start_conversation_v1"
	MessageTypeUserMessage       = "user_message_v1"
	MessageTypeAssistantMessage  = "assistant_message_v1"
	MessageTypeSystemMessage     = "system_message_v1"
	MessageTypeProcessingMessage = "processing_message_v1"
	MessageTypeCheckIdleJobs     = "check_idle_jobs_v1"
	MessageTypeJobComplete       = "job_complete_v1"
	// New message types for PMI-981
	MessageTypeContextSummary    = "context_summary_v1"
	MessageTypeCostAlert         = "cost_alert_v1"
	MessageTypeContextTruncated  = "context_truncated_v1"
)

type BaseMessage struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type StartConversationPayload struct {
	JobID              string `json:"job_id"`
	Message            string `json:"message"`
	ProcessedMessageID string `json:"processed_message_id"`
	MessageLink        string `json:"message_link"`
}

type UserMessagePayload struct {
	JobID              string `json:"job_id"`
	Message            string `json:"message"`
	ProcessedMessageID string `json:"processed_message_id"`
	MessageLink        string `json:"message_link"`
}

type AssistantMessagePayload struct {
	JobID              string `json:"job_id"`
	Message            string `json:"message"`
	ProcessedMessageID string `json:"processed_message_id"`
}

type SystemMessagePayload struct {
	Message            string                    `json:"message"`
	ProcessedMessageID string                    `json:"processed_message_id"`
	JobID              string                    `json:"job_id"`
	CostInfo           *ConversationCostInfo     `json:"cost_info,omitempty"`
	ContextInfo        *ConversationContextInfo  `json:"context_info,omitempty"`
}

// ConversationCostInfo provides cost and token usage information for system messages
type ConversationCostInfo struct {
	TotalTokens       int             `json:"total_tokens"`
	EstimatedCostUSD  decimal.Decimal `json:"estimated_cost_usd"`
	TokensThisMessage int             `json:"tokens_this_message"`
}

// ConversationContextInfo provides context management information for system messages
type ConversationContextInfo struct {
	CurrentTokens   int    `json:"current_tokens"`
	MaxTokens       int    `json:"max_tokens"`
	LeftoverContext string `json:"leftover_context,omitempty"`
	WasSummarized   bool   `json:"was_summarized"`
}

type ProcessingMessagePayload struct {
	ProcessedMessageID string `json:"processed_message_id"`
	JobID              string `json:"job_id"`
}

type CheckIdleJobsPayload struct {
	// Empty payload - agent checks all its jobs
}

type JobCompletePayload struct {
	JobID  string `json:"job_id"`
	Reason string `json:"reason"`
}
