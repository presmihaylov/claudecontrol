package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"ccbackend/models"
	"ccbackend/services"

	"github.com/slack-go/slack"
)

type SlackWebhooksHandler struct {
	slackClient   *slack.Client
	signingSecret string
	appService    *services.AppService
}

func NewSlackWebhooksHandler(slackClient *slack.Client, signingSecret string, appService *services.AppService) *SlackWebhooksHandler {
	return &SlackWebhooksHandler{
		slackClient:   slackClient,
		signingSecret: signingSecret,
		appService:    appService,
	}
}

func (h *SlackWebhooksHandler) HandleSlackCommand(w http.ResponseWriter, r *http.Request) {
	log.Printf("⚡ Slack command received from %s", r.RemoteAddr)
	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)

	verifier, err := slack.NewSecretsVerifier(r.Header, h.signingSecret)
	if err != nil {
		log.Printf("❌ Invalid secret verifier: %v", err)
		http.Error(w, "invalid secret verifier", http.StatusUnauthorized)
		return
	}

	if _, err := io.Copy(&verifier, tee); err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}

	if err := verifier.Ensure(); err != nil {
		log.Printf("❌ Slack signature verification failed: %v", err)
		http.Error(w, "signature verification failed", http.StatusUnauthorized)
		return
	}

	log.Printf("✅ Slack signature verification successful")

	r.Body = io.NopCloser(&buf)

	command, err := slack.SlashCommandParse(r)
	if err != nil {
		log.Printf("❌ Failed to parse slash command: %v", err)
		http.Error(w, "failed to parse slash command", http.StatusInternalServerError)
		return
	}

	log.Printf("⚡ Parsed slash command: %s from user %s in channel %s", command.Command, command.UserID, command.ChannelID)

	if command.Command == "/cc" {
		log.Printf("🎯 Processing /cc command with text: %s", command.Text)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		go func() {
			_, _, err := h.slackClient.PostMessage(command.ChannelID,
				slack.MsgOptionText("echo "+command.Text, false),
				slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{
					AsUser: true,
				}),
			)
			if err != nil {
				log.Printf("❌ Failed to post message: %v", err)
			} else {
				log.Printf("✅ /cc command response posted successfully to channel %s", command.ChannelID)
			}
		}()

		return
	}

	log.Printf("⚠️ Unknown slash command: %s", command.Command)
	w.WriteHeader(http.StatusOK)
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

	if err := h.appService.ProcessSlackMessageEvent(slackEvent); err != nil {
		log.Printf("❌ Failed to process Slack message event: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SlackWebhooksHandler) SetupEndpoints() {
	log.Printf("🚀 Registering Slack commands endpoint on /slack/commands")
	http.HandleFunc("/slack/commands", h.HandleSlackCommand)
	log.Printf("✅ Slack commands endpoint registered successfully")

	log.Printf("🚀 Registering Slack events endpoint on /slack/events")
	http.HandleFunc("/slack/events", h.HandleSlackEvent)
	log.Printf("✅ Slack events endpoint registered successfully")
}

