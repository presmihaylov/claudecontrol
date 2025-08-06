package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/jessevdk/go-flags"
	"github.com/zishang520/engine.io-client-go/transports"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-client-go/socket"

	"ccagent/clients"
	"ccagent/core"
	"ccagent/core/log"
	"ccagent/handlers"
	"ccagent/models"
	"ccagent/services"
	"ccagent/usecases"
	"ccagent/utils"
)

type CmdRunner struct {
	messageHandler   *handlers.MessageHandler
	gitUseCase       *usecases.GitUseCase
	logFilePath      string
	agentID          string
	reconnectChan    chan struct{}
	connectionCtx    context.Context
	connectionCancel context.CancelFunc
}

func NewCmdRunner(permissionMode string) (*CmdRunner, error) {
	log.Info("📋 Starting to initialize CmdRunner")
	claudeClient := clients.NewClaudeClient(permissionMode)
	claudeService := services.NewClaudeService(claudeClient)
	gitClient := clients.NewGitClient()
	appState := models.NewAppState()
	gitUseCase := usecases.NewGitUseCase(gitClient, claudeService, appState)
	messageHandler := handlers.NewMessageHandler(claudeService, gitUseCase, appState)

	agentID := core.NewID("ccaid")
	log.Info("🆔 Using persistent agent ID: %s", agentID)

	// Initialize connection context
	connectionCtx, connectionCancel := context.WithCancel(context.Background())

	log.Info("📋 Completed successfully - initialized CmdRunner with all services")
	return &CmdRunner{
		messageHandler:   messageHandler,
		gitUseCase:       gitUseCase,
		agentID:          agentID,
		reconnectChan:    make(chan struct{}, 1),
		connectionCtx:    connectionCtx,
		connectionCancel: connectionCancel,
	}, nil
}

type Options struct {
	BypassPermissions bool `long:"bypassPermissions" description:"Use bypassPermissions mode for Claude (WARNING: Only use in controlled sandbox environments)"`
}

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Always enable info level logging
	log.SetLevel(slog.LevelInfo)

	// Acquire directory lock to prevent multiple instances in same directory
	dirLock, err := utils.NewDirLock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory lock: %v\n", err)
		os.Exit(1)
	}

	if err := dirLock.TryLock(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Ensure lock is released on program exit
	defer func() {
		if unlockErr := dirLock.Unlock(); unlockErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to release directory lock: %v\n", unlockErr)
		}
	}()

	// Validate CCAGENT_API_KEY environment variable
	ccagentAPIKey := os.Getenv("CCAGENT_API_KEY")
	if ccagentAPIKey == "" {
		fmt.Fprintf(os.Stderr, "Error: CCAGENT_API_KEY environment variable is required but not set\n")
		os.Exit(1)
	}

	// Determine permission mode based on flag
	permissionMode := "acceptEdits"
	if opts.BypassPermissions {
		permissionMode = "bypassPermissions"
		fmt.Fprintf(os.Stderr, "Warning: --bypassPermissions flag should only be used in a controlled, sandbox environment. Otherwise, anyone from Slack will have access to your entire system\n")
	}

	cmdRunner, err := NewCmdRunner(permissionMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing CmdRunner: %v\n", err)
		os.Exit(1)
	}

	// Setup program-wide logging from start
	logFilePath, err := cmdRunner.setupProgramLogging()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up program logging: %v\n", err)
		os.Exit(1)
	}
	cmdRunner.logFilePath = logFilePath

	// Validate Git environment before starting
	err = cmdRunner.gitUseCase.ValidateGitEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Git environment validation failed: %v\n", err)
		os.Exit(1)
	}

	// Cleanup stale ccagent branches
	err = cmdRunner.gitUseCase.CleanupStaleBranches()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup stale branches: %v\n", err)
		// Don't exit - this is not critical for agent operation
	}

	// Get WebSocket URL from environment variable with default fallback
	wsURL := os.Getenv("CCAGENT_WS_API_URL")
	if wsURL == "" {
		wsURL = "https://claudecontrol.onrender.com/socketio/"
	}

	// Set up deferred cleanup
	defer func() {
		fmt.Fprintf(os.Stderr, "\n📝 App execution finished, logs for this session are stored in %s\n", cmdRunner.logFilePath)
	}()

	// Start Socket.IO client
	err = cmdRunner.startSocketIOClient(wsURL, ccagentAPIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting WebSocket client: %v\n", err)
		os.Exit(1)
	}
}

func (cr *CmdRunner) startSocketIOClient(serverURLStr, apiKey string) error {
	log.Info("📋 Starting to connect to Socket.IO server at %s", serverURLStr)

	// Set up global interrupt handling
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Set up Socket.IO client options
	opts := socket.DefaultOptions()
	opts.SetTransports(types.NewSet(transports.Polling, transports.WebSocket))

	// Set authentication headers
	opts.SetExtraHeaders(map[string][]string{
		"X-CCAGENT-API-KEY": {apiKey},
		"X-CCAGENT-ID":      {cr.agentID},
	})

	manager := socket.NewManager(serverURLStr, opts)
	socketClient := manager.Socket("/", opts)

	// Initialize worker pool with 1 worker for sequential processing
	wp := workerpool.New(1)
	defer wp.StopWait()

	// Connection event handlers
	err := socketClient.On("connect", func(args ...any) {
		log.Info("✅ Connected to Socket.IO server, socket ID: %s", socketClient.Id())
		// Reset connection context on successful connection
		if cr.connectionCancel != nil {
			cr.connectionCancel()
		}
		cr.connectionCtx, cr.connectionCancel = context.WithCancel(context.Background())
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up connect handler: %v", err))

	err = socketClient.On("connect_error", func(args ...any) {
		log.Info("❌ Socket.IO connection error: %v", args)
		// Cancel connection context on connection error
		if cr.connectionCancel != nil {
			cr.connectionCancel()
		}
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up connect_error handler: %v", err))

	err = socketClient.On("disconnect", func(args ...any) {
		log.Info("🔌 Socket.IO disconnected: %v", args)
		// Cancel connection context on disconnect
		if cr.connectionCancel != nil {
			cr.connectionCancel()
		}
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up disconnect handler: %v", err))

	// Set up message handler for cc_message event
	err = socketClient.On("cc_message", func(data ...any) {
		if len(data) == 0 {
			log.Info("❌ No data received for cc_message event")
			return
		}

		var msg models.BaseMessage
		msgBytes, err := json.Marshal(data[0])
		if err != nil {
			log.Info("❌ Failed to marshal message data: %v", err)
			return
		}

		err = json.Unmarshal(msgBytes, &msg)
		if err != nil {
			log.Info("❌ Failed to unmarshal message data: %v", err)
			return
		}

		log.Info("📨 Received message type: %s", msg.Type)
		wp.Submit(func() {
			// Check if connection is still active before processing
			select {
			case <-cr.connectionCtx.Done():
				log.Info("🔌 Connection lost - skipping message processing for type: %s", msg.Type)
				return
			default:
				// Connection is active, proceed with processing
			}

			// Pass connection context to message handler for context-aware processing
			cr.messageHandler.HandleMessageWithContext(cr.connectionCtx, msg, socketClient)
		})
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up cc_message handler: %v", err))

	// Built-in reconnection handlers
	err = manager.On("reconnect", func(...any) {
		log.Info("✅ Reconnected to Socket.IO server")
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up reconnect handler: %v", err))

	err = manager.On("reconnect_error", func(errs ...any) {
		log.Info("❌ Socket.IO reconnection error: %v", errs)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up reconnect_error handler: %v", err))

	err = manager.On("reconnect_failed", func(errs ...any) {
		log.Info("❌ Socket.IO reconnection failed: %v", errs)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up reconnect_failed handler: %v", err))

	// Start ping routine once connected
	pingCtx, pingCancel := context.WithCancel(context.Background())
	defer pingCancel()
	cr.startPingRoutine(pingCtx, socketClient)

	// Wait for interrupt signal
	<-interrupt
	log.Info("🔌 Interrupt received, closing Socket.IO connection...")

	socketClient.Disconnect()
	return nil
}

func (cr *CmdRunner) setupProgramLogging() (string, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create ~/.config/ccagent/logs directory if it doesn't exist
	logsDir := filepath.Join(homeDir, ".config", "ccagent", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with timestamp only
	timestamp := time.Now().Format("20060102-150405")
	logFileName := fmt.Sprintf("%s.log", timestamp)
	logFilePath := filepath.Join(logsDir, logFileName)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create log file: %w", err)
	}

	// Always write to both stdout and file
	writer := io.MultiWriter(os.Stdout, logFile)
	log.SetWriter(writer)

	return logFilePath, nil
}

func (cr *CmdRunner) startPingRoutine(ctx context.Context, socketClient *socket.Socket) {
	log.Info("📋 Starting ping routine")
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("📋 Ping routine stopped")
				return
			case <-ticker.C:
				log.Info("💓 Sending ping to server")
				if err := socketClient.Emit("ping"); err != nil {
					log.Error("❌ Failed to send ping: %v", err)
				}
			}
		}
	}()
}
