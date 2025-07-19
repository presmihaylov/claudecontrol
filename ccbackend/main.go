package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"io"
	"log"
	"net/http"
	"os"
)

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️ Could not load .env file, continuing with system env vars")
	}
}

func main() {
	token := os.Getenv("SLACK_BOT_TOKEN")
	if token == "" {
		panic("SLACK_BOT_TOKEN is not set")
	}

	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	if signingSecret == "" {
		panic("SLACK_SIGNING_SECRET is not set")
	}

	slackClient := slack.New(token)

	http.HandleFunc("/slack/commands", func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		tee := io.TeeReader(r.Body, &buf)

		verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			http.Error(w, "invalid secret verifier", http.StatusUnauthorized)
			return
		}

		if _, err := io.Copy(&verifier, tee); err != nil {
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return
		}

		if err := verifier.Ensure(); err != nil {
			http.Error(w, "signature verification failed", http.StatusUnauthorized)
			return
		}

		r.Body = io.NopCloser(&buf)

		command, err := slack.SlashCommandParse(r)
		if err != nil {
			http.Error(w, "failed to parse slash command", http.StatusInternalServerError)
			return
		}

		if command.Command == "/cc" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			go func() {
				_, _, err := slackClient.PostMessage(command.ChannelID,
					slack.MsgOptionText("echo "+command.Text, false),
					slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{
						AsUser: true,
					}),
				)
				if err != nil {
					log.Printf("❌ Failed to post message: %v", err)
				} else {
					fmt.Println("✅ Message posted successfully!")
				}
			}()

			return
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/slack/events", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
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

		if body["type"] == "event_callback" {
			event := body["event"].(map[string]interface{})
			eventType := event["type"].(string)

			if eventType == "app_mention" {
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
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	port := "3000"
	log.Printf("✅ Listening on http://localhost:%s/slack/commands", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
