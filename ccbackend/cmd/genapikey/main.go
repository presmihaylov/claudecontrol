package main

import (
	"fmt"
	"log"

	"ccbackend/core"
)

func main() {
	log.Printf("🔑 Generating new CCAgent API key...")

	// Generate a new secret key with "sys" prefix for ccagent system use
	apiKey, err := core.NewSecretKey("sys")
	if err != nil {
		log.Fatalf("❌ Failed to generate API key: %v", err)
	}

	fmt.Printf("Generated API Key: %s\n", apiKey)
	log.Printf("✅ Successfully generated CCAgent API key")
}