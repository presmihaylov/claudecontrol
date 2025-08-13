package models

import (
	"time"
)

// ConversationContext manages the full conversation context and summarization for a job
type ConversationContext struct {
	ID                 string    `db:"id"                    json:"id"`
	OrganizationID     OrgID     `db:"organization_id"       json:"organization_id"`
	JobID              string    `db:"job_id"                json:"job_id"`
	FullContext        string    `db:"full_context"          json:"full_context"`
	SummarizedContext  *string   `db:"summarized_context"    json:"summarized_context"`
	ContextSizeTokens  int       `db:"context_size_tokens"   json:"context_size_tokens"`
	IsActive           bool      `db:"is_active"             json:"is_active"`
	CreatedAt          time.Time `db:"created_at"            json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"            json:"updated_at"`
}

// HasSummarizedContext returns true if this context has been summarized
func (cc *ConversationContext) HasSummarizedContext() bool {
	return cc.SummarizedContext != nil && *cc.SummarizedContext != ""
}