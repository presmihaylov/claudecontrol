package models

import (
	"time"
)

type DiscordIntegration struct {
	ID               string    `db:"id"                 json:"id"`
	DiscordGuildID   string    `db:"discord_guild_id"   json:"discord_guild_id"`
	DiscordGuildName string    `db:"discord_guild_name" json:"discord_guild_name"`
	OrganizationID   OrganizationID `db:"organization_id"    json:"organization_id"`
	CreatedAt        time.Time `db:"created_at"         json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"         json:"updated_at"`
}
