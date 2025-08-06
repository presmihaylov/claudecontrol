package clients

import claudeclient "ccagent/clients/claude"

func NewClaudeClient(permissionMode string) ClaudeClient {
	return claudeclient.NewClaudeClient(permissionMode)
}
