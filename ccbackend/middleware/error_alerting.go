package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"ccbackend/clients"
)

type SlackAlertConfig struct {
	WebhookURL  string
	Environment string
	AppName     string
	LogsURL     string
}

type ErrorAlertMiddleware struct {
	config        SlackAlertConfig
	alertedErrors map[string]time.Time // hash -> last alert time
	mutex         sync.RWMutex
	alertCooldown time.Duration // prevent spam
}

func NewErrorAlertMiddleware(config SlackAlertConfig) *ErrorAlertMiddleware {
	return &ErrorAlertMiddleware{
		config:        config,
		alertedErrors: make(map[string]time.Time),
		alertCooldown: 10 * time.Minute, // Don't alert same error more than once per 10min
	}
}

// HTTP Middleware - wraps HTTP handlers
func (m *ErrorAlertMiddleware) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer m.recoverAndAlert(fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

// WebSocket Message Handler Wrapper
func (m *ErrorAlertMiddleware) WrapMessageHandler(handler func(*clients.Client, any) error) func(*clients.Client, any) {
	return func(client *clients.Client, msg any) {
		defer m.recoverAndAlert(fmt.Sprintf("WebSocket message from client %s", client.ID))

		if err := handler(client, msg); err != nil {
			m.alertOnError(err, fmt.Sprintf("WebSocket message handler (client: %s)", client.ID))
		}
	}
}

// WebSocket Hook Wrapper
func (m *ErrorAlertMiddleware) WrapConnectionHook(hook func(*clients.Client) error) func(*clients.Client) error {
	return func(client *clients.Client) error {
		defer m.recoverAndAlert(fmt.Sprintf("WebSocket connection hook for client %s", client.ID))

		if err := hook(client); err != nil {
			m.alertOnError(err, fmt.Sprintf("WebSocket connection hook (client: %s)", client.ID))
			return err
		}
		return nil
	}
}

// Background Task Wrapper
func (m *ErrorAlertMiddleware) WrapBackgroundTask(taskName string, task func() error) func() error {
	return func() error {
		defer m.recoverAndAlert(fmt.Sprintf("Background task: %s", taskName))

		if err := task(); err != nil {
			m.alertOnError(err, fmt.Sprintf("Background task: %s", taskName))
			return err
		}
		return nil
	}
}

// Core error alerting logic
func (m *ErrorAlertMiddleware) alertOnError(err error, contextMsg string) {
	errorMsg := fmt.Sprintf("%s: %v", contextMsg, err)

	// Create hash of error for deduplication
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(errorMsg)))

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if we've alerted for this error recently
	if lastAlert, exists := m.alertedErrors[hash]; exists {
		if time.Since(lastAlert) < m.alertCooldown {
			return // Skip alert - too recent
		}
	}

	// Send alert asynchronously
	go m.sendSlackAlert(errorMsg, contextMsg)
	m.alertedErrors[hash] = time.Now()
}

func (m *ErrorAlertMiddleware) recoverAndAlert(contextMsg string) {
	if r := recover(); r != nil {
		errorMsg := fmt.Sprintf("%s: PANIC - %v", contextMsg, r)
		log.Printf("❌ %s", errorMsg)
		go m.sendSlackAlert(errorMsg, contextMsg+" (PANIC)")
	}
}

func (m *ErrorAlertMiddleware) sendSlackAlert(errorMsg, contextMsg string) {
	if m.config.WebhookURL == "" {
		return // Slack alerts disabled
	}

	payload := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]any{
					"type": "plain_text",
					"text": fmt.Sprintf("🚨 %s[%s] Error Alert",
						func() string {
							if m.config.Environment == "dev" {
								return "[dev] "
							}
							return ""
						}(), m.config.AppName),
					"emoji": true,
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Service:* %s", m.config.AppName)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Environment:* %s", m.config.Environment)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Context:* %s", contextMsg)},
				},
			},
			{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Error:*\n```%s```", errorMsg),
				},
			},
			{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": fmt.Sprintf("🔗 <%s|View Logs>", m.config.LogsURL),
				},
			},
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	// Create request with context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", m.config.WebhookURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		log.Printf("❌ Failed to create Slack alert request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("❌ Failed to send Slack alert: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Slack alert failed with status: %d", resp.StatusCode)
	}
}
