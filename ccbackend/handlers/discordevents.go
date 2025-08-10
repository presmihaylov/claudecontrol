package handlers

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"ccbackend/services"
	"ccbackend/usecases/core"
)

type DiscordEventsHandler struct {
	publicKey                   string
	coreUseCase                 *core.CoreUseCase
	discordIntegrationsService  services.DiscordIntegrationsService
}

func NewDiscordEventsHandler(
	publicKey string,
	coreUseCase *core.CoreUseCase,
	discordIntegrationsService services.DiscordIntegrationsService,
) *DiscordEventsHandler {
	return &DiscordEventsHandler{
		publicKey:                  publicKey,
		coreUseCase:               coreUseCase,
		discordIntegrationsService: discordIntegrationsService,
	}
}

// verifyDiscordSignature verifies the authenticity of a Discord webhook request
func (h *DiscordEventsHandler) verifyDiscordSignature(r *http.Request, body []byte) error {
	signature := r.Header.Get("X-Signature-Ed25519")
	timestamp := r.Header.Get("X-Signature-Timestamp")

	if signature == "" || timestamp == "" {
		return fmt.Errorf("missing required Discord signature headers")
	}

	// Decode public key
	publicKey, err := hex.DecodeString(h.publicKey)
	if err != nil {
		return fmt.Errorf("invalid public key format: %v", err)
	}

	// Decode signature
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature format: %v", err)
	}

	// Verify signature using ed25519
	message := append([]byte(timestamp), body...)
	if !ed25519.Verify(ed25519.PublicKey(publicKey), message, sig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

func (h *DiscordEventsHandler) HandleDiscordInteraction(w http.ResponseWriter, r *http.Request) {
	log.Printf("📨 Discord interaction received from %s", r.RemoteAddr)

	// Read raw body for signature verification
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Verify Discord signature
	if err := h.verifyDiscordSignature(r, bodyBytes); err != nil {
		log.Printf("❌ Discord signature verification failed: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	log.Printf("✅ Discord signature verified successfully")

	// Parse JSON from body bytes
	var interaction map[string]any
	if err := json.Unmarshal(bodyBytes, &interaction); err != nil {
		log.Printf("❌ Failed to parse JSON body: %v", err)
		http.Error(w, "failed to parse body", http.StatusBadRequest)
		return
	}

	// Handle ping interactions (Discord verification)
	interactionType, ok := interaction["type"].(float64)
	if !ok {
		log.Printf("❌ Invalid interaction type")
		http.Error(w, "invalid interaction type", http.StatusBadRequest)
		return
	}

	if int(interactionType) == 1 { // PING
		log.Printf("🏓 Discord ping received, responding with pong")
		w.Header().Set("Content-Type", "application/json")
		response := map[string]int{"type": 1} // PONG
		json.NewEncoder(w).Encode(response)
		return
	}

	// Extract guild_id for integration lookup
	var guildID string
	if guild, ok := interaction["guild_id"].(string); ok {
		guildID = guild
	} else if guildObj, ok := interaction["guild"].(map[string]any); ok {
		if id, ok := guildObj["id"].(string); ok {
			guildID = id
		}
	}

	if guildID == "" {
		log.Printf("❌ Guild ID not found in Discord interaction")
		http.Error(w, "guild_id not found", http.StatusBadRequest)
		return
	}

	log.Printf("📨 Discord interaction details - Guild: %s, Type: %v", guildID, interactionType)

	// Lookup discord integration by guild_id
	maybeDiscordInt, err := h.discordIntegrationsService.GetDiscordIntegrationByGuildID(r.Context(), guildID)
	if err != nil {
		log.Printf("❌ Failed to find discord integration for guild %s: %v", guildID, err)
		http.Error(w, "integration lookup failed", http.StatusInternalServerError)
		return
	}
	if !maybeDiscordInt.IsPresent() {
		log.Printf("❌ Discord integration not found for guild %s", guildID)
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}
	discordIntegration := maybeDiscordInt.MustGet()

	log.Printf("🔑 Found discord integration for guild %s (ID: %s)", guildID, discordIntegration.ID)

	// For now, we only handle ping interactions
	// Additional interaction types can be added here (slash commands, message components, etc.)
	
	w.Header().Set("Content-Type", "application/json")
	response := map[string]int{"type": 4} // CHANNEL_MESSAGE_WITH_SOURCE - placeholder response
	json.NewEncoder(w).Encode(response)
}

func (h *DiscordEventsHandler) SetupEndpoints(router *mux.Router) {
	log.Printf("🚀 Registering Discord webhook endpoints")

	router.HandleFunc("/discord/interactions", h.HandleDiscordInteraction).Methods("POST")
	log.Printf("✅ POST /discord/interactions endpoint registered")

	log.Printf("✅ All Discord webhook endpoints registered successfully")
}