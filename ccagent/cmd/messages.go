package main

// Message types
const (
	MessageTypePing                        = "ping"
	MessageTypePong                        = "pong"
	MessageTypeStartConversation           = "start_conversation_v1"
	MessageTypeStartConversationResponse   = "start_conversation_response_v1"
	MessageTypeUserMessage                 = "user_message_v1"
	MessageTypeAssistantMessage           = "assistant_message_v1"
)

type UnknownMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type PingPayload struct{}

type PongPayload struct{}

type StartConversationPayload struct {
	Message string `json:"message"`
}

type StartConversationResponsePayload struct {
	SessionID string `json:"sessionID"`
	Message   string `json:"message"`
}

type UserMessagePayload struct {
	Message string `json:"message"`
}

type AssistantMessagePayload struct {
	Message string `json:"message"`
}