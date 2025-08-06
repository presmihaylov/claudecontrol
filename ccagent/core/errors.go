package core

// ClaudeParseError represents a failure to parse Claude output with error log file path
type ClaudeParseError struct {
	Message     string
	LogFilePath string
	OriginalErr error
}

func (e *ClaudeParseError) Error() string {
	return e.Message
}

// IsClaudeParseError checks if an error is a ClaudeParseError
func IsClaudeParseError(err error) (*ClaudeParseError, bool) {
	parseErr, ok := err.(*ClaudeParseError)
	return parseErr, ok
}
