package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	SlackSigningSecret string
	SlackClientID      string
	SlackClientSecret  string
	Port               string
	DatabaseURL        string
	DatabaseSchema     string
	ClerkSecretKey     string
	CORSAllowedOrigins string
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

	config := &AppConfig{
		SlackSigningSecret: slackSigningSecret,
		SlackClientID:      slackClientID,
		SlackClientSecret:  slackClientSecret,
		Port:               port,
		DatabaseURL:        databaseURL,
		DatabaseSchema:     databaseSchema,
		ClerkSecretKey:     clerkSecretKey,
		CORSAllowedOrigins: corsAllowedOrigins,
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
