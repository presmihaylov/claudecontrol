package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type AgentIdentity struct {
	AgentID    string `json:"agent_id"`
	CreatedAt  string `json:"created_at"`
	MachineName string `json:"machine_name"`
}

type IdentityService struct {
	configPath string
	identity   *AgentIdentity
}

func NewIdentityService() (*IdentityService, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "ccagent")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	service := &IdentityService{
		configPath: filepath.Join(configDir, "identity.json"),
	}

	if err := service.loadOrCreateIdentity(); err != nil {
		return nil, err
	}

	return service, nil
}

func (s *IdentityService) loadOrCreateIdentity() error {
	// Try to load existing identity
	data, err := os.ReadFile(s.configPath)
	if err == nil {
		var identity AgentIdentity
		if err := json.Unmarshal(data, &identity); err == nil {
			s.identity = &identity
			return nil
		}
	}

	// Create new identity if not found or invalid
	hostname, _ := os.Hostname()
	s.identity = &AgentIdentity{
		AgentID:     uuid.New().String(),
		CreatedAt:   time.Now().Format(time.RFC3339),
		MachineName: hostname,
	}

	// Save to disk
	data, err = json.MarshalIndent(s.identity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal identity: %w", err)
	}

	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save identity: %w", err)
	}

	return nil
}

func (s *IdentityService) GetAgentID() string {
	return s.identity.AgentID
}

func (s *IdentityService) GetIdentity() *AgentIdentity {
	return s.identity
}