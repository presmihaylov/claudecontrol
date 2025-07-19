package services

import (
	"fmt"
	"log"

	"ccbackend/clients"
	"ccbackend/models"

	"github.com/slack-go/slack"
)

type AppService struct {
	slackClient *slack.Client
	appState    *models.AppState
	wsClient    *clients.WebSocketClient
}

func NewAppService(slackClient *slack.Client, appState *models.AppState, wsClient *clients.WebSocketClient) *AppService {
	return &AppService{
		slackClient: slackClient,
		appState:    appState,
		wsClient:    wsClient,
	}
}

func (s *AppService) ProcessAssistantMessage(payload models.AssistantMessagePayload) error {
	log.Printf("🤖 Processing assistant message: %s", payload.Message)

	if s.appState.CurrentSlackThreadTS == "" || s.appState.CurrentSlackChannel == "" {
		log.Printf("⚠️ No current Slack thread/channel to send assistant message to")
		return nil
	}

	log.Printf("📤 Sending assistant message to Slack thread %s in channel %s", s.appState.CurrentSlackThreadTS, s.appState.CurrentSlackChannel)

	_, _, err := s.slackClient.PostMessage(s.appState.CurrentSlackChannel,
		slack.MsgOptionText(payload.Message, false),
		slack.MsgOptionTS(s.appState.CurrentSlackThreadTS),
	)
	if err != nil {
		return fmt.Errorf("❌ Failed to send assistant message to Slack: %v", err)
	}

	log.Printf("✅ Sent assistant message to Slack thread %s", s.appState.CurrentSlackThreadTS)
	return nil
}

func (s *AppService) ProcessSlackMessageEvent(event models.SlackMessageEvent) error {
	log.Printf("📨 Processing Slack message event from %s in %s: %s", event.User, event.Channel, event.Text)

	clientIDs := s.wsClient.GetClientIDs()
	if len(clientIDs) == 0 {
		log.Printf("⚠️ No WebSocket clients connected to handle Slack mention")
		return nil
	}

	firstClientID := clientIDs[0]

	if event.ThreadTs == "" {
		log.Printf("🆕 Bot mentioned at start of new thread in channel %s", event.Channel)

		s.appState.CurrentSlackThreadTS = event.Ts
		s.appState.CurrentSlackChannel = event.Channel
		log.Printf("📌 Updated current Slack thread timestamp to: %s in channel: %s", event.Ts, event.Channel)

		startConversationMessage := models.UnknownMessage{
			Type:    models.MessageTypeStartConversation,
			Payload: models.StartConversationPayload{Message: event.Text},
		}

		if err := s.wsClient.SendMessage(firstClientID, startConversationMessage); err != nil {
			return fmt.Errorf("failed to send start conversation message to client %s: %v", firstClientID, err)
		}
		log.Printf("🚀 Sent start conversation message to client %s", firstClientID)
	} else {
		log.Printf("💬 Bot mentioned in ongoing thread %s in channel %s", event.ThreadTs, event.Channel)

		userMessage := models.UnknownMessage{
			Type:    models.MessageTypeUserMessage,
			Payload: models.UserMessagePayload{Message: event.Text},
		}

		if err := s.wsClient.SendMessage(firstClientID, userMessage); err != nil {
			return fmt.Errorf("failed to send user message to client %s: %v", firstClientID, err)
		}
		log.Printf("💬 Sent user message to client %s", firstClientID)
	}

	return nil
}

