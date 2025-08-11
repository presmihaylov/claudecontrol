package api

import (
	"time"

	"ccbackend/models"
)

// DiscordIntegrationModel represents the discord integration data returned by the API
type DiscordIntegrationModel struct {
	ID               string       `json:"id"`
	DiscordGuildID   string       `json:"discord_guild_id"`
	DiscordGuildName string       `json:"discord_guild_name"`
	OrgID            models.OrgID `json:"organization_id"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}
