// Example showing minimal changes to main.go for middleware integration
// Only the changed sections are shown - most of main.go stays the same

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// ... existing database and service setup code stays the same ...

	// NEW: Initialize error alert middleware (only addition needed)
	alertMiddleware := middleware.NewErrorAlertMiddleware(middleware.SlackAlertConfig{
		WebhookURL:  cfg.SlackAlertWebhookURL,
		Environment: cfg.Environment,
		AppName:     "ccbackend",
		LogsURL:     cfg.LogsURL,
	})

	// ... existing service setup continues ...

	// Create a new router
	router := mux.NewRouter()

	// Setup endpoints with the new router
	wsClient.RegisterWithRouter(router)
	slackHandler.SetupEndpoints(router)
	dashboardHandler.SetupEndpoints(router, authMiddleware)

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// ... same health check code ...
	}).Methods("GET")

	// Register WebSocket hooks - WRAP existing hooks with middleware
	wsClient.RegisterConnectionHook(alertMiddleware.WrapConnectionHook(coreUseCase.RegisterAgent))
	wsClient.RegisterDisconnectionHook(alertMiddleware.WrapConnectionHook(coreUseCase.DeregisterAgent))
	wsClient.RegisterPingHook(alertMiddleware.WrapConnectionHook(coreUseCase.ProcessPing))

	// Register WebSocket message handler - WRAP existing handler
	wsClient.RegisterMessageHandler(alertMiddleware.WrapMessageHandler(wsHandler.HandleMessage))

	// Background tasks - WRAP existing tasks
	cleanupTicker := time.NewTicker(2 * time.Minute)
	go func() {
		for range cleanupTicker.C {
			// WRAP each background task
			alertMiddleware.WrapBackgroundTask("ProcessQueuedJobs", coreUseCase.ProcessQueuedJobs)()
			alertMiddleware.WrapBackgroundTask("BroadcastCheckIdleJobs", coreUseCase.BroadcastCheckIdleJobs)()
			alertMiddleware.WrapBackgroundTask("CleanupInactiveAgents", coreUseCase.CleanupInactiveAgents)()
		}
	}()
	defer cleanupTicker.Stop()

	// Setup CORS middleware
	// ... existing CORS setup ...

	// Setup and handle graceful shutdown - WRAP router with HTTP middleware
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           alertMiddleware.HTTPMiddleware(c.Handler(router)), // WRAP HTTP layer
		ReadHeaderTimeout: 30 * time.Second,
	}

	return handleGracefulShutdown(server)
}