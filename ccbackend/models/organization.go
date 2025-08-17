package models

import (
	"time"
)

type Organization struct {
	ID                          string     `db:"id"                              json:"id"`
	CCAgentSecretKey            *string    `db:"ccagent_secret_key"              json:"-"`
	CCAgentSecretKeyGeneratedAt *time.Time `db:"ccagent_secret_key_generated_at" json:"ccagent_secret_key_generated_at"`
	CCAgentSystemSecretKey      string     `db:"cc_agent_system_secret_key"      json:"-"`
	CreatedAt                   time.Time  `db:"created_at"                      json:"created_at"`
	UpdatedAt                   time.Time  `db:"updated_at"                      json:"updated_at"`
}
