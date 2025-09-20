package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"ccbackend/appctx"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases/core"
	"ccbackend/utils"
)

type SlackEventsHandler struct {
	signingSecret            string
	coreUseCase              *core.CoreUseCase
	slackIntegrationsService services.SlackIntegrationsService
	connectedChannelsService services.ConnectedChannelsService
	commandsService          services.CommandsService
}

func NewSlackEventsHandler(
	signingSecret string,
	coreUseCase *core.CoreUseCase,
	slackIntegrationsService services.SlackIntegrationsService,
	connectedChannelsService services.ConnectedChannelsService,
	commandsService services.CommandsService,
) *SlackEventsHandler {
	return &SlackEventsHandler{
		signingSecret:            signingSecret,
		coreUseCase:              coreUseCase,
		slackIntegrationsService: slackIntegrationsService,
		connectedChannelsService: connectedChannelsService,
		commandsService:          commandsService,
	}
}

// verifySlackSignature verifies the authenticity of a Slack webhook request
func (h *SlackEventsHandler) verifySlackSignature(r *http.Request, body []byte) error {
	// Extract headers
	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")

	if timestamp == "" || signature == "" {
		return fmt.Errorf("missing required headers")
	}

	// Verify timestamp freshness (within 5 minutes)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %v", err)
	}

	if time.Now().Unix()-ts > 300 { // 5 minutes
		return fmt.Errorf("request timestamp too old")
	}

	// Create signature base string: v0:timestamp:body
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(h.signingSecret))
	mac.Write([]byte(baseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Secure comparison
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

func (h *SlackEventsHandler) HandleSlackEvent(w http.ResponseWriter, r *http.Request) {
	log.Printf("📨 Slack event received from %s", r.RemoteAddr)

	// Read raw body for signature verification
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Verify Slack signature
	if err := h.verifySlackSignature(r, bodyBytes); err != nil {
		log.Printf("❌ Slack signature verification failed: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	log.Printf("✅ Slack signature verified successfully")

	// Parse JSON from body bytes
	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		log.Printf("❌ Failed to parse JSON body: %v", err)
		http.Error(w, "failed to parse body", http.StatusBadRequest)
		return
	}

	if body["type"] == "url_verification" {
		log.Printf("🔐 Slack URL verification challenge received")
		challenge, ok := body["challenge"].(string)
		if !ok {
			log.Printf("❌ Challenge not found in verification request")
			http.Error(w, "challenge not found", http.StatusBadRequest)
			return
		}
		log.Printf("✅ Responding to Slack URL verification challenge")
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(challenge)); err != nil {
			log.Printf("❌ Failed to write challenge response: %v", err)
		}
		return
	}

	if body["type"] != "event_callback" {
		log.Printf("📋 Non-event callback received: %s", body["type"])
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("📞 Event callback received from Slack")

	// Extract team_id from the event
	teamID, ok := body["team_id"].(string)
	if !ok || teamID == "" {
		log.Printf("❌ Team ID not found in Slack event")
		http.Error(w, "team_id not found", http.StatusBadRequest)
		return
	}

	// Extract channel information from the event for logging
	event := body["event"].(map[string]any)
	channelID := ""
	if channel, ok := event["channel"].(string); ok {
		channelID = channel
	}

	log.Printf("📨 Slack event details - Team: %s, Channel: %s", teamID, channelID)

	// Lookup slack integration by team_id
	maybeSlackInt, err := h.slackIntegrationsService.GetSlackIntegrationByTeamID(r.Context(), teamID)
	if err != nil {
		log.Printf("❌ Failed to find slack integration for team %s: %v", teamID, err)
		http.Error(w, "integration lookup failed", http.StatusInternalServerError)
		return
	}
	if !maybeSlackInt.IsPresent() {
		log.Printf("❌ Slack integration not found for team %s", teamID)
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}
	slackIntegration := maybeSlackInt.MustGet()

	log.Printf("🔑 Found slack integration for team %s (ID: %s)", teamID, slackIntegration.ID)

	eventType := event["type"].(string)

	switch eventType {
	case "app_mention":
		if err := h.handleAppMention(r.Context(), event, slackIntegration.ID, slackIntegration.OrgID, slackIntegration.SlackTeamID); err != nil {
			log.Printf("❌ Failed to handle app mention: %v", err)
		}
	case "reaction_added":
		if err := h.handleReactionAdded(r.Context(), event, slackIntegration.ID, slackIntegration.OrgID, slackIntegration.SlackTeamID); err != nil {
			log.Printf("❌ Failed to handle reaction added: %v", err)
		}
	default:
		log.Printf("❌ Unsupported event type: %s", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SlackEventsHandler) SetupEndpoints(router *mux.Router) {
	log.Printf("🚀 Registering Slack webhook endpoints")

	router.HandleFunc("/slack/events", h.HandleSlackEvent).Methods("POST")
	log.Printf("✅ POST /slack/events endpoint registered")

	log.Printf("✅ All Slack webhook endpoints registered successfully")
}

func (h *SlackEventsHandler) handleAppMention(
	ctx context.Context,
	event map[string]any,
	slackIntegrationID string,
	orgID models.OrgID,
	teamID string,
) error {
	channel := event["channel"].(string)
	user := event["user"].(string)
	text := event["text"].(string)
	timestamp := event["ts"].(string)

	log.Printf("📨 Bot mentioned by %s in %s: %s", user, channel, text)

	threadTS, hasThreadTS := event["thread_ts"].(string)
	if !hasThreadTS {
		threadTS = ""
	}

	// Track the channel in connected_channels table
	_, err := h.connectedChannelsService.UpsertSlackConnectedChannel(ctx, orgID, teamID, channel)
	if err != nil {
		log.Printf("❌ Failed to track Slack channel %s: %v", channel, err)
		return fmt.Errorf("failed to track Slack channel: %w", err)
	}

	// Check if this is a command
	commandResult := utils.DetectCommand(text)
	if commandResult.IsCommand {
		log.Printf("🎯 Command detected in Slack message: %s", commandResult.CommandText)
		return h.handleSlackCommand(ctx, commandResult.CommandText, slackIntegrationID, orgID, teamID, channel, user, timestamp, threadTS)
	}

	// Not a command - proceed with normal message processing
	slackEvent := models.SlackMessageEvent{
		Channel:  channel,
		User:     user,
		Text:     text,
		TS:       timestamp,
		ThreadTS: threadTS,
	}

	return h.coreUseCase.ProcessSlackMessageEvent(ctx, slackEvent, slackIntegrationID, orgID)
}

func (h *SlackEventsHandler) handleReactionAdded(
	ctx context.Context,
	event map[string]any,
	slackIntegrationID string,
	orgID models.OrgID,
	teamID string,
) error {
	reactionName := event["reaction"].(string)
	user := event["user"].(string)
	item := event["item"].(map[string]any)

	// Extract item details
	itemType := item["type"].(string)
	if itemType != "message" {
		log.Printf("⏭️ Ignoring reaction on non-message item: %s", itemType)
		return nil
	}

	channel := item["channel"].(string)
	ts := item["ts"].(string)

	log.Printf("📨 Reaction %s added by %s on message %s in %s", reactionName, user, ts, channel)

	// Track the channel in connected_channels table
	_, err := h.connectedChannelsService.UpsertSlackConnectedChannel(ctx, orgID, teamID, channel)
	if err != nil {
		log.Printf("❌ Failed to track Slack channel %s: %v", channel, err)
		return fmt.Errorf("failed to track Slack channel: %w", err)
	}

	return h.coreUseCase.ProcessReactionAdded(ctx, reactionName, user, channel, ts, slackIntegrationID, orgID)
}

func (h *SlackEventsHandler) handleSlackCommand(
	ctx context.Context,
	commandText string,
	slackIntegrationID string,
	orgID models.OrgID,
	teamID string,
	channelID string,
	userID string,
	messageTS string,
	threadTS string,
) error {
	log.Printf("📋 Starting to handle Slack command: %s in channel: %s", commandText, channelID)

	// Get the connected channel for this Slack channel
	connectedChannelOpt, err := h.connectedChannelsService.GetSlackConnectedChannel(ctx, orgID, teamID, channelID)
	if err != nil {
		log.Printf("❌ Failed to get connected channel: %v", err)
		return fmt.Errorf("failed to get connected channel: %w", err)
	}
	if !connectedChannelOpt.IsPresent() {
		log.Printf("❌ Connected channel not found for team: %s, channel: %s", teamID, channelID)
		return fmt.Errorf("connected channel not found")
	}

	connectedChannel := connectedChannelOpt.MustGet()

	// Add organization to context for the commands service
	org, err := h.getOrganizationByID(ctx, orgID)
	if err != nil {
		log.Printf("❌ Failed to get organization: %v", err)
		return fmt.Errorf("failed to get organization: %w", err)
	}

	ctx = appctx.SetOrganization(ctx, org)

	// Create simplified command request
	commandRequest := models.CommandRequest{
		Command:     commandText,
		UserID:      userID,
		MessageText: commandText,
	}

	// Process the command
	result, err := h.commandsService.ProcessCommand(ctx, commandRequest, connectedChannel)
	if err != nil {
		log.Printf("❌ Failed to process command: %v", err)
		// Send error message back to Slack
		return h.sendSlackResponse(ctx, slackIntegrationID, channelID, threadTS, "❌ Error processing command: "+err.Error())
	}

	// Send result message back to Slack
	return h.sendSlackResponse(ctx, slackIntegrationID, channelID, threadTS, result.Message)
}

func (h *SlackEventsHandler) getOrganizationByID(ctx context.Context, orgID models.OrgID) (*models.Organization, error) {
	// For now, create a mock organization - this should be replaced with actual service call
	// TODO: Use organization service to get the actual organization
	return &models.Organization{
		ID: string(orgID),
	}, nil
}

func (h *SlackEventsHandler) sendSlackResponse(
	ctx context.Context,
	slackIntegrationID string,
	channelID string,
	threadTS string,
	message string,
) error {
	log.Printf("📋 Starting to send Slack response to channel: %s, message: %s", channelID, message)

	// TODO: Implement actual Slack message sending
	// For now, just log the response - this needs to be implemented with a Slack client
	log.Printf("🎯 Would send to Slack channel %s: %s", channelID, message)

	// This is a temporary implementation - we need to add proper Slack client integration
	log.Printf("📋 Completed successfully - logged Slack response (actual sending not yet implemented)")
	return nil
}
