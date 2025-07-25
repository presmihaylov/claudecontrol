package handlers

import (
	"encoding/json"
	"log"

	"ccbackend/models"
	"ccbackend/usecases"
)

type WebSocketHandler struct {
	coreUseCase *usecases.CoreUseCase
}

func NewWebSocketHandler(coreUseCase *usecases.CoreUseCase) *WebSocketHandler {
	return &WebSocketHandler{
		coreUseCase: coreUseCase,
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
		if err := h.coreUseCase.ProcessAssistantMessage(clientID, payload); err != nil {
			log.Printf("❌ Failed to process assistant message from client %s: %v", clientID, err)
			return
		}

	case models.MessageTypeSystemMessage:
		var payload models.SystemMessagePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal system message payload from client %s: %v", clientID, err)
			return
		}

		log.Printf("⚙️ Received system message from client %s: %s", clientID, payload.Message)
		if err := h.coreUseCase.ProcessSystemMessage(clientID, payload); err != nil {
			log.Printf("❌ Failed to process system message from client %s: %v", clientID, err)
			return
		}

	default:
		log.Printf("⚠️ Unknown message type '%s' from client %s", parsedMsg.Type, clientID)
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