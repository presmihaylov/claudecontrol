package handlers

import (
	"encoding/json"
	"log"

	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases"
)

type WebSocketHandler struct {
	coreUseCase              *usecases.CoreUseCase
	slackIntegrationsService *services.SlackIntegrationsService
}

func NewWebSocketHandler(coreUseCase *usecases.CoreUseCase, slackIntegrationsService *services.SlackIntegrationsService) *WebSocketHandler {
	return &WebSocketHandler{
		coreUseCase:              coreUseCase,
		slackIntegrationsService: slackIntegrationsService,
	}
}

func (h *WebSocketHandler) HandleMessage(client *clients.Client, msg any) {
	// Log the slack integration associated with this client
	log.Printf("🔑 Processing message from client %s (Slack Integration ID: %s)", client.ID, client.SlackIntegrationID)

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("❌ Failed to marshal message from client %s: %v", client.ID, err)
		return
	}

	var parsedMsg models.UnknownMessage
	if err := json.Unmarshal(msgBytes, &parsedMsg); err != nil {
		log.Printf("❌ Failed to parse message from client %s: %v", client.ID, err)
		return
	}

	switch parsedMsg.Type {
	case models.MessageTypeAssistantMessage:
		var payload models.AssistantMessagePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal assistant message payload from client %s: %v", client.ID, err)
			return
		}

		log.Printf("🤖 Received assistant message from client %s: %s", client.ID, payload.Message)
		if err := h.coreUseCase.ProcessAssistantMessage(client.ID, payload, client.SlackIntegrationID); err != nil {
			log.Printf("❌ Failed to process assistant message from client %s: %v", client.ID, err)
			return
		}

	case models.MessageTypeSystemMessage:
		var payload models.SystemMessagePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal system message payload from client %s: %v", client.ID, err)
			return
		}

		log.Printf("⚙️ Received system message from client %s: %s", client.ID, payload.Message)
		if err := h.coreUseCase.ProcessSystemMessage(client.ID, payload, client.SlackIntegrationID); err != nil {
			log.Printf("❌ Failed to process system message from client %s: %v", client.ID, err)
			return
		}

	case models.MessageTypeProcessingSlackMessage:
		var payload models.ProcessingSlackMessagePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal processing slack message payload from client %s: %v", client.ID, err)
			return
		}

		log.Printf("🔔 Received processing slack message notification from client %s for message: %s", client.ID, payload.SlackMessageID)
		if err := h.coreUseCase.ProcessProcessingSlackMessage(client.ID, payload, client.SlackIntegrationID); err != nil {
			log.Printf("❌ Failed to process processing slack message notification from client %s: %v", client.ID, err)
			return
		}

	case models.MessageTypeJobComplete:
		var payload models.JobCompletePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal job complete payload from client %s: %v", client.ID, err)
			return
		}

		log.Printf("✅ Received job complete notification from client %s for job: %s, reason: %s", client.ID, payload.JobID, payload.Reason)
		if err := h.coreUseCase.ProcessJobComplete(client.ID, payload, client.SlackIntegrationID); err != nil {
			log.Printf("❌ Failed to process job complete notification from client %s: %v", client.ID, err)
			return
		}

	case models.MessageTypeHealthcheckAck:
		var payload models.HealthcheckAckPayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal healthcheck ack payload from client %s: %v", client.ID, err)
			return
		}

		log.Printf("💓 Received healthcheck ack from client %s", client.ID)
		if err := h.coreUseCase.ProcessHealthcheckAck(client.ID, payload, client.SlackIntegrationID); err != nil {
			log.Printf("❌ Failed to process healthcheck ack from client %s: %v", client.ID, err)
			return
		}

	case models.MessageTypeHealthcheckCheck:
		var payload models.HealthcheckCheckPayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal healthcheck check payload from client %s: %v", client.ID, err)
			return
		}

		log.Printf("💓 Received healthcheck check from client %s - sending ack", client.ID)

		// Send healthcheck ack back to the client
		if err := h.coreUseCase.SendHealthcheckAck(client.ID, client.SlackIntegrationID); err != nil {
			log.Printf("❌ Failed to send healthcheck ack to client %s: %v", client.ID, err)
			return
		}

	default:
		log.Printf("⚠️ Unknown message type '%s' from client %s", parsedMsg.Type, client.ID)
		return
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
