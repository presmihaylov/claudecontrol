package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"ccbackend/clients"
	"ccbackend/config"
	"ccbackend/db"
	"ccbackend/handlers"
	"ccbackend/middleware"
	"ccbackend/services"
	"ccbackend/usecases"
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
	processedSlackMessagesRepo := db.NewPostgresProcessedSlackMessagesRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	agentsService := services.NewAgentsService(agentsRepo)
	jobsService := services.NewJobsService(jobsRepo, processedSlackMessagesRepo)
	usersService := services.NewUsersService(usersRepo)
	slackOAuthClient := clients.NewConcreteSlackClient()
	slackIntegrationsService := services.NewSlackIntegrationsService(slackIntegrationsRepo, slackOAuthClient, cfg.SlackClientID, cfg.SlackClientSecret)

	// Create API key validator for WebSocket connections
	apiKeyValidator := func(apiKey string) (string, error) {
		integration, err := slackIntegrationsService.GetSlackIntegrationBySecretKey(apiKey)
		if err != nil {
			return "", err
		}
		return integration.ID.String(), nil
	}

	wsClient := clients.NewWebSocketClient(apiKeyValidator)

	// Initialize reliable message handler
	reliableMessageHandler := services.NewReliableMessageHandler(wsClient)

	// Initialize message processor for reliable outbound message delivery
	messageProcessor := services.NewMessageProcessor(wsClient)

	coreUseCase := usecases.NewCoreUseCase(wsClient, messageProcessor, agentsService, jobsService, slackIntegrationsService)
	wsHandler := handlers.NewWebSocketHandler(coreUseCase, slackIntegrationsService)
	slackHandler := handlers.NewSlackWebhooksHandler(cfg.SlackSigningSecret, coreUseCase, slackIntegrationsService)
	dashboardHandler := handlers.NewDashboardAPIHandler(usersService, slackIntegrationsService)
	authMiddleware := middleware.NewClerkAuthMiddleware(usersService, cfg.ClerkSecretKey)

	// Create a new router
	router := mux.NewRouter()

	// Setup endpoints with the new router
	wsClient.RegisterWithRouter(router)
	slackHandler.SetupEndpoints(router)
	dashboardHandler.SetupEndpoints(router, authMiddleware)

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			log.Printf("❌ Failed to write health check response: %v", err)
		}
	}).Methods("GET")

	// Register WebSocket hooks for agent lifecycle
	wsClient.RegisterConnectionHook(coreUseCase.RegisterAgent)
	wsClient.RegisterDisconnectionHook(coreUseCase.DeregisterAgent)

	// Register reliable message handler first (for deduplication and acknowledgements)
	wsClient.RegisterMessageHandler(func(client *clients.Client, msg any) {
		isAlreadyProcessed, err := reliableMessageHandler.ProcessReliableMessage(client, msg)
		if err != nil {
			log.Printf("❌ Error processing reliable message from client %s: %v", client.ID, err)
			return
		}
		if isAlreadyProcessed {
			return
		}

		// Check if this is an acknowledgement for outbound messages
		if msgMap, ok := msg.(map[string]any); ok {
			if msgType, ok := msgMap["type"].(string); ok && msgType == "ack" {
				if messageID, ok := msgMap["message_id"].(string); ok {
					messageProcessor.HandleAcknowledgement(messageID)
					return
				}
			}
		}

		// Handle message normally
		if err := wsHandler.HandleMessage(client, msg); err != nil {
			log.Printf("❌ Error handling message from client %s: %v", client.ID, err)
			// Do not mark message as processed - let it be retried from ccagent
			return
		}
		// Mark message as processed after successful handling
		if err := reliableMessageHandler.MarkMessageProcessed(client, msg); err != nil {
			log.Printf("❌ Error marking message as processed from client %s: %v", client.ID, err)
		}
	})

	// Start periodic broadcast of CheckIdleJobs, healthcheck, cleanup of inactive agents, and processing of queued jobs
	cleanupTicker := time.NewTicker(2 * time.Minute)
	go func() {
		for range cleanupTicker.C {
			if err := coreUseCase.ProcessQueuedJobs(); err != nil {
				log.Printf("⚠️ Periodic queued job processing encountered errors: %v", err)
			}
			if err := coreUseCase.BroadcastCheckIdleJobs(); err != nil {
				log.Printf("⚠️ Periodic CheckIdleJobs broadcast encountered errors: %v", err)
			}
			if err := coreUseCase.BroadcastHealthcheck(); err != nil {
				log.Printf("⚠️ Periodic healthcheck broadcast encountered errors: %v", err)
			}
			if err := coreUseCase.CleanupInactiveAgents(); err != nil {
				log.Printf("⚠️ Periodic inactive agent cleanup encountered errors: %v", err)
			}
		}
	}()
	defer cleanupTicker.Stop()

	// Setup CORS middleware
	allowedOrigins := strings.Split(cfg.CORSAllowedOrigins, ",")
	for i, origin := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(origin)
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Setup and handle graceful shutdown
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           c.Handler(router),
		ReadHeaderTimeout: 30 * time.Second,
	}

	return handleGracefulShutdown(server)
}

func handleGracefulShutdown(server *http.Server) error {
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
