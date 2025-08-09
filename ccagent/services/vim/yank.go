package vim

import (
	"strings"

	"ccagent/models"
)

type YankProcessor struct {
	vimService *VimService
}

func NewYankProcessor(vimService *VimService) *YankProcessor {
	return &YankProcessor{
		vimService: vimService,
	}
}

func (y *YankProcessor) ExecuteYank(text string, motion string, count int) *models.YankOperation {
	yankOp := &models.YankOperation{
		Register: "0",
		Type:     y.getYankType(motion),
	}

	switch motion {
	case "y", "Y":
		yankOp.Text = y.yankLines(text, count)
		yankOp.Type = models.YankTypeLine
	case "w":
		yankOp.Text = y.yankWords(text, count)
		yankOp.Type = models.YankTypeCharacter
	case "W":
		yankOp.Text = y.yankWORDS(text, count)
		yankOp.Type = models.YankTypeCharacter
	case "b":
		yankOp.Text = y.yankBackWords(text, count)
		yankOp.Type = models.YankTypeCharacter
	case "$":
		yankOp.Text = y.yankToEndOfLine(text)
		yankOp.Type = models.YankTypeCharacter
	case "0", "^":
		yankOp.Text = y.yankToStartOfLine(text)
		yankOp.Type = models.YankTypeCharacter
	case "G":
		yankOp.Text = y.yankToEndOfFile(text)
		yankOp.Type = models.YankTypeLine
	case "gg":
		yankOp.Text = y.yankToStartOfFile(text)
		yankOp.Type = models.YankTypeLine
	default:
		yankOp.Text = text
	}

	return yankOp
}

func (y *YankProcessor) getYankType(motion string) models.YankType {
	switch motion {
	case "y", "Y", "G", "gg":
		return models.YankTypeLine
	default:
		return models.YankTypeCharacter
	}
}

func (y *YankProcessor) yankLines(text string, count int) string {
	if count == 0 {
		count = 1
	}
	
	lines := strings.Split(text, "\n")
	if count > len(lines) {
		count = len(lines)
	}
	
	return strings.Join(lines[:count], "\n")
}

func (y *YankProcessor) yankWords(text string, count int) string {
	if count == 0 {
		count = 1
	}

	words := strings.Fields(text)
	if count > len(words) {
		count = len(words)
	}

	return strings.Join(words[:count], " ")
}

func (y *YankProcessor) yankWORDS(text string, count int) string {
	if count == 0 {
		count = 1
	}

	parts := strings.Split(text, " ")
	wordCount := 0
	endIdx := 0

	for i, part := range parts {
		if part != "" {
			wordCount++
			if wordCount >= count {
				endIdx = i + 1
				break
			}
		}
	}

	if endIdx == 0 {
		return text
	}

	return strings.Join(parts[:endIdx], " ")
}

func (y *YankProcessor) yankBackWords(text string, count int) string {
	if count == 0 {
		count = 1
	}

	words := strings.Fields(text)
	if count > len(words) {
		count = len(words)
	}

	startIdx := len(words) - count
	if startIdx < 0 {
		startIdx = 0
	}

	return strings.Join(words[startIdx:], " ")
}

func (y *YankProcessor) yankToEndOfLine(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

func (y *YankProcessor) yankToStartOfLine(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

func (y *YankProcessor) yankToEndOfFile(text string) string {
	return text
}

func (y *YankProcessor) yankToStartOfFile(text string) string {
	return text
}

func (y *YankProcessor) YankVisualSelection(text string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(text) {
		end = len(text)
	}
	if start > end {
		start, end = end, start
	}
	
	return text[start:end]
}