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

	"ccbackend/clients"
	"ccbackend/config"
	"ccbackend/db"
	"ccbackend/handlers"
	"ccbackend/middleware"
	"ccbackend/services"
	"ccbackend/usecases"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
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
	
	// Create API key validator for WebSocket connections (needed for SlackIntegrationsService creation)
	var wsClient *clients.WebSocketClient
	apiKeyValidator := func(apiKey string) (string, error) {
		// This is a temporary validator for bootstrapping - will be updated after SlackIntegrationsService is created
		return "", nil
	}
	wsClient = clients.NewWebSocketClient(apiKeyValidator)
	
	slackIntegrationsService := services.NewSlackIntegrationsService(slackIntegrationsRepo, slackOAuthClient, cfg.SlackClientID, cfg.SlackClientSecret, agentsService, wsClient)

	// Clear all active agents on startup
	log.Printf("🧹 Cleaning up stale active agents from previous server runs")
	if err := agentsService.DeleteAllActiveAgents(); err != nil {
		log.Printf("⚠️ Failed to clear stale active agents: %v", err)
	}

	// Update the API key validator now that SlackIntegrationsService is available
	actualApiKeyValidator := func(apiKey string) (string, error) {
		integration, err := slackIntegrationsService.GetSlackIntegrationBySecretKey(apiKey)
		if err != nil {
			return "", err
		}
		return integration.ID.String(), nil
	}
	
	// Replace the temporary validator with the actual one
	wsClient = clients.NewWebSocketClient(actualApiKeyValidator)
	
	coreUseCase := usecases.NewCoreUseCase(wsClient, agentsService, jobsService, slackIntegrationsService)
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
		Addr:    ":" + cfg.Port,
		Handler: c.Handler(router),
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

