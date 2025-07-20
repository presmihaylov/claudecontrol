package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ccbackend/clients"
	"ccbackend/config"
	"ccbackend/db"
	"ccbackend/handlers"
	"ccbackend/services"
	"ccbackend/usecases"

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

	// Initialize database connection
	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	// Initialize repositories with shared connection
	agentsRepo := db.NewPostgresAgentsRepository(dbConn, cfg.DatabaseSchema)
	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)

	agentsService := services.NewAgentsService(agentsRepo)
	jobsService := services.NewJobsService(jobsRepo)

	// Clear all active agents on startup
	log.Printf("🧹 Cleaning up stale active agents from previous server runs")
	if err := agentsService.DeleteAllActiveAgents(); err != nil {
		log.Printf("⚠️ Failed to clear stale active agents: %v", err)
	}

	slackClient := slack.New(cfg.SlackBotToken)
	wsClient := clients.NewWebSocketClient()
	wsClient.StartWebsocketServer()
	
	coreUseCase := usecases.NewCoreUseCase(slackClient, wsClient, agentsService, jobsService)
	wsHandler := handlers.NewWebSocketHandler(coreUseCase)
	slackHandler := handlers.NewSlackWebhooksHandler(slackClient, cfg.SlackSigningSecret, coreUseCase)
	slackHandler.SetupEndpoints()

	// Register WebSocket hooks for agent lifecycle
	wsClient.RegisterConnectionHook(coreUseCase.RegisterAgent)
	wsClient.RegisterDisconnectionHook(coreUseCase.DeregisterAgent)
	wsClient.RegisterMessageHandler(wsHandler.HandleMessage)

	// Start periodic cleanup of idle jobs
	cleanupTicker := time.NewTicker(2 * time.Minute)
	go func() {
		for range cleanupTicker.C {
			coreUseCase.CleanupIdleJobs()
		}
	}()
	defer cleanupTicker.Stop()

	// Setup and handle graceful shutdown
	server := &http.Server{
		Addr: ":" + cfg.Port,
	}

	return handleGracefulShutdown(server, agentsService)
}

func handleGracefulShutdown(server *http.Server, agentsService *services.AgentsService) error {
	// Channel to listen for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("✅ Listening on http://localhost%s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Printf("🛑 Shutdown signal received, cleaning up...")

	// Clear all active agents on shutdown
	if err := agentsService.DeleteAllActiveAgents(); err != nil {
		log.Printf("⚠️ Failed to clear active agents on shutdown: %v", err)
	}

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server shutdown error: %v", err)
		return err
	}

	log.Printf("✅ Server stopped gracefully")
	return nil
}

