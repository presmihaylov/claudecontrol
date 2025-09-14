package main

import (
	"context"
	"encoding/base64"
	"fmt"
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
	anthropic "ccbackend/clients/anthropic"
	discordclient "ccbackend/clients/discord"
	githubclient "ccbackend/clients/github"
	slackclient "ccbackend/clients/slack"
	socketioclient "ccbackend/clients/socketio"
	"ccbackend/clients/ssh"
	"ccbackend/config"
	"ccbackend/db"
	"ccbackend/handlers"
	"ccbackend/middleware"
	"ccbackend/salesnotif"
	"ccbackend/services"
	agentsservice "ccbackend/services/agents"
	anthropicintegrations "ccbackend/services/anthropic_integrations"
	ccagentcontainerintegrations "ccbackend/services/ccagent_container_integrations"
	discordintegrations "ccbackend/services/discord_integrations"
	discordmessages "ccbackend/services/discordmessages"
	githubintegrations "ccbackend/services/github_integrations"
	jobs "ccbackend/services/jobs"
	organizations "ccbackend/services/organizations"
	settingsservice "ccbackend/services/settings"
	slackintegrations "ccbackend/services/slack_integrations"
	slackmessages "ccbackend/services/slackmessages"
	"ccbackend/services/txmanager"
	"ccbackend/services/users"
	"ccbackend/usecases"
	"ccbackend/usecases/agents"
	"ccbackend/usecases/core"
	discordUseCase "ccbackend/usecases/discord"
	"ccbackend/usecases/slack"
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

	// Initialize sales notifications (only if Slack is configured)
	if cfg.SlackConfig.IsConfigured() {
		salesnotif.Init(cfg.SlackConfig.SalesWebhookURL, cfg.Environment)
	}

	// Initialize error alert middleware
	alertMiddleware := middleware.NewErrorAlertMiddleware(middleware.SlackAlertConfig{
		WebhookURL:  cfg.SlackConfig.AlertWebhookURL,
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
	processedDiscordMessagesRepo := db.NewPostgresProcessedDiscordMessagesRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	discordIntegrationsRepo := db.NewPostgresDiscordIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	githubIntegrationsRepo := db.NewPostgresGitHubIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	anthropicIntegrationsRepo := db.NewPostgresAnthropicIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	ccAgentContainerIntegrationsRepo := db.NewPostgresCCAgentContainerIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	settingsRepo := db.NewPostgresSettingsRepository(dbConn, cfg.DatabaseSchema)

	// Initialize transaction manager
	txManager := txmanager.NewTransactionManager(dbConn)

	// Core services (always needed)
	slackMessagesService := slackmessages.NewSlackMessagesService(processedSlackMessagesRepo)
	discordMessagesService := discordmessages.NewDiscordMessagesService(processedDiscordMessagesRepo)
	jobsService := jobs.NewJobsService(jobsRepo, slackMessagesService, discordMessagesService, txManager)
	organizationsService := organizations.NewOrganizationsService(organizationsRepo)
	usersService := users.NewUsersService(usersRepo, organizationsService, txManager)
	settingsService := settingsservice.NewSettingsService(settingsRepo)

	// Anthropic service (always needed for ccagent container service)
	anthropicClient := anthropic.NewAnthropicClient()
	anthropicService := anthropicintegrations.NewAnthropicIntegrationsService(
		anthropicIntegrationsRepo,
		anthropicClient,
	)

	// Initialize Slack components (optional)
	var slackIntegrationsService services.SlackIntegrationsService
	var slackUseCase usecases.SlackUseCaseInterface
	var slackHandler *handlers.SlackEventsHandler

	if cfg.SlackConfig.IsConfigured() {
		log.Printf("🔧 Initializing Slack components...")
		slackOAuthClient := slackclient.NewSlackOAuthClient()
		slackIntegrationsService = slackintegrations.NewSlackIntegrationsService(
			slackIntegrationsRepo,
			slackOAuthClient,
			cfg.SlackConfig.ClientID,
			cfg.SlackConfig.ClientSecret,
		)
	} else {
		log.Printf("⚠️ Slack not configured - Using unconfigured service")
		slackIntegrationsService = slackintegrations.NewUnconfiguredSlackIntegrationsService()
	}

	// Initialize Discord components (optional)
	var discordClient clients.DiscordClient
	var discordIntegrationsService services.DiscordIntegrationsService
	var discordUseCaseInstance usecases.DiscordUseCaseInterface
	var discordHandler *handlers.DiscordEventsHandler

	if cfg.DiscordConfig.IsConfigured() {
		log.Printf("🔧 Initializing Discord components...")
		var err error
		discordClient, err = discordclient.NewDiscordClient(&http.Client{}, cfg.DiscordConfig.BotToken)
		if err != nil {
			return fmt.Errorf("failed to create Discord client: %w", err)
		}

		discordIntegrationsService = discordintegrations.NewDiscordIntegrationsService(
			discordIntegrationsRepo,
			discordClient,
			cfg.DiscordConfig.ClientID,
			cfg.DiscordConfig.ClientSecret,
		)
	} else {
		log.Printf("⚠️ Discord not configured - Using unconfigured service")
		discordIntegrationsService = discordintegrations.NewUnconfiguredDiscordIntegrationsService()
	}

	// Initialize GitHub components (optional)
	var githubClient clients.GitHubClient
	var githubService services.GitHubIntegrationsService

	if cfg.GitHubConfig.IsConfigured() {
		log.Printf("🔧 Initializing GitHub components...")
		privateKey, err := base64.StdEncoding.DecodeString(cfg.GitHubConfig.AppPrivateKey)
		if err != nil {
			return fmt.Errorf("failed to decode GitHub private key: %w", err)
		}
		githubClient, err = githubclient.NewGitHubClient(
			cfg.GitHubConfig.ClientID,
			cfg.GitHubConfig.ClientSecret,
			cfg.GitHubConfig.AppID,
			privateKey,
		)
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}
		githubService = githubintegrations.NewGitHubIntegrationsService(githubIntegrationsRepo, githubClient)
	} else {
		log.Printf("⚠️ GitHub not configured - Using unconfigured service")
		githubService = githubintegrations.NewUnconfiguredGitHubIntegrationsService()
	}

	// Initialize SSH/Container components (optional)
	var sshClient ssh.SSHClientInterface
	var ccAgentContainerService services.CCAgentContainerIntegrationsService

	if cfg.SSHConfig.IsConfigured() {
		log.Printf("🔧 Initializing SSH/Container components...")
		sshClient = ssh.NewSSHClient(cfg.SSHConfig.PrivateKeyBase64)
		ccAgentContainerService = ccagentcontainerintegrations.NewCCAgentContainerIntegrationsService(
			ccAgentContainerIntegrationsRepo,
			cfg,
			githubService,
			anthropicService,
			organizationsService,
			sshClient,
		)
	} else {
		log.Printf("⚠️ SSH/Container not configured - Using unconfigured service")
		ccAgentContainerService = ccagentcontainerintegrations.NewUnconfiguredCCAgentContainerIntegrationsService()
	}

	// Create API key validator using organizationsService directly
	apiKeyValidator := func(apiKey string) (string, error) {
		maybeOrg, err := organizationsService.GetOrganizationBySecretKey(context.Background(), apiKey)
		if err != nil {
			return "", err
		}
		if !maybeOrg.IsPresent() {
			return "", fmt.Errorf("organization not found - invalid API key")
		}
		return maybeOrg.MustGet().ID, nil
	}

	wsClient := socketioclient.NewSocketIOClient(apiKeyValidator)

	// Create agents service after wsClient is available
	agentsService := agentsservice.NewAgentsService(agentsRepo, wsClient)

	// Create use cases in dependency order
	agentsUseCase := agents.NewAgentsUseCase(wsClient, agentsService)

	// Create Slack use case
	if cfg.SlackConfig.IsConfigured() {
		slackUseCase = slack.NewSlackUseCase(
			wsClient,
			agentsService,
			jobsService,
			slackMessagesService,
			slackIntegrationsService,
			txManager,
			agentsUseCase,
			slackclient.NewSlackClient,
		)
	} else {
		slackUseCase = slack.NewUnconfiguredSlackUseCase()
	}

	// Create Discord use case
	if cfg.DiscordConfig.IsConfigured() && discordClient != nil {
		discordUseCaseInstance = discordUseCase.NewDiscordUseCase(
			discordClient,
			wsClient,
			agentsService,
			jobsService,
			discordMessagesService,
			discordIntegrationsService,
			txManager,
			agentsUseCase,
		)
	} else {
		discordUseCaseInstance = discordUseCase.NewUnconfiguredDiscordUseCase()
	}

	// Create core use case with available components
	coreUseCase := core.NewCoreUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackIntegrationsService,
		organizationsService,
		slackUseCase,
		discordUseCaseInstance,
	)

	wsHandler := handlers.NewMessagesHandler(coreUseCase)

	// Create Slack handler if Slack is configured
	if cfg.SlackConfig.IsConfigured() {
		slackHandler = handlers.NewSlackEventsHandler(
			cfg.SlackConfig.SigningSecret,
			coreUseCase,
			slackIntegrationsService,
		)
	}

	// Create Discord handler if Discord is configured
	if cfg.DiscordConfig.IsConfigured() && discordClient != nil {
		var err error
		discordHandler, err = handlers.NewDiscordEventsHandler(
			cfg.DiscordConfig.BotToken,
			discordClient,
			coreUseCase,
			discordIntegrationsService,
			discordUseCaseInstance,
		)
		if err != nil {
			return fmt.Errorf("failed to create Discord events handler: %w", err)
		}
	}

	// Create dashboard handler with available services
	dashboardHandler := handlers.NewDashboardAPIHandler(
		usersService,
		slackIntegrationsService,
		discordIntegrationsService,
		githubService,
		anthropicService,
		ccAgentContainerService,
		organizationsService,
		agentsService,
		settingsService,
		txManager,
	)
	dashboardHTTPHandler := handlers.NewDashboardHTTPHandler(dashboardHandler)

	// Create authentication middleware if Clerk is configured
	var authMiddleware *middleware.ClerkAuthMiddleware
	if cfg.ClerkConfig.IsConfigured() {
		authMiddleware = middleware.NewClerkAuthMiddleware(
			usersService,
			organizationsService,
			cfg.ClerkConfig.SecretKey,
		)
	} else {
		log.Printf("⚠️ Clerk authentication not configured - Dashboard will be unauthenticated")
	}

	// Create a new router
	router := mux.NewRouter()

	// Setup endpoints with the new router
	wsClient.RegisterWithRouter(router)

	// Setup Slack endpoints if configured
	if slackHandler != nil {
		slackHandler.SetupEndpoints(router)
	}

	// Setup dashboard endpoints (with or without auth)
	if authMiddleware != nil {
		dashboardHTTPHandler.SetupEndpoints(router, authMiddleware)
	} else {
		// Setup public endpoints without authentication
		log.Printf("⚠️ Dashboard endpoints running without authentication - public access enabled")
		dashboardHTTPHandler.SetupPublicEndpoints(router)
	}

	// Start Discord bot if configured
	if discordHandler != nil {
		err = discordHandler.StartBot()
		if err != nil {
			return fmt.Errorf("failed to start Discord bot: %w", err)
		}
	}

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

	return handleGracefulShutdown(server, discordHandler)
}

func handleGracefulShutdown(server *http.Server, discordHandler *handlers.DiscordEventsHandler) error {
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

	// Stop Discord bot if it was started
	if discordHandler != nil {
		discordHandler.StopBot()
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
