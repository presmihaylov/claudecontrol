package models

// Message types
const (
	MessageTypeStartConversation     = "start_conversation_v1"
	MessageTypeUserMessage           = "user_message_v1"
	MessageTypeAssistantMessage      = "assistant_message_v1"
	MessageTypeJobUnassigned         = "job_unassigned_v1"
	MessageTypeSystemMessage         = "system_message_v1"
	MessageTypeProcessingSlackMessage = "processing_slack_message_v1"
	MessageTypeCheckIdleJobs         = "check_idle_jobs_v1"
	MessageTypeJobComplete           = "job_complete_v1"
	MessageTypeHealthcheckCheck      = "healthcheck_check_v1"
	MessageTypeHealthcheckAck        = "healthcheck_ack_v1"
	MessageTypeAgentHealthcheckPing  = "agent_healthcheck_ping_v1"
	MessageTypeAgentHealthcheckPong  = "agent_healthcheck_pong_v1"
)

type UnknownMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type StartConversationPayload struct {
	JobID           string `json:"job_id"`
	Message         string `json:"message"`
	SlackMessageID  string `json:"slack_message_id"`
	SlackMessageLink string `json:"slack_message_link"`
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

type JobUnassignedPayload struct {}

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
	// Empty payload - simple ping to agent
}

type HealthcheckAckPayload struct {
	// Empty payload - simple pong response from agent
}

type AgentHealthcheckPingPayload struct {
	// Empty payload - simple ping from agent to backend
}

type AgentHealthcheckPongPayload struct {
	// Empty payload - simple pong response from backend to agent
}
