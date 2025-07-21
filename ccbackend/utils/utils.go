package utils

import (
	"github.com/eritikass/githubmarkdownconvertergo"
)

// ConvertMarkdownToSlack converts standard markdown format to Slack's mrkdwn format
// Examples:
// - **bold** → *bold*
// - [link text](url) → <url|link text>
// - ~~strikethrough~~ → ~strikethrough~
func ConvertMarkdownToSlack(message string) string {
	return githubmarkdownconvertergo.Slack(message)
}