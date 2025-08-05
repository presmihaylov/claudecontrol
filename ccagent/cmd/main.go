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
	"strings"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/google/uuid"
	"github.com/jessevdk/go-flags"
	"github.com/zishang520/engine.io-client-go/transports"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-client-go/socket"

	"ccagent/clients"
	"ccagent/core/log"
	"ccagent/models"
	"ccagent/services"
	"ccagent/usecases"
	"ccagent/utils"
)

type CmdRunner struct {
	sessionService *services.SessionService
	claudeService  *services.ClaudeService
	gitUseCase     *usecases.GitUseCase
	appState       *models.AppState
	logFilePath    string
	agentID        uuid.UUID
	reconnectChan  chan struct{}
}

func NewCmdRunner(permissionMode string) (*CmdRunner, error) {
	log.Info("📋 Starting to initialize CmdRunner")
	sessionService := services.NewSessionService()
	claudeClient := clients.NewClaudeClient(permissionMode)
	claudeService := services.NewClaudeService(claudeClient)
	gitClient := clients.NewGitClient()
	appState := models.NewAppState()
	gitUseCase := usecases.NewGitUseCase(gitClient, claudeService, appState)

	agentID := uuid.New()
	log.Info("🆔 Using persistent agent ID: %s", agentID)

	log.Info("📋 Completed successfully - initialized CmdRunner with all services")
	return &CmdRunner{
		sessionService: sessionService,
		claudeService:  claudeService,
		gitUseCase:     gitUseCase,
		appState:       appState,
		agentID:        agentID,
		reconnectChan:  make(chan struct{}, 1),
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
		"X-CCAGENT-ID":      {cr.agentID.String()},
	})

	manager := socket.NewManager(serverURLStr, opts)
	socketClient := manager.Socket("/", opts)

	// Initialize worker pool with 1 worker for sequential processing
	wp := workerpool.New(1)
	defer wp.StopWait()

	// Connection event handlers
	err := socketClient.On("connect", func(args ...any) {
		log.Info("✅ Connected to Socket.IO server, socket ID: %s", socketClient.Id())
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up connect handler: %v", err))

	err = socketClient.On("connect_error", func(args ...any) {
		log.Info("❌ Socket.IO connection error: %v", args)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up connect_error handler: %v", err))

	err = socketClient.On("disconnect", func(args ...any) {
		log.Info("🔌 Socket.IO disconnected: %v", args)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up disconnect handler: %v", err))

	// Set up message handler for cc_message event
	err = socketClient.On("cc_message", func(data ...any) {
		if len(data) == 0 {
			log.Info("❌ No data received for cc_message event")
			return
		}

		var msg models.UnknownMessage
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
			cr.handleMessage(msg, socketClient)
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
		ticker := time.NewTicker(30 * time.Second)
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

func (cr *CmdRunner) handleMessage(msg models.UnknownMessage, socketClient *socket.Socket) {
	switch msg.Type {
	case models.MessageTypeStartConversation:
		if err := cr.handleStartConversation(msg, socketClient); err != nil {
			// Extract SlackMessageID from payload for error reporting
			var payload models.StartConversationPayload
			slackMessageID := ""
			if unmarshalErr := unmarshalPayload(msg.Payload, &payload); unmarshalErr == nil {
				slackMessageID = payload.SlackMessageID
			}
			if sendErr := cr.sendErrorMessage(socketClient, err, slackMessageID); sendErr != nil {
				log.Error("Failed to send error message: %v", sendErr)
			}
		}
	case models.MessageTypeUserMessage:
		if err := cr.handleUserMessage(msg, socketClient); err != nil {
			// Extract SlackMessageID from payload for error reporting
			var payload models.UserMessagePayload
			slackMessageID := ""
			if unmarshalErr := unmarshalPayload(msg.Payload, &payload); unmarshalErr == nil {
				slackMessageID = payload.SlackMessageID
			}
			if sendErr := cr.sendErrorMessage(socketClient, err, slackMessageID); sendErr != nil {
				log.Error("Failed to send error message: %v", sendErr)
			}
		}
	case models.MessageTypeJobUnassigned:
		if err := cr.handleJobUnassigned(msg, socketClient); err != nil {
			log.Info("❌ Error handling JobUnassigned message: %v", err)
		}
	case models.MessageTypeCheckIdleJobs:
		if err := cr.handleCheckIdleJobs(msg, socketClient); err != nil {
			log.Info("❌ Error handling CheckIdleJobs message: %v", err)
		}
	default:
		log.Info("⚠️ Unhandled message type: %s", msg.Type)
	}
}

func (cr *CmdRunner) handleStartConversation(msg models.UnknownMessage, socketClient *socket.Socket) error {
	log.Info("📋 Starting to handle start conversation message")
	var payload models.StartConversationPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal start conversation payload: %v", err)
		return fmt.Errorf("failed to unmarshal start conversation payload: %w", err)
	}

	// Send processing slack message notification that agent is starting to process
	if err := cr.sendProcessingSlackMessage(socketClient, payload.SlackMessageID); err != nil {
		log.Info("❌ Failed to send processing slack message notification: %v", err)
		return fmt.Errorf("failed to send processing slack message notification: %w", err)
	}

	log.Info("🚀 Starting new conversation with message: %s", payload.Message)

	// Prepare Git environment for new conversation - FAIL if this doesn't work
	branchName, err := cr.gitUseCase.PrepareForNewConversation(payload.Message)
	if err != nil {
		log.Error("❌ Failed to prepare Git environment: %v", err)
		return fmt.Errorf("failed to prepare Git environment: %w", err)
	}

	behaviourInstructions := `You are a claude code instance which will be referred to by the user as "Claude Control" for this session. When someone calls you claude control, they refer to you.

You are being interacted with over Slack (the software). I want you to adjust your responses to account for this. In particular:
- CRITICAL: Keep ALL responses under 800 characters maximum - this is a hard limit for Slack readability
- Focus on high-level summaries and avoid implementation details unless specifically requested
- Structure your responses in sections split via bold text instead of using markdown headings because slack doesnt support markdown headings
- Use the following Markdown formatting rules for all of your responses:
	- Bold text: *example*
	- Italic text: _example_
	- Strikethrough text: ~example~
	- Block quotes: > example
	- Bulleted list - - one\n - two\n - three
	- Inline code blocks: ` + "`example`" + `
	- Full code blocks: ` + "```example```" + `
- Do not use any other sort of markdown format except the ones listed above
- Do not show any specific language in code blocks because Slack doesn't support syntax highlighting
    - This is incorrect - ` + "```python\nexample\n```" + `
    - This is correct - ` + "```\nexample\n```" + `
- Especially be careful when using bold text:
    - This is incorrect - **example**
	- This is correct - *example*
- Use emojis liberally to draw attention to the relevant pieces of your message that are most important
- Be more explicit about errors and failures with clear emoji indicators
- Use clear file paths with line numbers for easy navigation

IMPORTANT: If you are editing a pull request description, never include or override the "Generated with [Claude Control](https://claudecontrol.com) from this [slack thread]" footer. The system will add this footer automatically. Do not include any "Generated with Claude Code" or similar footer text in PR descriptions.
`

	claudeResult, err := cr.claudeService.StartNewConversationWithSystemPrompt(payload.Message, behaviourInstructions)
	if err != nil {
		log.Info("❌ Error starting Claude session: %v", err)
		systemErr := cr.sendSystemMessage(socketClient, fmt.Sprintf("ccagent encountered error: %v", err), payload.SlackMessageID)
		if systemErr != nil {
			log.Error("❌ Failed to send system message for Claude error: %v", systemErr)
		}
		return fmt.Errorf("error starting Claude session: %w", err)
	}

	// Auto-commit changes if needed
	commitResult, err := cr.gitUseCase.AutoCommitChangesIfNeeded(payload.SlackMessageLink)
	if err != nil {
		log.Info("❌ Auto-commit failed: %v", err)
		return fmt.Errorf("auto-commit failed: %w", err)
	}

	// Update JobData with conversation info (use commitResult.BranchName if available, otherwise branchName)
	finalBranchName := branchName
	if commitResult != nil && commitResult.BranchName != "" {
		finalBranchName = commitResult.BranchName
	}

	// Extract PR ID from commit result if available
	prID := ""
	if commitResult != nil && commitResult.PullRequestID != "" {
		prID = commitResult.PullRequestID
	}

	cr.appState.UpdateJobData(payload.JobID, models.JobData{
		JobID:           payload.JobID,
		BranchName:      finalBranchName,
		ClaudeSessionID: claudeResult.SessionID,
		PullRequestID:   prID,
		UpdatedAt:       time.Now(),
	})

	// Send assistant response back first
	assistantPayload := models.AssistantMessagePayload{
		JobID:          payload.JobID,
		Message:        claudeResult.Output,
		SlackMessageID: payload.SlackMessageID,
	}

	assistantMsg := models.UnknownMessage{
		ID:      uuid.New().String(),
		Type:    models.MessageTypeAssistantMessage,
		Payload: assistantPayload,
	}
	if err := socketClient.Emit("cc_message", assistantMsg); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}

	log.Info("🤖 Sent assistant response (message ID: %s)", assistantMsg.ID)

	// Send system message after assistant message for git activity
	if err := cr.sendGitActivitySystemMessage(socketClient, commitResult, payload.SlackMessageID); err != nil {
		log.Info("❌ Failed to send git activity system message: %v", err)
		return fmt.Errorf("failed to send git activity system message: %w", err)
	}

	// Validate and restore PR description footer if needed
	if err := cr.gitUseCase.ValidateAndRestorePRDescriptionFooter(payload.SlackMessageLink); err != nil {
		log.Info("❌ Failed to validate PR description footer: %v", err)
		return fmt.Errorf("failed to validate PR description footer: %w", err)
	}

	log.Info("📋 Completed successfully - handled start conversation message")
	return nil
}

func (cr *CmdRunner) handleUserMessage(msg models.UnknownMessage, socketClient *socket.Socket) error {
	log.Info("📋 Starting to handle user message")
	var payload models.UserMessagePayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal user message payload: %v", err)
		return fmt.Errorf("failed to unmarshal user message payload: %w", err)
	}

	// Send processing slack message notification that agent is starting to process
	if err := cr.sendProcessingSlackMessage(socketClient, payload.SlackMessageID); err != nil {
		log.Info("❌ Failed to send processing slack message notification: %v", err)
		return fmt.Errorf("failed to send processing slack message notification: %w", err)
	}

	log.Info("💬 Continuing conversation with message: %s", payload.Message)

	// Get the current job data to retrieve the Claude session ID and branch
	jobData, exists := cr.appState.GetJobData(payload.JobID)
	if !exists {
		log.Info("❌ JobID %s not found in AppState", payload.JobID)
		return fmt.Errorf("job %s not found - conversation may have been started elsewhere", payload.JobID)
	}

	sessionID := jobData.ClaudeSessionID
	if sessionID == "" {
		log.Info("❌ No Claude session ID found for job %s", payload.JobID)
		return fmt.Errorf("no active Claude session found for job %s", payload.JobID)
	}

	// Assert that BranchName is never empty
	utils.AssertInvariant(jobData.BranchName != "", "BranchName must not be empty for job "+payload.JobID)

	// Switch to the job's branch before continuing the conversation
	if err := cr.gitUseCase.SwitchToJobBranch(jobData.BranchName); err != nil {
		log.Error("❌ Failed to switch to job branch %s: %v", jobData.BranchName, err)
		return fmt.Errorf("failed to switch to job branch %s: %w", jobData.BranchName, err)
	}
	log.Info("✅ Successfully switched to job branch: %s", jobData.BranchName)

	claudeResult, err := cr.claudeService.ContinueConversation(sessionID, payload.Message)
	if err != nil {
		log.Info("❌ Error continuing Claude session: %v", err)
		systemErr := cr.sendSystemMessage(socketClient, fmt.Sprintf("ccagent encountered error: %v", err), payload.SlackMessageID)
		if systemErr != nil {
			log.Error("❌ Failed to send system message for Claude error: %v", systemErr)
		}
		return fmt.Errorf("error continuing Claude session: %w", err)
	}

	// Auto-commit changes if needed
	commitResult, err := cr.gitUseCase.AutoCommitChangesIfNeeded(payload.SlackMessageLink)
	if err != nil {
		log.Info("❌ Auto-commit failed: %v", err)
		return fmt.Errorf("auto-commit failed: %w", err)
	}

	// Update JobData with latest session ID and branch name from commit result
	finalBranchName := jobData.BranchName
	if commitResult != nil && commitResult.BranchName != "" {
		finalBranchName = commitResult.BranchName
	}

	// Extract PR ID from existing job data or commit result
	prID := jobData.PullRequestID
	if commitResult != nil && commitResult.PullRequestID != "" {
		prID = commitResult.PullRequestID
	}

	cr.appState.UpdateJobData(payload.JobID, models.JobData{
		JobID:           payload.JobID,
		BranchName:      finalBranchName,
		ClaudeSessionID: claudeResult.SessionID,
		PullRequestID:   prID,
		UpdatedAt:       time.Now(),
	})

	// Send assistant response back first
	assistantPayload := models.AssistantMessagePayload{
		JobID:          payload.JobID,
		Message:        claudeResult.Output,
		SlackMessageID: payload.SlackMessageID,
	}

	assistantMsg := models.UnknownMessage{
		ID:      uuid.New().String(),
		Type:    models.MessageTypeAssistantMessage,
		Payload: assistantPayload,
	}
	if err := socketClient.Emit("cc_message", assistantMsg); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}

	log.Info("🤖 Sent assistant response (message ID: %s)", assistantMsg.ID)

	// Send system message after assistant message for git activity
	if err := cr.sendGitActivitySystemMessage(socketClient, commitResult, payload.SlackMessageID); err != nil {
		log.Info("❌ Failed to send git activity system message: %v", err)
		return fmt.Errorf("failed to send git activity system message: %w", err)
	}

	// Validate and restore PR description footer if needed
	if err := cr.gitUseCase.ValidateAndRestorePRDescriptionFooter(payload.SlackMessageLink); err != nil {
		log.Info("❌ Failed to validate PR description footer: %v", err)
		return fmt.Errorf("failed to validate PR description footer: %w", err)
	}

	log.Info("📋 Completed successfully - handled user message")
	return nil
}

func (cr *CmdRunner) handleJobUnassigned(msg models.UnknownMessage, _ *socket.Socket) error {
	log.Info("📋 Starting to handle job unassigned message")
	var payload models.JobUnassignedPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal job unassigned payload: %v", err)
		return fmt.Errorf("failed to unmarshal job unassigned payload: %w", err)
	}

	log.Info("🚫 Job has been unassigned from this agent")
	log.Info("📋 Completed successfully - handled job unassigned message")
	return nil
}

func (cr *CmdRunner) handleCheckIdleJobs(msg models.UnknownMessage, socketClient *socket.Socket) error {
	log.Info("📋 Starting to handle check idle jobs message")
	var payload models.CheckIdleJobsPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal check idle jobs payload: %v", err)
		return fmt.Errorf("failed to unmarshal check idle jobs payload: %w", err)
	}

	log.Info("🔍 Checking all assigned jobs for idleness")

	// Get all job data from app state
	allJobData := cr.appState.GetAllJobs()
	if len(allJobData) == 0 {
		log.Info("📋 No jobs assigned to this agent")
		return nil
	}

	log.Info("🔍 Found %d jobs assigned to this agent", len(allJobData))

	// Check each job for idleness
	for jobID, jobData := range allJobData {
		log.Info("🔍 Checking job %s on branch %s", jobID, jobData.BranchName)

		if err := cr.checkJobIdleness(jobID, jobData, socketClient); err != nil {
			log.Info("❌ Failed to check idleness for job %s: %v", jobID, err)
			// Continue checking other jobs even if one fails
			continue
		}
	}

	log.Info("📋 Completed successfully - checked all jobs for idleness")
	return nil
}

func (cr *CmdRunner) checkJobIdleness(jobID string, jobData models.JobData, socketClient *socket.Socket) error {
	log.Info("📋 Starting to check idleness for job %s", jobID)

	// Switch to the job's branch to check PR status
	if err := cr.gitUseCase.SwitchToJobBranch(jobData.BranchName); err != nil {
		log.Error("❌ Failed to switch to job branch %s: %v", jobData.BranchName, err)
		return fmt.Errorf("failed to switch to job branch %s: %w", jobData.BranchName, err)
	}

	var prStatus string
	var err error

	// Use stored PR ID if available, otherwise fall back to branch-based check
	if jobData.PullRequestID != "" {
		log.Info("ℹ️ Using stored PR ID %s for job %s", jobData.PullRequestID, jobID)
		prStatus, err = cr.gitUseCase.CheckPRStatusByID(jobData.PullRequestID)
		if err != nil {
			log.Error("❌ Failed to check PR status by ID %s: %v", jobData.PullRequestID, err)
			return fmt.Errorf("failed to check PR status by ID %s: %w", jobData.PullRequestID, err)
		}
	} else {
		log.Info("ℹ️ No stored PR ID for job %s, using branch-based check", jobID)
		prStatus, err = cr.gitUseCase.CheckPRStatus(jobData.BranchName)
		if err != nil {
			log.Error("❌ Failed to check PR status for branch %s: %v", jobData.BranchName, err)
			return fmt.Errorf("failed to check PR status for branch %s: %w", jobData.BranchName, err)
		}
	}

	var reason string
	var shouldComplete bool

	switch prStatus {
	case "merged":
		reason = "Job complete - Pull request was merged"
		shouldComplete = true
		log.Info("✅ Job %s PR was merged - marking as complete", jobID)
	case "closed":
		reason = "Job complete - Pull request was closed"
		shouldComplete = true
		log.Info("✅ Job %s PR was closed - marking as complete", jobID)
	case "open":
		log.Info("ℹ️ Job %s has open PR - not marking as complete", jobID)
		shouldComplete = false
	case "no_pr":
		log.Info("ℹ️ Job %s has no PR - checking timeout", jobID)
		jobData, exists := cr.appState.GetJobData(jobID)
		if !exists {
			log.Info("❌ Job %s not found in app state - cannot check idleness", jobID)
			return fmt.Errorf("job %s not found in app state", jobID)
		}

		if jobData.UpdatedAt.Add(1 * time.Hour).After(time.Now()) {
			log.Info("ℹ️ Job %s has no PR but is still active - not marking as complete", jobID)
			shouldComplete = false
		} else {
			log.Info("⏰ Job %s has no PR and is idle - marking as complete", jobID)
			reason = "Job complete - Thread is inactive"
			shouldComplete = true
		}
	default:
		log.Info("ℹ️ Job %s PR status unclear (%s) - keeping active", jobID, prStatus)
		shouldComplete = false
	}

	if shouldComplete {
		if err := cr.sendJobCompleteMessage(socketClient, jobID, reason); err != nil {
			log.Error("❌ Failed to send job complete message for job %s: %v", jobID, err)
			return fmt.Errorf("failed to send job complete message: %w", err)
		}

		// Remove job from app state since it's complete
		cr.appState.RemoveJob(jobID)
		log.Info("🗑️ Removed completed job %s from app state", jobID)
	}

	log.Info("📋 Completed successfully - checked idleness for job %s", jobID)
	return nil
}

func (cr *CmdRunner) sendJobCompleteMessage(socketClient *socket.Socket, jobID, reason string) error {
	log.Info("📋 Sending job complete message for job %s with reason: %s", jobID, reason)

	payload := models.JobCompletePayload{
		JobID:  jobID,
		Reason: reason,
	}

	jobMsg := models.UnknownMessage{
		ID:      uuid.New().String(),
		Type:    models.MessageTypeJobComplete,
		Payload: payload,
	}
	if err := socketClient.Emit("cc_message", jobMsg); err != nil {
		log.Info("❌ Failed to send job complete message: %v", err)
		return fmt.Errorf("failed to send job complete message: %w", err)
	}

	log.Info("📤 Sent job complete message for job: %s (message ID: %s)", jobID, jobMsg.ID)

	return nil
}

func (cr *CmdRunner) sendSystemMessage(socketClient *socket.Socket, message, slackMessageID string) error {
	payload := models.SystemMessagePayload{
		Message:        message,
		SlackMessageID: slackMessageID,
	}

	sysMsg := models.UnknownMessage{
		ID:      uuid.New().String(),
		Type:    models.MessageTypeSystemMessage,
		Payload: payload,
	}
	if err := socketClient.Emit("cc_message", sysMsg); err != nil {
		log.Info("❌ Failed to send system message: %v", err)
		return fmt.Errorf("failed to send system message: %w", err)
	}

	log.Info("⚙️ Sent system message: %s (message ID: %s)", message, sysMsg.ID)

	return nil
}

// sendErrorMessage sends an error as a system message. The Claude service handles
// all error processing internally, so we just need to format and send the error.
func (cr *CmdRunner) sendErrorMessage(socketClient *socket.Socket, err error, slackMessageID string) error {
	messageToSend := fmt.Sprintf("ccagent encountered error: %v", err)
	return cr.sendSystemMessage(socketClient, messageToSend, slackMessageID)
}

func (cr *CmdRunner) sendProcessingSlackMessage(socketClient *socket.Socket, slackMessageID string) error {
	processingSlackMessageMsg := models.UnknownMessage{
		ID:   uuid.New().String(),
		Type: models.MessageTypeProcessingSlackMessage,
		Payload: models.ProcessingSlackMessagePayload{
			SlackMessageID: slackMessageID,
		},
	}

	if err := socketClient.Emit("cc_message", processingSlackMessageMsg); err != nil {
		log.Info("❌ Failed to send processing slack message notification: %v", err)
		return fmt.Errorf("failed to send processing slack message notification: %w", err)
	}

	log.Info("🔔 Sent processing slack message notification for message: %s", slackMessageID)
	return nil
}

func extractPRNumber(prURL string) string {
	if prURL == "" {
		return ""
	}

	// Extract PR number from URL like https://github.com/user/repo/pull/1234
	parts := strings.Split(prURL, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return "#" + parts[len(parts)-1]
	}

	return ""
}

func (cr *CmdRunner) sendGitActivitySystemMessage(socketClient *socket.Socket, commitResult *usecases.AutoCommitResult, slackMessageID string) error {
	if commitResult == nil {
		return nil
	}

	if commitResult.JustCreatedPR && commitResult.PullRequestLink != "" {
		// New PR created
		message := fmt.Sprintf("Agent opened a <%s|pull request>", commitResult.PullRequestLink)
		if err := cr.sendSystemMessage(socketClient, message, slackMessageID); err != nil {
			log.Info("❌ Failed to send PR creation system message: %v", err)
			return fmt.Errorf("failed to send PR creation system message: %w", err)
		}
	} else if !commitResult.JustCreatedPR && commitResult.CommitHash != "" && commitResult.RepositoryURL != "" {
		// Commit added to existing PR
		shortHash := commitResult.CommitHash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		commitURL := fmt.Sprintf("%s/commit/%s", commitResult.RepositoryURL, commitResult.CommitHash)
		message := fmt.Sprintf("New commit added: <%s|%s>", commitURL, shortHash)

		// Add PR link if available
		if commitResult.PullRequestLink != "" {
			prNumber := extractPRNumber(commitResult.PullRequestLink)
			if prNumber != "" {
				message += fmt.Sprintf(" in <%s|%s>", commitResult.PullRequestLink, prNumber)
			}
		}

		if err := cr.sendSystemMessage(socketClient, message, slackMessageID); err != nil {
			log.Info("❌ Failed to send commit system message: %v", err)
			return fmt.Errorf("failed to send commit system message: %w", err)
		}
	}

	return nil
}

func unmarshalPayload(payload any, target any) error {
	if payload == nil {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return json.Unmarshal(payloadBytes, target)
}
