package handlers

import (
	"encoding/json"
	"log"

	"ccbackend/models"
	"ccbackend/services"
)

type WebSocketHandler struct {
	appService *services.AppService
}

func NewWebSocketHandler(appService *services.AppService) *WebSocketHandler {
	return &WebSocketHandler{
		appService: appService,
	}
}

func (h *WebSocketHandler) HandleMessage(clientID string, msg any) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("❌ Failed to marshal message from client %s: %v", clientID, err)
		return
	}

	var parsedMsg models.UnknownMessage
	if err := json.Unmarshal(msgBytes, &parsedMsg); err != nil {
		log.Printf("❌ Failed to parse message from client %s: %v", clientID, err)
		return
	}

	switch parsedMsg.Type {
	case models.MessageTypeAssistantMessage:
		var payload models.AssistantMessagePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal assistant message payload from client %s: %v", clientID, err)
			return
		}

		log.Printf("🤖 Received assistant message from client %s: %s", clientID, payload.Message)
		if err := h.appService.ProcessAssistantMessage(payload); err != nil {
			log.Printf("❌ Failed to process assistant message from client %s: %v", clientID, err)
		}

	default:
		log.Printf("⚠️ Unknown message type '%s' from client %s", parsedMsg.Type, clientID)
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