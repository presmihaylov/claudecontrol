package utils

import (
	"testing"
)

func TestConvertSlackPermalinkToDeepLink(t *testing.T) {
	tests := []struct {
		name        string
		permalink   string
		expected    string
		description string
	}{
		{
			name:        "Valid Slack permalink",
			permalink:   "https://myworkspace.slack.com/archives/C1234567890/p1640995200123456",
			expected:    "slack://channel?id=C1234567890",
			description: "Should convert standard Slack permalink to deep link",
		},
		{
			name:        "Slack permalink with thread",
			permalink:   "https://myworkspace.slack.com/archives/C1234567890/p1640995200123456?thread_ts=1640995200.123456",
			expected:    "slack://channel?id=C1234567890",
			description: "Should convert threaded Slack permalink to deep link",
		},
		{
			name:        "Already a deep link",
			permalink:   "slack://channel?team=T1234567890&id=C1234567890",
			expected:    "slack://channel?team=T1234567890&id=C1234567890",
			description: "Should return deep links unchanged",
		},
		{
			name:        "Invalid URL",
			permalink:   "not-a-valid-url",
			expected:    "not-a-valid-url",
			description: "Should return invalid URLs unchanged",
		},
		{
			name:        "Non-Slack URL",
			permalink:   "https://example.com/some/path",
			expected:    "https://example.com/some/path",
			description: "Should return non-Slack URLs unchanged",
		},
		{
			name:        "Enterprise Slack URL",
			permalink:   "https://mycompany.enterprise.slack.com/archives/C1234567890/p1640995200123456",
			expected:    "slack://channel?id=C1234567890",
			description: "Should handle enterprise Slack URLs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSlackPermalinkToDeepLink(tt.permalink)
			if result != tt.expected {
				t.Errorf("ConvertSlackPermalinkToDeepLink(%q) = %q, expected %q\nDescription: %s", 
					tt.permalink, result, tt.expected, tt.description)
			}
		})
	}
}