package handlers

import (
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

	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases"
)

type SlackWebhooksHandler struct {
	signingSecret            string
	coreUseCase              *usecases.CoreUseCase
	slackIntegrationsService *services.SlackIntegrationsService
}

func NewSlackWebhooksHandler(signingSecret string, coreUseCase *usecases.CoreUseCase, slackIntegrationsService *services.SlackIntegrationsService) *SlackWebhooksHandler {
	return &SlackWebhooksHandler{
		signingSecret:            signingSecret,
		coreUseCase:              coreUseCase,
		slackIntegrationsService: slackIntegrationsService,
	}
}

// verifySlackSignature verifies the authenticity of a Slack webhook request
func (h *SlackWebhooksHandler) verifySlackSignature(r *http.Request, body []byte) error {
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

func (h *SlackWebhooksHandler) HandleSlackEvent(w http.ResponseWriter, r *http.Request) {
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

	// Lookup slack integration by team_id
	slackIntegration, err := h.slackIntegrationsService.GetSlackIntegrationByTeamID(teamID)
	if err != nil {
		log.Printf("❌ Failed to find slack integration for team %s: %v", teamID, err)
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	log.Printf("🔑 Found slack integration for team %s (ID: %s)", teamID, slackIntegration.ID)

	event := body["event"].(map[string]any)
	eventType := event["type"].(string)

	switch eventType {
	case "app_mention":
		if err := h.handleAppMention(event, slackIntegration.ID.String()); err != nil {
			log.Printf("❌ Failed to handle app mention: %v", err)
		}
	case "reaction_added":
		if err := h.handleReactionAdded(event, slackIntegration.ID.String()); err != nil {
			log.Printf("❌ Failed to handle reaction added: %v", err)
		}
	default:
		log.Printf("❌ Unsupported event type: %s", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SlackWebhooksHandler) SetupEndpoints(router *mux.Router) {
	log.Printf("🚀 Registering Slack webhook endpoints")

	router.HandleFunc("/slack/events", h.HandleSlackEvent).Methods("POST")
	log.Printf("✅ POST /slack/events endpoint registered")

	log.Printf("✅ All Slack webhook endpoints registered successfully")
}

func (h *SlackWebhooksHandler) handleAppMention(event map[string]any, slackIntegrationID string) error {
	channel := event["channel"].(string)
	user := event["user"].(string)
	text := event["text"].(string)
	timestamp := event["ts"].(string)

	log.Printf("📨 Bot mentioned by %s in %s: %s", user, channel, text)

	threadTS, hasThreadTS := event["thread_ts"].(string)
	if !hasThreadTS {
		threadTS = ""
	}

	slackEvent := models.SlackMessageEvent{
		Channel:  channel,
		User:     user,
		Text:     text,
		TS:       timestamp,
		ThreadTS: threadTS,
	}

	return h.coreUseCase.ProcessSlackMessageEvent(slackEvent, slackIntegrationID)
}

func (h *SlackWebhooksHandler) handleReactionAdded(event map[string]any, slackIntegrationID string) error {
	reactionName := event["reaction"].(string)
	user := event["user"].(string)
	item := event["item"].(map[string]any)

	// Only handle white check mark, check mark, or white tick reactions
	if reactionName != "white_check_mark" && reactionName != "heavy_check_mark" && reactionName != "white_tick" {
		log.Printf("⏭️ Ignoring reaction: %s (not a completion emoji)", reactionName)
		return nil
	}

	// Extract item details
	itemType := item["type"].(string)
	if itemType != "message" {
		log.Printf("⏭️ Ignoring reaction on non-message item: %s", itemType)
		return nil
	}

	channel := item["channel"].(string)
	ts := item["ts"].(string)

	log.Printf("✅ Completion reaction added by %s on message %s in %s", user, ts, channel)

	return h.coreUseCase.ProcessReactionAdded(user, channel, ts, slackIntegrationID)
}
