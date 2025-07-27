package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases"

	"github.com/gorilla/mux"
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


func (h *SlackWebhooksHandler) HandleSlackEvent(w http.ResponseWriter, r *http.Request) {
	log.Printf("📨 Slack event received from %s", r.RemoteAddr)
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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
		w.Write([]byte(challenge))
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
	if eventType != "app_mention" {
		log.Printf("❌ Unsupported event type: %s", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}

	channel := event["channel"].(string)
	user := event["user"].(string)
	text := event["text"].(string)
	timestamp := event["ts"].(string)

	log.Printf("📨 Bot mentioned by %s in %s: %s", user, channel, text)

	threadTs, hasThreadTs := event["thread_ts"].(string)
	if !hasThreadTs {
		threadTs = ""
	}

	slackEvent := models.SlackMessageEvent{
		Channel:  channel,
		User:     user,
		Text:     text,
		Ts:       timestamp,
		ThreadTs: threadTs,
	}

	if err := h.coreUseCase.ProcessSlackMessageEvent(slackEvent, slackIntegration.ID.String()); err != nil {
		log.Printf("❌ Failed to process Slack message event: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SlackWebhooksHandler) SetupEndpoints(router *mux.Router) {
	log.Printf("🚀 Registering Slack webhook endpoints")
	
	router.HandleFunc("/slack/events", h.HandleSlackEvent).Methods("POST")
	log.Printf("✅ POST /slack/events endpoint registered")
	
	log.Printf("✅ All Slack webhook endpoints registered successfully")
}

