package clients

import (
	"errors"
	"fmt"
)

// ErrClaudeCommandErr represents an error from the Claude command that includes the command output
type ErrClaudeCommandErr struct {
	Err    error  // The original command error
	Output string // The Claude command output (may contain JSON response)
}

func (e *ErrClaudeCommandErr) Error() string {
	return fmt.Sprintf("claude command failed: %v\nOutput: %s", e.Err, e.Output)
}

func (e *ErrClaudeCommandErr) Unwrap() error {
	return e.Err
}

// IsClaudeCommandErr checks if an error is a Claude command error
func IsClaudeCommandErr(err error) (*ErrClaudeCommandErr, bool) {
	var claudeErr *ErrClaudeCommandErr
	if errors.As(err, &claudeErr) {
		return claudeErr, true
	}
	return nil, false
}