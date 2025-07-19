package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

func setupSlackEventsEndpoints(slackClient *slack.Client, wsServer *WebSocketServer) {
	log.Printf("🚀 Registering Slack events endpoint on /slack/events")
	http.HandleFunc("/slack/events", func(w http.ResponseWriter, r *http.Request) {
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

		// Send pong message to all WebSocket clients
		clientIDs := wsServer.GetClientIDs()
		log.Printf("🔔 Sending pong to %d WebSocket clients due to Slack mention", len(clientIDs))

		pongMessage := UnknownMessage{
			Type:    "pong",
			Payload: PongPayload{},
		}

		for _, clientID := range clientIDs {
			if err := wsServer.SendMessage(clientID, pongMessage); err != nil {
				log.Printf("❌ Failed to send pong to WebSocket client %s: %v", clientID, err)
			} else {
				log.Printf("🏓 Sent pong to WebSocket client %s", clientID)
			}
		}

		_, _, err := slackClient.PostMessage(channel,
			slack.MsgOptionText("👋 Got it! Thanks for the mention. ", false),
			slack.MsgOptionTS(timestamp),
			slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{
				AsUser: true,
			}),
		)
		if err != nil {
			log.Printf("❌ Failed to reply to mention: %v", err)
		} else {
			log.Printf("✅ Replied to Slack mention in channel %s", channel)
		}

		w.WriteHeader(http.StatusOK)
	})
	log.Printf("✅ Slack events endpoint registered successfully")
}
