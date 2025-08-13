package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ConversationCost tracks token usage and estimated costs for a conversation
type ConversationCost struct {
	ID                string          `db:"id"                   json:"id"`
	OrganizationID    OrgID           `db:"organization_id"      json:"organization_id"`
	JobID             string          `db:"job_id"               json:"job_id"`
	TotalInputTokens  int             `db:"total_input_tokens"   json:"total_input_tokens"`
	TotalOutputTokens int             `db:"total_output_tokens"  json:"total_output_tokens"`
	EstimatedCostUSD  decimal.Decimal `db:"estimated_cost_usd"   json:"estimated_cost_usd"`
	CreatedAt         time.Time       `db:"created_at"           json:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at"           json:"updated_at"`
}

// TotalTokens returns the sum of input and output tokens
func (cc *ConversationCost) TotalTokens() int {
	return cc.TotalInputTokens + cc.TotalOutputTokens
}