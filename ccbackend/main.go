package main

import (
	"encoding/json"
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

	wsServer := NewWebsocketServer()
	wsServer.StartWebsocketServer()

	wsServer.registerMessageHandler(func(client *Client, msg any) {
		handleWSMessage(client, msg, wsServer)
	})

	port := "8080"
	log.Printf("✅ Listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWSMessage(client *Client, msg any, wsServer *WebSocketServer) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("❌ Failed to marshal message from client %s: %v", client.ID, err)
		return
	}

	var parsedMsg UnknownMessage
	if err := json.Unmarshal(msgBytes, &parsedMsg); err != nil {
		log.Printf("❌ Failed to parse message from client %s: %v", client.ID, err)
		return
	}

	switch parsedMsg.Type {
	case "ping":
		var payload PingPayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal ping payload from client %s: %v", client.ID, err)
			return
		}

		log.Printf("📨 Received ping message from client %s", client.ID)
		response := UnknownMessage{
			Type:    "pong",
			Payload: PongPayload{},
		}

		if err := wsServer.SendMessage(client.ID, response); err != nil {
			log.Printf("❌ Failed to send pong to client %s: %v", client.ID, err)
		} else {
			log.Printf("🏓 Sent pong response to client %s", client.ID)
		}

	case "pong":
		var payload PongPayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal pong payload from client %s: %v", client.ID, err)
			return
		}

		log.Printf("🏓 Received pong from client %s", client.ID)
	default:
		log.Printf("⚠️ Unknown message type '%s' from client %s", parsedMsg.Type, client.ID)
	}
}

func unmarshalPayload(payload any, target any) error {
	if payload == nil {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return json.Unmarshal(payloadBytes, target)
}
