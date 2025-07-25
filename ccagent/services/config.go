package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"ccagent/core/log"
	"ccagent/models"

	"github.com/google/uuid"
)

const configDir = ".ccagent"
const configFile = "config.json"

type ConfigService struct{}

func NewConfigService() *ConfigService {
	return &ConfigService{}
}

func (s *ConfigService) GetOrCreateConfig() (*models.Config, error) {
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
	log.Info("Creating config directory: %s", configDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Error("Failed to create config directory", "error", err)
		return nil, err
	}

	if err := s.addToGitignore(configDir); err != nil {
		return nil, fmt.Errorf("failed to add to gitignore: %w", err)
	}

	agentID := uuid.New().String()
	log.Info("Generated new agent ID: %s", agentID)
	config := &models.Config{
		AgentID: agentID,
	}

	configPath := filepath.Join(configDir, configFile)
	log.Info("Writing config file to path: %s", configPath)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Error("Failed to marshal config", "error", err)
		return nil, err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		log.Error("Failed to write config file", "error", err)
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
		log.Error("Failed to read config file", "error", err)
		return nil, err
	}

	var config models.Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Error("Failed to unmarshal config", "error", err)
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
	
	log.Info("Adding directory to gitignore", "dir", dir)
	gitignorePath := ".gitignore"
	
	var content string
	if data, err := os.ReadFile(gitignorePath); err == nil {
		content = string(data)
	}
	
	if !contains(content, dir) {
		log.Info("Directory not in gitignore, adding it", "dir", dir)
		f, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Error("Failed to open gitignore file", "error", err)
			return fmt.Errorf("failed to open gitignore file: %w", err)
		}
		defer f.Close()
		
		if len(content) > 0 && content[len(content)-1] != '\n' {
			f.WriteString("\n")
		}
		f.WriteString(dir + "\n")
		log.Info("Successfully added directory to gitignore", "dir", dir)
	} else {
		log.Debug("Directory already in gitignore", "dir", dir)
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)+1] == substr+"\n" || 
		s[len(s)-len(substr)-1:] == "\n"+substr || 
		(len(s) > len(substr)+1 && s[len(s)-len(substr)-1:len(s)-1] == "\n"+substr))))
}