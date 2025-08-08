package services

import (
	"bufio"
	"encoding/json"
	"strings"
)

// ClaudeMessage represents a message from Claude command output
type ClaudeMessage interface {
	GetType() string
	GetSessionID() string
}

// AssistantMessage represents an assistant message from Claude
type AssistantMessage struct {
	Type    string `json:"type"`
	Message struct {
		ID      string            `json:"id"`
		Type    string            `json:"type"`
		Content []json.RawMessage `json:"content"` // Use RawMessage to handle both text and tool_use content
	} `json:"message"`
	SessionID string `json:"session_id"`
}

func (a AssistantMessage) GetType() string {
	return a.Type
}

func (a AssistantMessage) GetSessionID() string {
	return a.SessionID
}

// UnknownClaudeMessage represents an unknown message type from Claude
type UnknownClaudeMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

func (u UnknownClaudeMessage) GetType() string {
	return u.Type
}

func (u UnknownClaudeMessage) GetSessionID() string {
	return u.SessionID
}

// SystemMessage represents a system message from Claude
type SystemMessage struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	SessionID string `json:"session_id"`
}

func (s SystemMessage) GetType() string {
	return s.Type
}

func (s SystemMessage) GetSessionID() string {
	return s.SessionID
}

// UserMessage represents a user message from Claude
type UserMessage struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"` // Can be string or array
	} `json:"message"`
	SessionID string `json:"session_id"`
}

func (u UserMessage) GetType() string {
	return u.Type
}

func (u UserMessage) GetSessionID() string {
	return u.SessionID
}

// ResultMessage represents a result message from Claude
type ResultMessage struct {
	Type          string  `json:"type"`
	Subtype       string  `json:"subtype"`
	IsError       bool    `json:"is_error"`
	DurationMs    int     `json:"duration_ms"`
	DurationAPIMs int     `json:"duration_api_ms"`
	NumTurns      int     `json:"num_turns"`
	Result        string  `json:"result"`
	SessionID     string  `json:"session_id"`
	TotalCostUsd  float64 `json:"total_cost_usd"`
}

func (r ResultMessage) GetType() string {
	return r.Type
}

func (r ResultMessage) GetSessionID() string {
	return r.SessionID
}

// MapClaudeOutputToMessages parses Claude command output into structured messages
// This is exported to allow reuse across different modules
func MapClaudeOutputToMessages(output string) ([]ClaudeMessage, error) {
	var messages []ClaudeMessage

	// Use a scanner with a larger buffer to handle long lines
	scanner := bufio.NewScanner(strings.NewReader(output))
	// Set a 1MB buffer to handle very long JSON lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse the message based on type
		message := parseClaudeMessage([]byte(line))
		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// parseClaudeMessage attempts to parse a JSON line into the appropriate message type
func parseClaudeMessage(lineBytes []byte) ClaudeMessage {
	// First, extract just the type to determine which struct to use
	var typeCheck struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(lineBytes, &typeCheck); err != nil {
		// If we can't even parse the type, return unknown message
		return UnknownClaudeMessage{
			Type:      "unknown",
			SessionID: "",
		}
	}

	// Parse based on type
	switch typeCheck.Type {
	case "assistant":
		var assistantMsg AssistantMessage
		if err := json.Unmarshal(lineBytes, &assistantMsg); err == nil {
			return assistantMsg
		}
	case "system":
		var systemMsg SystemMessage
		if err := json.Unmarshal(lineBytes, &systemMsg); err == nil {
			return systemMsg
		}
	case "user":
		var userMsg UserMessage
		if err := json.Unmarshal(lineBytes, &userMsg); err == nil {
			return userMsg
		}
	case "result":
		var resultMsg ResultMessage
		if err := json.Unmarshal(lineBytes, &resultMsg); err == nil {
			return resultMsg
		}
	}

	// If specific type parsing failed, try to extract basic info for unknown message
	var unknownMsg struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
	}

	if err := json.Unmarshal(lineBytes, &unknownMsg); err == nil {
		return UnknownClaudeMessage{
			Type:      unknownMsg.Type,
			SessionID: unknownMsg.SessionID,
		}
	}

	// Last resort - completely unknown message
	return UnknownClaudeMessage{
		Type:      "unknown",
		SessionID: "",
	}
}
