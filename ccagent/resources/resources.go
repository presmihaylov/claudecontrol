package resources

import (
	_ "embed"
)

//go:embed claude-settings.json
var DefaultClaudeSettings []byte
