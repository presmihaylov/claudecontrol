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
	discordclient "ccbackend/clients/discord"
	slackclient "ccbackend/clients/slack"
	socketioclient "ccbackend/clients/socketio"
	"ccbackend/config"
	"ccbackend/db"
	"ccbackend/handlers"
	"ccbackend/middleware"
	agentsservice "ccbackend/services/agents"
	discordintegrations "ccbackend/services/discord_integrations"
	jobs "ccbackend/services/jobs"
	organizations "ccbackend/services/organizations"
	slackintegrations "ccbackend/services/slack_integrations"
	"ccbackend/services/txmanager"
	"ccbackend/services/users"
	"ccbackend/usecases/agents"
	"ccbackend/usecases/core"
	"ccbackend/usecases/slack"
	"ccbackend/utils"
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
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	discordIntegrationsRepo := db.NewPostgresDiscordIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Initialize transaction manager
	txManager := txmanager.NewTransactionManager(dbConn)

	agentsService := agentsservice.NewAgentsService(agentsRepo)
	jobsService := jobs.NewJobsService(jobsRepo, processedSlackMessagesRepo, txManager)
	organizationsService := organizations.NewOrganizationsService(organizationsRepo)
	usersService := users.NewUsersService(usersRepo, organizationsService, txManager)
	slackOAuthClient := slackclient.NewSlackOAuthClient()
	slackIntegrationsService := slackintegrations.NewSlackIntegrationsService(
		slackIntegrationsRepo,
		slackOAuthClient,
		cfg.SlackClientID,
		cfg.SlackClientSecret,
	)
	discordClient, err := discordclient.NewDiscordClient(&http.Client{}, cfg.DiscordBotToken)
	utils.AssertInvariant(err == nil, "Failed to create Discord client")

	discordIntegrationsService := discordintegrations.NewDiscordIntegrationsService(
		discordIntegrationsRepo,
		discordClient,
		cfg.DiscordClientID,
		cfg.DiscordClientSecret,
	)

	// Create API key validator using organizationsService directly
	apiKeyValidator := func(apiKey string) (string, error) {
		maybeOrg, err := organizationsService.GetOrganizationBySecretKey(context.Background(), apiKey)
		if err != nil {
			return "", err
		}
		if !maybeOrg.IsPresent() {
			return "", err
		}
		return maybeOrg.MustGet().ID, nil
	}

	wsClient := socketioclient.NewSocketIOClient(apiKeyValidator)

	// Create use cases in dependency order
	agentsUseCase := agents.NewAgentsUseCase(wsClient, agentsService)

	slackUseCase := slack.NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	coreUseCase := core.NewCoreUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackIntegrationsService,
		organizationsService,
		slackUseCase,
	)
	wsHandler := handlers.NewMessagesHandler(coreUseCase)
	slackHandler := handlers.NewSlackEventsHandler(cfg.SlackSigningSecret, coreUseCase, slackIntegrationsService)
	dashboardHandler := handlers.NewDashboardAPIHandler(
		usersService,
		slackIntegrationsService,
		discordIntegrationsService,
		organizationsService,
	)
	dashboardHTTPHandler := handlers.NewDashboardHTTPHandler(dashboardHandler)
	authMiddleware := middleware.NewClerkAuthMiddleware(usersService, organizationsService, cfg.ClerkSecretKey)

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

	// Create wrapper functions for usecase methods that now require context
	registerAgent := func(client *clients.Client) error {
		return coreUseCase.RegisterAgent(context.Background(), client)
	}
	deregisterAgent := func(client *clients.Client) error {
		return coreUseCase.DeregisterAgent(context.Background(), client)
	}
	processPing := func(client *clients.Client) error {
		return coreUseCase.ProcessPing(context.Background(), client)
	}

	// Register WebSocket hooks for agent lifecycle
	wsClient.RegisterConnectionHook(alertMiddleware.WrapConnectionHook(registerAgent))
	wsClient.RegisterDisconnectionHook(alertMiddleware.WrapConnectionHook(deregisterAgent))
	wsClient.RegisterPingHook(alertMiddleware.WrapConnectionHook(processPing))

	// Register WebSocket message handler (middleware consumes errors internally)
	wrappedHandler := alertMiddleware.WrapMessageHandler(wsHandler.HandleMessage)
	messageHandlerAdapter := func(client *clients.Client, msg any) error {
		wrappedHandler(client, msg)
		return nil // Middleware handles errors internally
	}
	wsClient.RegisterMessageHandler(messageHandlerAdapter)

	// Start periodic broadcast of CheckIdleJobs, cleanup of inactive agents, and processing of queued jobs
	cleanupTicker := time.NewTicker(1 * time.Minute)
	go func() {
		for range cleanupTicker.C {
			_ = alertMiddleware.WrapBackgroundTask("ProcessQueuedJobs", func() error {
				return coreUseCase.ProcessQueuedJobs(context.Background())
			})()
			_ = alertMiddleware.WrapBackgroundTask("BroadcastCheckIdleJobs", func() error {
				return coreUseCase.BroadcastCheckIdleJobs(context.Background())
			})()
			_ = alertMiddleware.WrapBackgroundTask("CleanupInactiveAgents", func() error {
				return coreUseCase.CleanupInactiveAgents(context.Background())
			})()
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
