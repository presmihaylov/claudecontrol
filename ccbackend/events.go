package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

func setupSlackEventsEndpoints(slackClient *slack.Client) {
	http.HandleFunc("/slack/events", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "failed to parse body", http.StatusBadRequest)
			return
		}

		if body["type"] == "url_verification" {
			challenge, ok := body["challenge"].(string)
			if !ok {
				http.Error(w, "challenge not found", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(challenge))
			return
		}

		if body["type"] != "event_callback" {
			w.WriteHeader(http.StatusOK)
			return
		}

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

		fmt.Printf("📨 Mentioned by %s in %s: %s\n", user, channel, text)

		_, _, err := slackClient.PostMessage(channel,
			slack.MsgOptionText("👋 Got it! Thanks for the mention. ", false),
			slack.MsgOptionTS(timestamp),
			slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{
				AsUser: true,
			}),
		)
		if err != nil {
			log.Printf("❌ Failed to reply to mention: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	})
}

