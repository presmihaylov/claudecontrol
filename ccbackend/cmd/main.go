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
	slackintegrations "ccbackend/services/slack_integrations"
	"ccbackend/services/users"
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

	// Initialize error alert middleware
	alertMiddleware := middleware.NewErrorAlertMiddleware(middleware.SlackAlertConfig{
		WebhookURL:  cfg.SlackAlertWebhookURL,
		Environment: cfg.Environment,
		AppName:     "ccbackend",
		LogsURL:     cfg.ServerLogsURL,
	})

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
	usersService := users.NewUsersService(usersRepo)
	slackOAuthClient := clients.NewConcreteSlackClient()
	slackIntegrationsService := slackintegrations.NewSlackIntegrationsService(slackIntegrationsRepo, slackOAuthClient, cfg.SlackClientID, cfg.SlackClientSecret)

	// Create API key validator for WebSocket connections
	apiKeyValidator := func(apiKey string) (string, error) {
		integration, err := slackIntegrationsService.GetSlackIntegrationBySecretKey(context.Background(), apiKey)
		if err != nil {
			return "", err
		}
		return integration.ID, nil
	}

	wsClient := clients.NewWebSocketClient(apiKeyValidator)

	coreUseCase := usecases.NewCoreUseCase(wsClient, agentsService, jobsService, slackIntegrationsService)
	wsHandler := handlers.NewWebSocketHandler(coreUseCase)
	slackHandler := handlers.NewSlackWebhooksHandler(cfg.SlackSigningSecret, coreUseCase, slackIntegrationsService)
	dashboardHandler := handlers.NewDashboardAPIHandler(usersService, slackIntegrationsService)
	dashboardHTTPHandler := handlers.NewDashboardHTTPHandler(dashboardHandler)
	authMiddleware := middleware.NewClerkAuthMiddleware(usersService, cfg.ClerkSecretKey)

	// Create a new router
	router := mux.NewRouter()

	// Setup endpoints with the new router
	wsClient.RegisterWithRouter(router)
	slackHandler.SetupEndpoints(router)
	dashboardHTTPHandler.SetupEndpoints(router, authMiddleware)

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			log.Printf("❌ Failed to write health check response: %v", err)
		}
	}).Methods("GET")

	// Register WebSocket hooks for agent lifecycle
	wsClient.RegisterConnectionHook(alertMiddleware.WrapConnectionHook(coreUseCase.RegisterAgent))
	wsClient.RegisterDisconnectionHook(alertMiddleware.WrapConnectionHook(coreUseCase.DeregisterAgent))
	wsClient.RegisterPingHook(alertMiddleware.WrapConnectionHook(coreUseCase.ProcessPing))

	// Register WebSocket message handler
	wsClient.RegisterMessageHandler(alertMiddleware.WrapMessageHandler(wsHandler.HandleMessage))

	// Start periodic broadcast of CheckIdleJobs, cleanup of inactive agents, and processing of queued jobs
	cleanupTicker := time.NewTicker(2 * time.Minute)
	go func() {
		for range cleanupTicker.C {
			_ = alertMiddleware.WrapBackgroundTask("ProcessQueuedJobs", coreUseCase.ProcessQueuedJobs)()
			_ = alertMiddleware.WrapBackgroundTask("BroadcastCheckIdleJobs", coreUseCase.BroadcastCheckIdleJobs)()
			_ = alertMiddleware.WrapBackgroundTask("CleanupInactiveAgents", coreUseCase.CleanupInactiveAgents)()
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
		Handler:           alertMiddleware.HTTPMiddleware(c.Handler(router)),
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
