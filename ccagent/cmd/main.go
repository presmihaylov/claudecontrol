package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/google/uuid"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"github.com/jessevdk/go-flags"

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
	gitUseCase := usecases.NewGitUseCase(gitClient, claudeService)
	appState := models.NewAppState()

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

	// Get Socket.IO URL from environment variable with default fallback
	wsURL := os.Getenv("CCAGENT_WS_API_URL")
	if wsURL == "" {
		wsURL = "https://claudecontrol.onrender.com"
	}

	// Set up deferred cleanup
	defer func() {
		fmt.Fprintf(os.Stderr, "\n📝 App execution finished, logs for this session are stored in %s\n", cmdRunner.logFilePath)
	}()

	// Start Socket.IO client
	err = cmdRunner.startSocketIOClient(wsURL, ccagentAPIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting Socket.IO client: %v\n", err)
		os.Exit(1)
	}
}

func (cr *CmdRunner) startSocketIOClient(serverURLStr, apiKey string) error {
	log.Info("📋 Starting to connect to Socket.IO server at %s", serverURLStr)
	
	// Set up global interrupt handling
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Retry intervals in seconds: 5, 10, 20, 30, 60, 120
	retryIntervals := []time.Duration{
		5 * time.Second,
		10 * time.Second,
		20 * time.Second,
		30 * time.Second,
		60 * time.Second,
		120 * time.Second,
	}

	for {
		client, connected := cr.connectSocketIOWithRetry(serverURLStr, apiKey, retryIntervals, interrupt)
		if client == nil {
			select {
			case <-interrupt:
				log.Info("🔌 Interrupt received during connection attempts, shutting down")
				return nil
			default:
				log.Info("❌ All retry attempts exhausted, shutting down")
				return fmt.Errorf("failed to connect after all retry attempts")
			}
		}

		if !connected {
			select {
			case <-interrupt:
				log.Info("🔌 Interrupt received, shutting down")
				return nil
			default:
				continue // Retry loop will handle reconnection
			}
		}

		log.Info("✅ Connected to Socket.IO server")

		done := make(chan struct{})
		reconnect := make(chan struct{})

		// Initialize worker pool with 1 worker for sequential processing
		wp := workerpool.New(1)
		defer wp.StopWait()

		// Set up event handlers
		cr.setupSocketIOEventHandlers(client, wp, reconnect)

		// Wait for connection to close or interruption  
		shouldExit := false
		select {
		case <-done:
			// Connection closed, trigger reconnection
			client.Close()
			log.Info("🔄 Connection lost, attempting to reconnect...")
		case <-reconnect:
			// Connection lost, trigger reconnection
			client.Close()
			log.Info("🔄 Connection lost, attempting to reconnect...")
		case <-interrupt:
			log.Info("🔌 Interrupt received, closing connection...")
			client.Close()
			shouldExit = true
		}

		if shouldExit {
			return nil
		}
	}
}

func (cr *CmdRunner) connectSocketIOWithRetry(serverURL, apiKey string, retryIntervals []time.Duration, interrupt <-chan os.Signal) (*socketio.Client, bool) {
	log.Info("🔌 Attempting to connect to Socket.IO server at %s", serverURL)

	// Build Socket.IO URL with query parameters for authentication
	socketURL := fmt.Sprintf("%s/socket.io/?api_key=%s&agent_id=%s", serverURL, url.QueryEscape(apiKey), url.QueryEscape(cr.agentID.String()))
	
	client, err := socketio.NewClient(socketURL)
	if err == nil {
		err = client.Connect()
		if err == nil {
			return client, true
		}
	}

	log.Info("❌ Initial connection failed: %v", err)
	log.Info("🔄 Starting retry sequence with exponential backoff...")

	for attempt, interval := range retryIntervals {
		log.Info("⏱️ Waiting %v before retry attempt %d/%d", interval, attempt+1, len(retryIntervals))

		// Use select to wait for either timeout or interrupt
		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
			// Timer expired, continue with retry
		case <-interrupt:
			timer.Stop()
			log.Info("🔌 Interrupt received during retry wait, aborting")
			return nil, false
		}

		log.Info("🔌 Retry attempt %d/%d: connecting to %s", attempt+1, len(retryIntervals), socketURL)
		client, err := socketio.NewClient(socketURL)
		if err == nil {
			err = client.Connect()
			if err == nil {
				log.Info("✅ Successfully connected on retry attempt %d/%d", attempt+1, len(retryIntervals))
				return client, true
			}
		}

		log.Info("❌ Retry attempt %d/%d failed: %v", attempt+1, len(retryIntervals), err)
	}

	log.Info("💀 All %d retry attempts failed, giving up", len(retryIntervals))
	return nil, false
}

func (cr *CmdRunner) setupSocketIOEventHandlers(client *socketio.Client, wp *workerpool.WorkerPool, reconnect chan struct{}) {
	// Handle connection events
	client.OnConnect(func() {
		log.Info("🔗 Socket.IO connected")
	})

	client.OnDisconnect(func() {
		log.Info("🔌 Socket.IO disconnected")
		close(reconnect)
	})

	client.OnError(func(err error) {
		log.Info("❌ Socket.IO error: %v", err)
		close(reconnect)
	})

	// Handle message events - maintain compatibility with existing protocol
	client.On("message", func(data any) {
		log.Info("📥 Generic message received")
		wp.Submit(func() {
			cr.handleSocketIOMessage(data, client)
		})
	})

	// Handle specific event types
	eventTypes := []string{
		"start_conversation_v1",
		"user_message_v1",
		"job_unassigned_v1", 
		"check_idle_jobs_v1",
	}

	for _, eventType := range eventTypes {
		eventType := eventType // capture for closure
		client.On(eventType, func(data any) {
			log.Info("📥 Event '%s' received", eventType)
			wp.Submit(func() {
				// Convert to compatible message format
				msg := models.UnknownMessage{
					Type:    eventType,
					Payload: data,
				}
				cr.handleSocketIOMessage(msg, client)
			})
		})
	}
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

func (cr *CmdRunner) handleSocketIOMessage(msg any, client *socketio.Client) {
	// Convert message to UnknownMessage format if needed
	var unknownMsg models.UnknownMessage
	if umsg, ok := msg.(models.UnknownMessage); ok {
		unknownMsg = umsg
	} else {
		// Try to convert from raw data
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Info("❌ Failed to marshal message: %v", err)
			return
		}
		if err := json.Unmarshal(msgBytes, &unknownMsg); err != nil {
			log.Info("❌ Failed to parse message: %v", err)
			return
		}
	}

	switch unknownMsg.Type {
	case models.MessageTypeStartConversation:
		if err := cr.handleStartConversation(unknownMsg, client); err != nil {
			// Extract SlackMessageID from payload for error reporting
			var payload models.StartConversationPayload
			slackMessageID := ""
			if unmarshalErr := unmarshalPayload(unknownMsg.Payload, &payload); unmarshalErr == nil {
				slackMessageID = payload.SlackMessageID
			}
			if sendErr := cr.sendErrorMessage(client, err, slackMessageID); sendErr != nil {
				log.Error("Failed to send error message: %v", sendErr)
			}
		}
	case models.MessageTypeUserMessage:
		if err := cr.handleUserMessage(unknownMsg, client); err != nil {
			// Extract SlackMessageID from payload for error reporting
			var payload models.UserMessagePayload
			slackMessageID := ""
			if unmarshalErr := unmarshalPayload(unknownMsg.Payload, &payload); unmarshalErr == nil {
				slackMessageID = payload.SlackMessageID
			}
			if sendErr := cr.sendErrorMessage(client, err, slackMessageID); sendErr != nil {
				log.Error("Failed to send error message: %v", sendErr)
			}
		}
	case models.MessageTypeJobUnassigned:
		if err := cr.handleJobUnassigned(unknownMsg, client); err != nil {
			log.Info("❌ Error handling JobUnassigned message: %v", err)
		}
	case models.MessageTypeCheckIdleJobs:
		if err := cr.handleCheckIdleJobs(unknownMsg, client); err != nil {
			log.Info("❌ Error handling CheckIdleJobs message: %v", err)
		}
	default:
		log.Info("⚠️ Unhandled message type: %s", unknownMsg.Type)
	}
}

func (cr *CmdRunner) handleStartConversation(msg models.UnknownMessage, client *socketio.Client) error {
	log.Info("📋 Starting to handle start conversation message")
	var payload models.StartConversationPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal start conversation payload: %v", err)
		return fmt.Errorf("failed to unmarshal start conversation payload: %w", err)
	}

	// Send processing slack message notification that agent is starting to process
	if err := cr.sendProcessingSlackMessage(client, payload.SlackMessageID); err != nil {
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
		systemErr := cr.sendSystemMessage(client, fmt.Sprintf("ccagent encountered error: %v", err), payload.SlackMessageID)
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
	client.Emit(assistantMsg.Type, assistantMsg.Payload)
	if false { // Socket.IO doesn't return errors from emit, so we skip error handling
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}
	log.Info("🤖 Sent assistant response (message ID: %s)", assistantMsg.ID)

	// Send system message after assistant message for git activity
	if err := cr.sendGitActivitySystemMessage(client, commitResult, payload.SlackMessageID); err != nil {
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

func (cr *CmdRunner) handleUserMessage(msg models.UnknownMessage, client *socketio.Client) error {
	log.Info("📋 Starting to handle user message")
	var payload models.UserMessagePayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal user message payload: %v", err)
		return fmt.Errorf("failed to unmarshal user message payload: %w", err)
	}

	// Send processing slack message notification that agent is starting to process
	if err := cr.sendProcessingSlackMessage(client, payload.SlackMessageID); err != nil {
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
		systemErr := cr.sendSystemMessage(client, fmt.Sprintf("ccagent encountered error: %v", err), payload.SlackMessageID)
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
	client.Emit(assistantMsg.Type, assistantMsg.Payload)
	if false { // Socket.IO doesn't return errors from emit, so we skip error handling
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}
	log.Info("🤖 Sent assistant response (message ID: %s)", assistantMsg.ID)

	// Send system message after assistant message for git activity
	if err := cr.sendGitActivitySystemMessage(client, commitResult, payload.SlackMessageID); err != nil {
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

func (cr *CmdRunner) handleJobUnassigned(msg models.UnknownMessage, _ *socketio.Client) error {
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

func (cr *CmdRunner) handleCheckIdleJobs(msg models.UnknownMessage, client *socketio.Client) error {
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

		if err := cr.checkJobIdleness(jobID, jobData, conn); err != nil {
			log.Info("❌ Failed to check idleness for job %s: %v", jobID, err)
			// Continue checking other jobs even if one fails
			continue
		}
	}

	log.Info("📋 Completed successfully - checked all jobs for idleness")
	return nil
}

func (cr *CmdRunner) checkJobIdleness(jobID string, jobData models.JobData, client *socketio.Client) error {
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
		if err := cr.sendJobCompleteMessage(client, jobID, reason); err != nil {
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

func (cr *CmdRunner) sendJobCompleteMessage(conn *socketio.Client, jobID, reason string) error {
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
	client.Emit(jobMsg.Type, jobMsg.Payload)
	if false { // Socket.IO doesn't return errors from emit, so we skip error handling
		log.Error("❌ Failed to send job complete message: %v", err)
		return fmt.Errorf("failed to send job complete message: %w", err)
	}
	log.Info("📤 Sent job complete message for job: %s (message ID: %s)", jobID, jobMsg.ID)

	return nil
}

func (cr *CmdRunner) sendSystemMessage(conn *socketio.Client, message, slackMessageID string) error {
	payload := models.SystemMessagePayload{
		Message:        message,
		SlackMessageID: slackMessageID,
	}

	sysMsg := models.UnknownMessage{
		ID:      uuid.New().String(),
		Type:    models.MessageTypeSystemMessage,
		Payload: payload,
	}
	client.Emit(sysMsg.Type, sysMsg.Payload)
	if false { // Socket.IO doesn't return errors from emit, so we skip error handling
		log.Info("❌ Failed to send system message: %v", err)
		return err
	}
	log.Info("⚙️ Sent system message: %s (message ID: %s)", message, sysMsg.ID)

	return nil
}

// sendErrorMessage sends an error as a system message. The Claude service handles
// all error processing internally, so we just need to format and send the error.
func (cr *CmdRunner) sendErrorMessage(conn *socketio.Client, err error, slackMessageID string) error {
	messageToSend := fmt.Sprintf("ccagent encountered error: %v", err)
	return cr.sendSystemMessage(client, messageToSend, slackMessageID)
}

func (cr *CmdRunner) sendProcessingSlackMessage(conn *socketio.Client, slackMessageID string) error {
	processingSlackMessageMsg := models.UnknownMessage{
		ID:   uuid.New().String(),
		Type: models.MessageTypeProcessingSlackMessage,
		Payload: models.ProcessingSlackMessagePayload{
			SlackMessageID: slackMessageID,
		},
	}

	client.Emit(processingSlackMessageMsg.Type, processingSlackMessageMsg.Payload)
	if false { // Socket.IO doesn't return errors from emit, so we skip error handling
		log.Info("❌ Failed to send processing slack message notification: %v", err)
		return err
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

func (cr *CmdRunner) sendGitActivitySystemMessage(conn *socketio.Client, commitResult *usecases.AutoCommitResult, slackMessageID string) error {
	if commitResult == nil {
		return nil
	}

	if commitResult.JustCreatedPR && commitResult.PullRequestLink != "" {
		// New PR created
		message := fmt.Sprintf("Agent opened a <%s|pull request>", commitResult.PullRequestLink)
		if err := cr.sendSystemMessage(client, message, slackMessageID); err != nil {
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

		if err := cr.sendSystemMessage(client, message, slackMessageID); err != nil {
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
