package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
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

	setupSlackCommandsEndpoints(slackClient, signingSecret)
	setupSlackEventsEndpoints(slackClient)
	setupWebSocketEndpoint()

	port := "3000"
	log.Printf("✅ Listening on http://localhost:%s/slack/commands", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
