package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

func setupSlackCommandsEndpoints(slackClient *slack.Client, signingSecret string) {
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
}

