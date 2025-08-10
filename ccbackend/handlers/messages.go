package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/usecases/core"
)

type MessagesHandler struct {
	coreUseCase *core.CoreUseCase
}

func NewMessagesHandler(coreUseCase *core.CoreUseCase) *MessagesHandler {
	return &MessagesHandler{
		coreUseCase: coreUseCase,
	}
}

func (h *MessagesHandler) HandleMessage(client *clients.Client, msg any) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("❌ Failed to marshal message from client %s: %v", client.ID, err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	var parsedMsg models.BaseMessage
	if err := json.Unmarshal(msgBytes, &parsedMsg); err != nil {
		log.Printf("❌ Failed to parse message from client %s: %v", client.ID, err)
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Log processing of message with message ID, organization ID, and agent ID
	log.Printf(
		"📋 Processing message %s from ccagent (Organization: %s, Agent: %s)",
		parsedMsg.ID,
		client.OrgID,
		client.AgentID,
	)

	switch parsedMsg.Type {
	case models.MessageTypeAssistantMessage:
		var payload models.AssistantMessagePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal assistant message payload from client %s: %v", client.ID, err)
			return fmt.Errorf("failed to unmarshal assistant message payload: %w", err)
		}

		log.Printf("🤖 Received assistant message from client %s", client.ID)
		if err := h.coreUseCase.ProcessAssistantMessage(context.Background(), client.ID, payload, client.OrgID); err != nil {
			log.Printf("❌ Failed to process assistant message from client %s: %v", client.ID, err)
			return fmt.Errorf("failed to process assistant message: %w", err)
		}

	case models.MessageTypeSystemMessage:
		var payload models.SystemMessagePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal system message payload from client %s: %v", client.ID, err)
			return fmt.Errorf("failed to unmarshal system message payload: %w", err)
		}

		log.Printf("⚙️ Received system message from client %s: %s", client.ID, payload.Message)
		if err := h.coreUseCase.ProcessSystemMessage(context.Background(), client.ID, payload, client.OrgID); err != nil {
			log.Printf("❌ Failed to process system message from client %s: %v", client.ID, err)
			return fmt.Errorf("failed to process system message: %w", err)
		}

	case models.MessageTypeProcessingMessage:
		var payload models.ProcessingMessagePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal processing slack message payload from client %s: %v", client.ID, err)
			return fmt.Errorf("failed to unmarshal processing slack message payload: %w", err)
		}

		log.Printf(
			"🔔 Received processing slack message notification from client %s for message: %s",
			client.ID,
			payload.ProcessedMessageID,
		)
		if err := h.coreUseCase.ProcessProcessingMessage(context.Background(), client.ID, payload, client.OrgID); err != nil {
			log.Printf("❌ Failed to process processing slack message notification from client %s: %v", client.ID, err)
			return fmt.Errorf("failed to process processing slack message: %w", err)
		}

	case models.MessageTypeJobComplete:
		var payload models.JobCompletePayload
		if err := unmarshalPayload(parsedMsg.Payload, &payload); err != nil {
			log.Printf("❌ Failed to unmarshal job complete payload from client %s: %v", client.ID, err)
			return fmt.Errorf("failed to unmarshal job complete payload: %w", err)
		}

		log.Printf(
			"✅ Received job complete notification from client %s for job: %s, reason: %s",
			client.ID,
			payload.JobID,
			payload.Reason,
		)
		if err := h.coreUseCase.ProcessJobComplete(context.Background(), client.ID, payload, client.OrgID); err != nil {
			log.Printf("❌ Failed to process job complete notification from client %s: %v", client.ID, err)
			return fmt.Errorf("failed to process job complete: %w", err)
		}

	default:
		log.Printf("⚠️ Unknown message type '%s' from client %s", parsedMsg.Type, client.ID)
		return fmt.Errorf("unknown message type: %s", parsedMsg.Type)
	}

	return nil
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
