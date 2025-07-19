package services

import (
	"encoding/json"
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
	configPath := filepath.Join(configDir, configFile)
	log.Info("Checking for config file", "path", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Info("Config file does not exist, creating new config")
		return s.createConfig()
	}

	log.Info("Config file exists, loading existing config")
	return s.loadConfig(configPath)
}

func (s *ConfigService) createConfig() (*models.Config, error) {
	log.Info("Creating config directory", "dir", configDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Error("Failed to create config directory", "error", err)
		return nil, err
	}

	s.addToGitignore(configDir)

	agentID := uuid.New().String()
	log.Info("Generated new agent ID", "agentID", agentID)
	config := &models.Config{
		AgentID: agentID,
	}

	configPath := filepath.Join(configDir, configFile)
	log.Info("Writing config file", "path", configPath)
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
	return config, nil
}

func (s *ConfigService) loadConfig(configPath string) (*models.Config, error) {
	log.Info("Loading config file", "path", configPath)
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

	log.Info("Config loaded successfully", "agentID", config.AgentID)
	return &config, nil
}

func (s *ConfigService) addToGitignore(dir string) {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		log.Debug("Not a git repository, skipping gitignore update")
		return
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
			return
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
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)+1] == substr+"\n" || 
		s[len(s)-len(substr)-1:] == "\n"+substr || 
		(len(s) > len(substr)+1 && s[len(s)-len(substr)-1:len(s)-1] == "\n"+substr))))
}