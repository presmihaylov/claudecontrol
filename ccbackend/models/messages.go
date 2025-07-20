package models

// Message types
const (
	MessageTypeStartConversation = "start_conversation_v1"
	MessageTypeUserMessage       = "user_message_v1"
	MessageTypeAssistantMessage  = "assistant_message_v1"
)

type UnknownMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type StartConversationPayload struct {
	Message string `json:"message"`
}

type UserMessagePayload struct {
	Message string `json:"message"`
}

type AssistantMessagePayload struct {
	Message string `json:"message"`
}
