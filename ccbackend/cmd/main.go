package main

import (
	"log"
	"net/http"
	"os"

	"ccbackend/clients"
	"ccbackend/config"
	"ccbackend/handlers"
	"ccbackend/models"
	"ccbackend/services"

	"github.com/slack-go/slack"
)

func main() {
	if err := run(); err != nil {
		log.Printf("❌ Fatal error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	slackClient := slack.New(cfg.SlackBotToken)
	appState := &models.AppState{}
	wsClient := clients.NewWebSocketClient()
	wsClient.StartWebsocketServer()
	
	appService := services.NewAppService(slackClient, appState, wsClient)
	wsHandler := handlers.NewWebSocketHandler(appService)
	slackHandler := handlers.NewSlackWebhooksHandler(slackClient, cfg.SlackSigningSecret, appService)
	slackHandler.SetupEndpoints()

	wsClient.RegisterMessageHandler(wsHandler.HandleMessage)

	log.Printf("✅ Listening on http://localhost:%s", cfg.Port)
	return http.ListenAndServe(":"+cfg.Port, nil)
}

