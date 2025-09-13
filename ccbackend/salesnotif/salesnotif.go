package salesnotif

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"ccbackend/models"
)

var (
	instance *SalesNotifier
	once     sync.Once
)

type SalesNotifier struct {
	webhookURL  string
	environment string
	appName     string
	mu          sync.RWMutex
}

// Init initializes the global sales notifier instance
func Init(webhookURL, environment string) {
	once.Do(func() {
		instance = &SalesNotifier{
			webhookURL:  webhookURL,
			environment: environment,
			appName:     "Claude Control",
		}
	})
}

// New sends a sales notification message to Slack
func New(orgID models.OrgID, message string) {
	if instance == nil {
		log.Printf("⚠️ Sales notifier not initialized, skipping notification: %s", message)
		return
	}

	instance.send(orgID, message)
}

func (s *SalesNotifier) send(orgID models.OrgID, message string) {
	if s.webhookURL == "" {
		return // Sales notifications disabled
	}

	// Exclude specific organization from sales notifications
	if orgID == "org_1984B8360FCB1S6VZJSR87AJAM" {
		return // Organization excluded from sales notifications
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Send notification asynchronously to avoid blocking
	go s.sendSlackNotification(orgID, message)
}

func (s *SalesNotifier) sendSlackNotification(orgID models.OrgID, message string) {
	// Build fields array
	fields := []map[string]any{
		{"type": "mrkdwn", "text": fmt.Sprintf("*Service:* %s", s.appName)},
		{"type": "mrkdwn", "text": fmt.Sprintf("*Environment:* %s", s.environment)},
	}

	// Add OrgID field if provided
	if orgID != "" {
		fields = append(fields, map[string]any{
			"type": "mrkdwn",
			"text": fmt.Sprintf("*OrgID:* `%s`", string(orgID)),
		})
	}

	// Add timestamp
	fields = append(fields, map[string]any{
		"type": "mrkdwn",
		"text": fmt.Sprintf("*Timestamp:* %s", time.Now().Format("2006-01-02 15:04:05 UTC")),
	})

	payload := map[string]any{
		"blocks": []map[string]any{
			{
				"type":   "section",
				"fields": fields,
			},
			{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": fmt.Sprintf("📊 *Activity:*\n%s", message),
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ Failed to marshal sales notification payload: %v", err)
		return
	}

	// Create request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		log.Printf("❌ Failed to create sales notification request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("❌ Failed to send sales notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Sales notification failed with status: %d", resp.StatusCode)
		return
	}

	log.Printf("💰 Sales notification sent: %s", message)
}
