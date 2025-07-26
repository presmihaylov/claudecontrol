package models

// Message types
const (
	MessageTypeStartConversation = "start_conversation_v1"
	MessageTypeUserMessage       = "user_message_v1"
	MessageTypeAssistantMessage  = "assistant_message_v1"
	MessageTypeJobUnassigned     = "job_unassigned_v1"
	MessageTypeSystemMessage     = "system_message_v1"
)

type UnknownMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type StartConversationPayload struct {
	Message         string `json:"message"`
	SlackMessageID  string `json:"slack_message_id"`
	SlackMessageLink string `json:"slack_message_link"`
}

type UserMessagePayload struct {
	Message          string `json:"message"`
	SlackMessageID   string `json:"slack_message_id"`
	SlackMessageLink string `json:"slack_message_link"`
}

type AssistantMessagePayload struct {
	Message        string `json:"message"`
	SlackMessageID string `json:"slack_message_id"`
}

type JobUnassignedPayload struct {}

type SystemMessagePayload struct {
	Message        string `json:"message"`
	SlackMessageID string `json:"slack_message_id"`
}
