package discord

import (
	"ccbackend/clients"
)

// DiscordUseCase handles all Discord-specific operations
type DiscordUseCase struct {
	discordClient clients.DiscordClient
}

// NewDiscordUseCase creates a new instance of DiscordUseCase
func NewDiscordUseCase(
	discordClient clients.DiscordClient,
) *DiscordUseCase {
	return &DiscordUseCase{
		discordClient: discordClient,
	}
}