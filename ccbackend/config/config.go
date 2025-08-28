package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	SlackSigningSecret     string
	SlackClientID          string
	SlackClientSecret      string
	DiscordClientID        string
	DiscordClientSecret    string
	DiscordBotToken        string
	GitHubClientID         string
	GitHubClientSecret     string
	GitHubAppID            string
	GitHubAppPrivateKey    string
	Port                   string
	DatabaseURL            string
	DatabaseSchema         string
	ClerkSecretKey         string
	CORSAllowedOrigins     string
	SlackAlertWebhookURL   string
	Environment            string
	ServerLogsURL          string
	DefaultSSHHost         string
	SSHPrivateKeyBase64    string
}

func LoadConfig() (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️ Could not load .env file, continuing with system env vars")
	}

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

	databaseURL, err := getEnvRequired("DB_URL")
	if err != nil {
		return nil, err
	}

	databaseSchema, err := getEnvRequired("DB_SCHEMA")
	if err != nil {
		return nil, err
	}

	clerkSecretKey, err := getEnvRequired("CLERK_SECRET_KEY")
	if err != nil {
		return nil, err
	}

	corsAllowedOrigins, err := getEnvRequired("CORS_ALLOWED_ORIGINS")
	if err != nil {
		return nil, err
	}

	port, err := getEnvRequired("PORT")
	if err != nil {
		return nil, err
	}

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

	defaultSSHHost, err := getEnvRequired("DEFAULT_SSH_HOST")
	if err != nil {
		return nil, err
	}

	sshPrivateKeyBase64, err := getEnvRequired("SSH_PRIVATE_KEY_BASE64")
	if err != nil {
		return nil, err
	}

	config := &AppConfig{
		SlackSigningSecret:     slackSigningSecret,
		SlackClientID:          slackClientID,
		SlackClientSecret:      slackClientSecret,
		DiscordClientID:        discordClientID,
		DiscordClientSecret:    discordClientSecret,
		DiscordBotToken:        discordBotToken,
		GitHubClientID:         githubClientID,
		GitHubClientSecret:     githubClientSecret,
		GitHubAppID:            githubAppID,
		GitHubAppPrivateKey:    githubAppPrivateKey,
		Port:                   port,
		DatabaseURL:            databaseURL,
		DatabaseSchema:         databaseSchema,
		ClerkSecretKey:         clerkSecretKey,
		CORSAllowedOrigins:     corsAllowedOrigins,
		SlackAlertWebhookURL:   os.Getenv("SLACK_ALERT_WEBHOOK_URL"),
		Environment:            getEnvWithDefault("ENVIRONMENT", "dev"),
		ServerLogsURL:          getEnvWithDefault("SERVER_LOGS_URL", ""),
		DefaultSSHHost:         defaultSSHHost,
		SSHPrivateKeyBase64:    sshPrivateKeyBase64,
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
