package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestVerifySlackSignature(t *testing.T) {
	signingSecret := "test_signing_secret"
	handler := &SlackEventsHandler{
		signingSecret: signingSecret,
	}

	timestamp := time.Now().Unix()
	body := `{"type":"url_verification","challenge":"test_challenge"}`

	// Create expected signature
	baseString := fmt.Sprintf("v0:%d:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Create request with proper headers
	req, _ := http.NewRequest("POST", "/slack/events", strings.NewReader(body))
	req.Header.Set("X-Slack-Request-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Slack-Signature", expectedSignature)

	// Test valid signature
	err := handler.verifySlackSignature(req, []byte(body))
	if err != nil {
		t.Errorf("Expected valid signature to pass, got error: %v", err)
	}

	// Test invalid signature
	req.Header.Set("X-Slack-Signature", "v0=invalid_signature")
	err = handler.verifySlackSignature(req, []byte(body))
	if err == nil {
		t.Error("Expected invalid signature to fail")
	}

	// Test missing headers
	req.Header.Del("X-Slack-Signature")
	err = handler.verifySlackSignature(req, []byte(body))
	if err == nil {
		t.Error("Expected missing headers to fail")
	}

	// Test old timestamp
	oldTimestamp := time.Now().Unix() - 400 // 6+ minutes ago
	req.Header.Set("X-Slack-Request-Timestamp", strconv.FormatInt(oldTimestamp, 10))
	req.Header.Set("X-Slack-Signature", expectedSignature)
	err = handler.verifySlackSignature(req, []byte(body))
	if err == nil {
		t.Error("Expected old timestamp to fail")
	}
}
