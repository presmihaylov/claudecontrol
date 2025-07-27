package clients

import (
	"bufio"
	"encoding/json"
	"strings"
)

type ClaudeMessage interface {
	GetType() string
	GetSessionID() string
}

type AssistantMessage struct {
	Type    string `json:"type"`
	Message struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
	SessionID string `json:"session_id"`
}

func (a AssistantMessage) GetType() string {
	return a.Type
}

func (a AssistantMessage) GetSessionID() string {
	return a.SessionID
}

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

func mapClaudeOutputToMessages(output string) ([]ClaudeMessage, error) {
	var messages []ClaudeMessage
	
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		// Try to parse as AssistantMessage first
		var assistantMsg AssistantMessage
		if err := json.Unmarshal([]byte(line), &assistantMsg); err == nil && assistantMsg.Type == "assistant" {
			messages = append(messages, assistantMsg)
			continue
		}
		
		// Fallback to UnknownClaudeMessage
		var unknownMsg struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
		}
		
		if err := json.Unmarshal([]byte(line), &unknownMsg); err == nil {
			messages = append(messages, UnknownClaudeMessage{
				Type:      unknownMsg.Type,
				SessionID: unknownMsg.SessionID,
			})
		} else {
			// If even basic parsing fails, create unknown message
			messages = append(messages, UnknownClaudeMessage{
				Type:      "unknown",
				SessionID: "",
			})
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	return messages, nil
}