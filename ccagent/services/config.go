package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"ccagent/core/log"
	"ccagent/models"
	"ccagent/resources"

	"github.com/google/uuid"
)

const configDir = ".ccagent"
const configFile = "config.json"
const claudeSettingsFile = "claude-settings.json"
const claudeDir = "claude"
const claudeTargetSettingsFile = "settings.json"

type ConfigService struct{}

func NewConfigService() *ConfigService {
	return &ConfigService{}
}

func (s *ConfigService) InitCCAgentConfig() (*models.Config, error) {
	log.Info("📋 Starting to initialize CCAgent configuration")

	// Create .ccagent directory
	log.Info("Creating config directory: %s", configDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Error("Failed to create config directory: %v", err)
		return nil, err
	}

	// Add to gitignore
	if err := s.addToGitignore(configDir); err != nil {
		return nil, fmt.Errorf("failed to add to gitignore: %w", err)
	}

	// Get or create agent config
	config, err := s.getOrCreateConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get or create config: %w", err)
	}

	// Create claude-settings.json if it doesn't exist
	if err := s.createClaudeSettingsIfNeeded(); err != nil {
		return nil, fmt.Errorf("failed to create claude settings: %w", err)
	}

	// Copy claude-settings.json to .ccagent/claude/settings.json
	if err := s.copyClaudeSettingsToClaudeDir(); err != nil {
		return nil, fmt.Errorf("failed to copy claude settings: %w", err)
	}

	log.Info("📋 Completed successfully - initialized CCAgent config with agent ID: %s", config.AgentID)
	return config, nil
}

func (s *ConfigService) GetOrCreateConfig() (*models.Config, error) {
	return s.InitCCAgentConfig()
}

func (s *ConfigService) getOrCreateConfig() (*models.Config, error) {
	log.Info("📋 Starting to get or create config")
	configPath := filepath.Join(configDir, configFile)
	log.Info("Checking for config file at path: %s", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Info("Config file does not exist, creating new config")
		config, err := s.createConfig()
		if err != nil {
			return nil, err
		}
		log.Info("📋 Completed successfully - created new config with agent ID: %s", config.AgentID)
		return config, nil
	}

	log.Info("Config file exists, loading existing config")
	config, err := s.loadConfig(configPath)
	if err != nil {
		return nil, err
	}
	log.Info("📋 Completed successfully - loaded existing config with agent ID: %s", config.AgentID)
	return config, nil
}

func (s *ConfigService) createConfig() (*models.Config, error) {
	log.Info("📋 Starting to create new config")

	agentID := uuid.New().String()
	log.Info("Generated new agent ID: %s", agentID)
	config := &models.Config{
		AgentID: agentID,
	}

	configPath := filepath.Join(configDir, configFile)
	log.Info("Writing config file to path: %s", configPath)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Error("Failed to marshal config: %s", err)
		return nil, err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		log.Error("Failed to write config file: %v", err)
		return nil, err
	}

	log.Info("Config created successfully")
	log.Info("📋 Completed successfully - created config with agent ID: %s", config.AgentID)
	return config, nil
}

func (s *ConfigService) loadConfig(configPath string) (*models.Config, error) {
	log.Info("📋 Starting to load config from path: %s", configPath)
	log.Info("Loading config file from path: %s", configPath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Error("Failed to read config file: %v", err)
		return nil, err
	}

	var config models.Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Error("Failed to unmarshal config: %v", err)
		return nil, err
	}

	log.Info("Config loaded successfully with agent ID: %s", config.AgentID)
	log.Info("📋 Completed successfully - loaded config with agent ID: %s", config.AgentID)
	return &config, nil
}

func (s *ConfigService) addToGitignore(dir string) error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		log.Debug("Not a git repository, skipping gitignore update")
		return nil
	}

	log.Info("Adding directory to gitignore: %s", dir)
	gitignorePath := ".gitignore"

	var content string
	if data, err := os.ReadFile(gitignorePath); err == nil {
		content = string(data)
	}

	if contains(content, dir) {
		log.Debug("Directory already in gitignore: %s", dir)
		return nil
	}

	log.Info("Directory not in gitignore, adding it: %s", dir)
	f, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Error("Failed to open gitignore file: %v", err)
		return fmt.Errorf("failed to open gitignore file: %w", err)
	}
	defer f.Close()

	if len(content) > 0 && content[len(content)-1] != '\n' {
		f.WriteString("\n")
	}
	f.WriteString(dir + "\n")
	log.Info("Successfully added directory to gitignore: %s", dir)
	return nil
}

func (s *ConfigService) createClaudeSettingsIfNeeded() error {
	log.Info("📋 Starting to create claude-settings.json if needed")
	claudeSettingsPath := filepath.Join(configDir, claudeSettingsFile)

	if _, err := os.Stat(claudeSettingsPath); !os.IsNotExist(err) {
		log.Info("Claude settings file already exists, skipping creation")
		log.Info("📋 Completed successfully - claude-settings.json is ready")
		return nil
	}

	log.Info("Claude settings file does not exist, creating from default template")

	if err := os.WriteFile(claudeSettingsPath, resources.DefaultClaudeSettings, 0644); err != nil {
		log.Error("Failed to write claude settings file: %v", err)
		return fmt.Errorf("failed to write claude settings file: %w", err)
	}

	log.Info("Successfully created claude-settings.json from embedded template")
	log.Info("📋 Completed successfully - claude-settings.json is ready")
	return nil
}

func (s *ConfigService) copyClaudeSettingsToClaudeDir() error {
	log.Info("📋 Starting to copy claude-settings.json to claude directory")

	claudeSettingsPath := filepath.Join(configDir, claudeSettingsFile)
	claudeDirPath := filepath.Join(configDir, claudeDir)
	claudeTargetPath := filepath.Join(claudeDirPath, claudeTargetSettingsFile)

	// Create claude directory if it doesn't exist
	log.Info("Creating claude directory: %s", claudeDirPath)
	if err := os.MkdirAll(claudeDirPath, 0755); err != nil {
		log.Error("Failed to create claude directory: %v", err)
		return fmt.Errorf("failed to create claude directory: %w", err)
	}

	// Read the source file
	log.Info("Reading claude settings from: %s", claudeSettingsPath)
	data, err := os.ReadFile(claudeSettingsPath)
	if err != nil {
		log.Error("Failed to read claude settings file: %v", err)
		return fmt.Errorf("failed to read claude settings file: %w", err)
	}

	// Write to target location
	log.Info("Copying claude settings to: %s", claudeTargetPath)
	if err := os.WriteFile(claudeTargetPath, data, 0644); err != nil {
		log.Error("Failed to write claude target settings file: %v", err)
		return fmt.Errorf("failed to write claude target settings file: %w", err)
	}

	log.Info("📋 Completed successfully - copied claude-settings.json to claude directory")
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)+1] == substr+"\n" ||
			s[len(s)-len(substr)-1:] == "\n"+substr ||
			(len(s) > len(substr)+1 && s[len(s)-len(substr)-1:len(s)-1] == "\n"+substr))))
}
