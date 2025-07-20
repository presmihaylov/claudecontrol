package services

import (
	"fmt"
	"log"

	"ccbackend/clients"
	"ccbackend/models"

	"github.com/slack-go/slack"
)

type AppService struct {
	slackClient   *slack.Client
	appState      *models.AppState
	wsClient      *clients.WebSocketClient
	agentsService *AgentsService
}

func NewAppService(slackClient *slack.Client, appState *models.AppState, wsClient *clients.WebSocketClient, agentsService *AgentsService) *AppService {
	return &AppService{
		slackClient:   slackClient,
		appState:      appState,
		wsClient:      wsClient,
		agentsService: agentsService,
	}
}

func (s *AppService) ProcessAssistantMessage(payload models.AssistantMessagePayload) error {
	log.Printf("📋 Starting to process assistant message: %s", payload.Message)

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

	log.Printf("📋 Completed successfully - sent assistant message to Slack thread %s", s.appState.CurrentSlackThreadTS)
	return nil
}

func (s *AppService) ProcessSlackMessageEvent(event models.SlackMessageEvent) error {
	log.Printf("📋 Starting to process Slack message event from %s in %s: %s", event.User, event.Channel, event.Text)

	availableAgents, err := s.agentsService.GetAvailableAgents()
	if err != nil {
		log.Printf("❌ Failed to get available agents: %v", err)
		return fmt.Errorf("failed to get available agents: %w", err)
	}

	if len(availableAgents) == 0 {
		log.Printf("⚠️ No available agents to handle Slack mention")
		return nil
	}

	firstAgent := availableAgents[0]
	clientID := firstAgent.WSConnectionID

	if event.ThreadTs == "" {
		log.Printf("🆕 Bot mentioned at start of new thread in channel %s", event.Channel)

		s.appState.CurrentSlackThreadTS = event.Ts
		s.appState.CurrentSlackChannel = event.Channel
		log.Printf("📌 Updated current Slack thread timestamp to: %s in channel: %s", event.Ts, event.Channel)

		startConversationMessage := models.UnknownMessage{
			Type:    models.MessageTypeStartConversation,
			Payload: models.StartConversationPayload{Message: event.Text},
		}

		if err := s.wsClient.SendMessage(clientID, startConversationMessage); err != nil {
			return fmt.Errorf("failed to send start conversation message to client %s: %v", clientID, err)
		}
		log.Printf("🚀 Sent start conversation message to client %s", clientID)
	} else {
		log.Printf("💬 Bot mentioned in ongoing thread %s in channel %s", event.ThreadTs, event.Channel)

		userMessage := models.UnknownMessage{
			Type:    models.MessageTypeUserMessage,
			Payload: models.UserMessagePayload{Message: event.Text},
		}

		if err := s.wsClient.SendMessage(clientID, userMessage); err != nil {
			return fmt.Errorf("failed to send user message to client %s: %v", clientID, err)
		}
		log.Printf("💬 Sent user message to client %s", clientID)
	}

	log.Printf("📋 Completed successfully - processed Slack message event")
	return nil
}

func (s *AppService) RegisterAgent(clientID string) {
	log.Printf("📋 Starting to register agent for client %s", clientID)

	_, err := s.agentsService.CreateActiveAgent(clientID, nil)
	if err != nil {
		log.Printf("❌ Failed to register agent for client %s: %v", clientID, err)
		return
	}

	log.Printf("📋 Completed successfully - registered agent for client %s", clientID)
}

func (s *AppService) DeregisterAgent(clientID string) {
	log.Printf("📋 Starting to deregister agent for client %s", clientID)

	err := s.agentsService.DeleteActiveAgentByWsConnectionID(clientID)
	if err != nil {
		log.Printf("❌ Failed to deregister agent for client %s: %v", clientID, err)
		return
	}

	log.Printf("📋 Completed successfully - deregistered agent for client %s", clientID)
}

