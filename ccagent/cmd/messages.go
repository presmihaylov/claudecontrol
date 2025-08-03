package main

// Message types
const (
	MessageTypeStartConversation         = "start_conversation_v1"
	MessageTypeStartConversationResponse = "start_conversation_response_v1"
	MessageTypeUserMessage               = "user_message_v1"
	MessageTypeAssistantMessage          = "assistant_message_v1"
	MessageTypeJobUnassigned             = "job_unassigned_v1"
	MessageTypeSystemMessage             = "system_message_v1"
	MessageTypeProcessingSlackMessage    = "processing_slack_message_v1"
	MessageTypeCheckIdleJobs             = "check_idle_jobs_v1"
	MessageTypeJobComplete               = "job_complete_v1"
	MessageTypeHealthcheckCheck          = "healthcheck_check_v1"
	MessageTypeHealthcheckAck            = "healthcheck_ack_v1"
	MessageTypeAcknowledgement           = "acknowledgement_v1"
)

type UnknownMessage struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type StartConversationPayload struct {
	JobID            string `json:"job_id"`
	Message          string `json:"message"`
	SlackMessageID   string `json:"slack_message_id"`
	SlackMessageLink string `json:"slack_message_link"`
}

type StartConversationResponsePayload struct {
	SessionID string `json:"sessionID"`
	Message   string `json:"message"`
}

type UserMessagePayload struct {
	JobID            string `json:"job_id"`
	Message          string `json:"message"`
	SlackMessageID   string `json:"slack_message_id"`
	SlackMessageLink string `json:"slack_message_link"`
}

type AssistantMessagePayload struct {
	JobID          string `json:"job_id"`
	Message        string `json:"message"`
	SlackMessageID string `json:"slack_message_id"`
}

type JobUnassignedPayload struct{}

type SystemMessagePayload struct {
	Message        string `json:"message"`
	SlackMessageID string `json:"slack_message_id"`
}

type ProcessingSlackMessagePayload struct {
	SlackMessageID string `json:"slack_message_id"`
}

type CheckIdleJobsPayload struct {
	// Empty payload - agent checks all its jobs
}

type JobCompletePayload struct {
	JobID  string `json:"job_id"`
	Reason string `json:"reason"`
}

type HealthcheckCheckPayload struct {
	// Empty payload - simple ping from backend
}

type HealthcheckAckPayload struct {
	// Empty payload - simple pong response to backend
}

type AcknowledgementPayload struct {
	MessageID string `json:"message_id"`
}
