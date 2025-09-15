package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type SlackConfig struct {
	SigningSecret   string
	ClientID        string
	ClientSecret    string
	AlertWebhookURL string
	SalesWebhookURL string
}

// IsConfigured returns true if all required Slack configuration is present
func (c SlackConfig) IsConfigured() bool {
	return c.SigningSecret != "" &&
		c.ClientID != "" &&
		c.ClientSecret != ""
	// Note: AlertWebhookURL and SalesWebhookURL are optional
}

type DiscordConfig struct {
	ClientID     string
	ClientSecret string
	BotToken     string
}

// IsConfigured returns true if all required Discord configuration is present
func (c DiscordConfig) IsConfigured() bool {
	return c.ClientID != "" &&
		c.ClientSecret != "" &&
		c.BotToken != ""
}

type GitHubConfig struct {
	ClientID      string
	ClientSecret  string
	AppID         string
	AppPrivateKey string
}

// IsConfigured returns true if all required GitHub configuration is present
func (c GitHubConfig) IsConfigured() bool {
	return c.ClientID != "" &&
		c.ClientSecret != "" &&
		c.AppID != "" &&
		c.AppPrivateKey != ""
}

type SSHConfig struct {
	DefaultHost      string
	PrivateKeyBase64 string
	KnownHostsFile   string
}

// IsConfigured returns true if all required SSH configuration is present
func (c SSHConfig) IsConfigured() bool {
	return c.DefaultHost != "" &&
		c.PrivateKeyBase64 != "" &&
		c.KnownHostsFile != ""
}

type ClerkConfig struct {
	SecretKey string
}

// IsConfigured returns true if all required Clerk configuration is present
func (c ClerkConfig) IsConfigured() bool {
	return c.SecretKey != ""
}

type AppConfig struct {
	// Core configuration (always required)
	DatabaseURL        string
	DatabaseSchema     string
	Port               string // Optional with default "8080"
	CORSAllowedOrigins string // Optional with default "*"
	Environment        string
	ServerLogsURL      string
	UseStrictConfig    bool // If true, error when any integration is not fully configured

	// Integration configurations (grouped)
	SlackConfig   SlackConfig
	DiscordConfig DiscordConfig
	GitHubConfig  GitHubConfig
	SSHConfig     SSHConfig
	ClerkConfig   ClerkConfig
}

func LoadConfig() (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️ Could not load .env file, continuing with system env vars")
	}

	// Core required configuration
	databaseURL, err := getEnvRequired("DB_URL")
	if err != nil {
		return nil, err
	}

	databaseSchema, err := getEnvRequired("DB_SCHEMA")
	if err != nil {
		return nil, err
	}

	config := &AppConfig{
		// Core configuration
		DatabaseURL:        databaseURL,
		DatabaseSchema:     databaseSchema,
		Port:               getEnvWithDefault("PORT", "8080"),
		CORSAllowedOrigins: getEnvWithDefault("CORS_ALLOWED_ORIGINS", "*"),
		Environment:        getEnvWithDefault("ENVIRONMENT", "dev"),
		ServerLogsURL:      getEnvWithDefault("SERVER_LOGS_URL", ""),
		UseStrictConfig:    getEnvWithDefault("USE_STRICT_CONFIG", "true") == "true",

		// Slack configuration (optional)
		SlackConfig: SlackConfig{
			SigningSecret:   os.Getenv("SLACK_SIGNING_SECRET"),
			ClientID:        os.Getenv("SLACK_CLIENT_ID"),
			ClientSecret:    os.Getenv("SLACK_CLIENT_SECRET"),
			AlertWebhookURL: os.Getenv("SLACK_ALERT_WEBHOOK_URL"),
			SalesWebhookURL: os.Getenv("SLACK_SALES_WEBHOOK_URL"),
		},

		// Discord configuration (optional)
		DiscordConfig: DiscordConfig{
			ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
			ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
			BotToken:     os.Getenv("DISCORD_BOT_TOKEN"),
		},

		// GitHub configuration (optional)
		GitHubConfig: GitHubConfig{
			ClientID:      os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret:  os.Getenv("GITHUB_CLIENT_SECRET"),
			AppID:         os.Getenv("GITHUB_APP_ID"),
			AppPrivateKey: os.Getenv("GITHUB_APP_PRIVATE_KEY"),
		},

		// SSH configuration (optional)
		SSHConfig: SSHConfig{
			DefaultHost:      os.Getenv("DEFAULT_SSH_HOST"),
			PrivateKeyBase64: os.Getenv("SSH_PRIVATE_KEY_B64"),
			KnownHostsFile:   os.Getenv("SSH_KNOWN_HOSTS"),
		},

		// Clerk configuration (optional)
		ClerkConfig: ClerkConfig{
			SecretKey: os.Getenv("CLERK_SECRET_KEY"),
		},
	}

	// Log which integrations are configured
	if config.SlackConfig.IsConfigured() {
		log.Printf("✅ Slack integration configured")
	} else {
		log.Printf("⚠️ Slack integration not configured - Slack features will be disabled")
		if config.UseStrictConfig {
			return nil, fmt.Errorf("slack integration is not fully configured (USE_STRICT_CONFIG=true)")
		}
	}

	if config.DiscordConfig.IsConfigured() {
		log.Printf("✅ Discord integration configured")
	} else {
		log.Printf("⚠️ Discord integration not configured - Discord features will be disabled")
		if config.UseStrictConfig {
			return nil, fmt.Errorf("discord integration is not fully configured (USE_STRICT_CONFIG=true)")
		}
	}

	if config.GitHubConfig.IsConfigured() {
		log.Printf("✅ GitHub integration configured")
	} else {
		log.Printf("⚠️ GitHub integration not configured - GitHub features will be disabled")
		if config.UseStrictConfig {
			return nil, fmt.Errorf("GitHub integration is not fully configured (USE_STRICT_CONFIG=true)")
		}
	}

	if config.SSHConfig.IsConfigured() {
		log.Printf("✅ SSH/Container integration configured")
	} else {
		log.Printf("⚠️ SSH/Container integration not configured - Container features will be disabled")
		if config.UseStrictConfig {
			return nil, fmt.Errorf("SSH/Container integration is not fully configured (USE_STRICT_CONFIG=true)")
		}
	}

	if config.ClerkConfig.IsConfigured() {
		log.Printf("✅ Clerk authentication configured")
	} else {
		log.Printf("⚠️ Clerk authentication not configured - Dashboard authentication will be disabled")
		if config.UseStrictConfig {
			return nil, fmt.Errorf("clerk authentication is not fully configured (USE_STRICT_CONFIG=true)")
		}
	}

	return config, nil
}

func getEnvRequired(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("%s is not set", key)
	}
	return value, nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
