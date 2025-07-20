package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	SlackBotToken      string
	SlackSigningSecret string
	Port               string
	DatabaseURL        string
	DatabaseSchema     string
}

func LoadConfig() (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️ Could not load .env file, continuing with system env vars")
	}

	slackBotToken, err := getEnvRequired("SLACK_BOT_TOKEN")
	if err != nil {
		return nil, err
	}

	slackSigningSecret, err := getEnvRequired("SLACK_SIGNING_SECRET")
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

	config := &AppConfig{
		SlackBotToken:      slackBotToken,
		SlackSigningSecret: slackSigningSecret,
		Port:               getEnvWithDefault("PORT", "8080"),
		DatabaseURL:        databaseURL,
		DatabaseSchema:     databaseSchema,
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