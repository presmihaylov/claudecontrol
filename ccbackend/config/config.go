package config

import (
	"fmt"
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

type DiscordConfig struct {
	ClientID     string
	ClientSecret string
	BotToken     string
}

type GitHubConfig struct {
	ClientID       string
	ClientSecret   string
	AppID          string
	AppPrivateKey  string
}

type SSHConfig struct {
	DefaultHost       string
	PrivateKeyBase64  string
}

type ClerkConfig struct {
	SecretKey string
}

type AppConfig struct {
	// Core configuration (always required)
	DatabaseURL        string
	DatabaseSchema     string
	Port               string // Optional with default "8080"
	CORSAllowedOrigins string // Optional with default "*"
	Environment        string
	ServerLogsURL      string

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

	// Slack configuration (all required for now)
	slackSigningSecret, err := getEnvRequired("SLACK_SIGNING_SECRET")
	if err != nil {
		return nil, err
	}

	slackClientID, err := getEnvRequired("SLACK_CLIENT_ID")
	if err != nil {
		return nil, err
	}

	slackClientSecret, err := getEnvRequired("SLACK_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}

	// Discord configuration (all required for now)
	discordClientID, err := getEnvRequired("DISCORD_CLIENT_ID")
	if err != nil {
		return nil, err
	}

	discordClientSecret, err := getEnvRequired("DISCORD_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}

	discordBotToken, err := getEnvRequired("DISCORD_BOT_TOKEN")
	if err != nil {
		return nil, err
	}

	// GitHub configuration (all required for now)
	githubClientID, err := getEnvRequired("GITHUB_CLIENT_ID")
	if err != nil {
		return nil, err
	}

	githubClientSecret, err := getEnvRequired("GITHUB_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}

	githubAppID, err := getEnvRequired("GITHUB_APP_ID")
	if err != nil {
		return nil, err
	}

	githubAppPrivateKey, err := getEnvRequired("GITHUB_APP_PRIVATE_KEY")
	if err != nil {
		return nil, err
	}

	// SSH configuration (all required for now)
	defaultSSHHost, err := getEnvRequired("DEFAULT_SSH_HOST")
	if err != nil {
		return nil, err
	}

	sshPrivateKeyBase64, err := getEnvRequired("SSH_PRIVATE_KEY_B64")
	if err != nil {
		return nil, err
	}

	// Clerk configuration (required for now)
	clerkSecretKey, err := getEnvRequired("CLERK_SECRET_KEY")
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

		// Slack configuration
		SlackConfig: SlackConfig{
			SigningSecret:   slackSigningSecret,
			ClientID:        slackClientID,
			ClientSecret:    slackClientSecret,
			AlertWebhookURL: os.Getenv("SLACK_ALERT_WEBHOOK_URL"),
			SalesWebhookURL: os.Getenv("SLACK_SALES_WEBHOOK_URL"),
		},

		// Discord configuration
		DiscordConfig: DiscordConfig{
			ClientID:     discordClientID,
			ClientSecret: discordClientSecret,
			BotToken:     discordBotToken,
		},

		// GitHub configuration
		GitHubConfig: GitHubConfig{
			ClientID:       githubClientID,
			ClientSecret:   githubClientSecret,
			AppID:          githubAppID,
			AppPrivateKey:  githubAppPrivateKey,
		},

		// SSH configuration
		SSHConfig: SSHConfig{
			DefaultHost:      defaultSSHHost,
			PrivateKeyBase64: sshPrivateKeyBase64,
		},

		// Clerk configuration
		ClerkConfig: ClerkConfig{
			SecretKey: clerkSecretKey,
		},
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