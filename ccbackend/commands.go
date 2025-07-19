package main

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

func setupSlackCommandsEndpoints(slackClient *slack.Client, signingSecret string) {
	log.Printf("🚀 Registering Slack commands endpoint on /slack/commands")
	http.HandleFunc("/slack/commands", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("⚡ Slack command received from %s", r.RemoteAddr)
		var buf bytes.Buffer
		tee := io.TeeReader(r.Body, &buf)

		verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
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
				_, _, err := slackClient.PostMessage(command.ChannelID,
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
	})
	log.Printf("✅ Slack commands endpoint registered successfully")
}
