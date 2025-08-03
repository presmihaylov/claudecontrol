package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"ccagent/clients"
	"ccagent/core/log"
	"ccagent/models"
	"ccagent/services"
	"ccagent/usecases"
	"ccagent/utils"

	"github.com/gammazero/workerpool"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jessevdk/go-flags"
)

type CmdRunner struct {
	sessionService *services.SessionService
	claudeService  *services.ClaudeService
	gitUseCase     *usecases.GitUseCase
	appState       *models.AppState
	logFilePath    string
	agentID        uuid.UUID
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

	// Validate CCAGENT_API_KEY environment variable
	ccagentApiKey := os.Getenv("CCAGENT_API_KEY")
	if ccagentApiKey == "" {
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

	// Get WebSocket URL from environment variable with default fallback
	wsURL := os.Getenv("CCAGENT_WS_API_URL")
	if wsURL == "" {
		wsURL = "wss://claudecontrol.onrender.com/ws"
	}

	// Set up deferred exit message
	defer func() {
		fmt.Fprintf(os.Stderr, "\n📝 App execution finished, logs for this session are stored in %s\n", cmdRunner.logFilePath)
	}()

	// Start WebSocket client
	err = cmdRunner.startWebSocketClient(wsURL, ccagentApiKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting WebSocket client: %v\n", err)
		os.Exit(1)
	}
}

func (cr *CmdRunner) startWebSocketClient(serverURL, apiKey string) error {
	log.Info("📋 Starting to connect to WebSocket server at %s", serverURL)
	u, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

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
		conn, connected := cr.connectWithRetry(u.String(), apiKey, retryIntervals, interrupt)
		if conn == nil {
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

		log.Info("✅ Connected to WebSocket server")

		done := make(chan struct{})
		reconnect := make(chan struct{})

		// Initialize worker pool with 1 worker for sequential processing
		wp := workerpool.New(1)
		defer wp.StopWait()

		// Initialize instant worker pool for healthcheck messages
		instantWP := workerpool.New(1)
		defer instantWP.StopWait()

		// Start message reading goroutine
		go func() {
			defer close(done)
			defer wp.StopWait()      // Ensure all queued messages complete
			defer instantWP.StopWait() // Ensure all instant messages complete

			for {
				var msg UnknownMessage
				err := conn.ReadJSON(&msg)
				if err != nil {
					log.Info("❌ Read error: %v", err)

					// trigger ws reconnect
					close(reconnect)
					return
				}

				log.Info("📨 Received message type: %s", msg.Type)

				// Route healthcheck messages to instant worker pool
				if msg.Type == MessageTypeHealthcheckCheck {
					instantWP.Submit(func() {
						cr.handleMessage(msg, conn)
					})
				} else {
					// NON-BLOCKING: Submit to regular worker pool
					wp.Submit(func() {
						cr.handleMessage(msg, conn)
					})
				}
			}
		}()

		// Wait for connection to close or interruption
		shouldExit := false
		select {
		case <-done:
			// Connection closed, trigger reconnection
			conn.Close()
			log.Info("🔄 Connection lost, attempting to reconnect...")
		case <-reconnect:
			// Connection lost from read goroutine, trigger reconnection
			conn.Close()
			log.Info("🔄 Connection lost, attempting to reconnect...")
		case <-interrupt:
			log.Info("🔌 Interrupt received, closing connection...")

			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Info("❌ Failed to send close message: %v", err)
			}

			select {
			case <-done:
			case <-time.After(time.Second):
			}
			conn.Close()
			shouldExit = true
		}

		if shouldExit {
			return nil
		}
	}
}

func (cr *CmdRunner) connectWithRetry(serverURL, apiKey string, retryIntervals []time.Duration, interrupt <-chan os.Signal) (*websocket.Conn, bool) {
	log.Info("🔌 Attempting to connect to WebSocket server at %s", serverURL)

	headers := http.Header{
		"X-CCAGENT-API-KEY": []string{apiKey},
		"X-CCAGENT-ID":      []string{cr.agentID.String()},
	}
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, headers)
	if err == nil {
		return conn, true
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

		log.Info("🔌 Retry attempt %d/%d: connecting to %s", attempt+1, len(retryIntervals), serverURL)
		conn, _, err := websocket.DefaultDialer.Dial(serverURL, headers)
		if err == nil {
			log.Info("✅ Successfully connected on retry attempt %d/%d", attempt+1, len(retryIntervals))
			return conn, true
		}

		log.Info("❌ Retry attempt %d/%d failed: %v", attempt+1, len(retryIntervals), err)
	}

	log.Info("💀 All %d retry attempts failed, giving up", len(retryIntervals))
	return nil, false
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

func (cr *CmdRunner) handleMessage(msg UnknownMessage, conn *websocket.Conn) {
	switch msg.Type {
	case MessageTypeStartConversation:
		if err := cr.handleStartConversation(msg, conn); err != nil {
			// Extract SlackMessageID from payload for error reporting
			var payload StartConversationPayload
			slackMessageID := ""
			if unmarshalErr := unmarshalPayload(msg.Payload, &payload); unmarshalErr == nil {
				slackMessageID = payload.SlackMessageID
			}
			cr.sendErrorMessage(conn, err, slackMessageID)
		}
	case MessageTypeUserMessage:
		if err := cr.handleUserMessage(msg, conn); err != nil {
			// Extract SlackMessageID from payload for error reporting
			var payload UserMessagePayload
			slackMessageID := ""
			if unmarshalErr := unmarshalPayload(msg.Payload, &payload); unmarshalErr == nil {
				slackMessageID = payload.SlackMessageID
			}
			cr.sendErrorMessage(conn, err, slackMessageID)
		}
	case MessageTypeJobUnassigned:
		if err := cr.handleJobUnassigned(msg, conn); err != nil {
			log.Info("❌ Error handling JobUnassigned message: %v", err)
		}
	case MessageTypeCheckIdleJobs:
		if err := cr.handleCheckIdleJobs(msg, conn); err != nil {
			log.Info("❌ Error handling CheckIdleJobs message: %v", err)
		}
	case MessageTypeHealthcheckCheck:
		if err := cr.handleHealthcheckCheck(msg, conn); err != nil {
			log.Info("❌ Error handling HealthcheckCheck message: %v", err)
		}
	default:
		log.Info("⚠️ Unhandled message type: %s", msg.Type)
	}
}

func (cr *CmdRunner) handleStartConversation(msg UnknownMessage, conn *websocket.Conn) error {
	log.Info("📋 Starting to handle start conversation message")
	var payload StartConversationPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal start conversation payload: %v", err)
		return fmt.Errorf("failed to unmarshal start conversation payload: %w", err)
	}

	// Send processing slack message notification that agent is starting to process
	if err := cr.sendProcessingSlackMessage(conn, payload.SlackMessageID); err != nil {
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
	response := UnknownMessage{
		Type: MessageTypeAssistantMessage,
		Payload: AssistantMessagePayload{
			JobID:          payload.JobID,
			Message:        claudeResult.Output,
			SlackMessageID: payload.SlackMessageID,
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}

	log.Info("🤖 Sent assistant response")

	// Send system message after assistant message for git activity
	if err := cr.sendGitActivitySystemMessage(conn, commitResult, payload.SlackMessageID); err != nil {
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

func (cr *CmdRunner) handleUserMessage(msg UnknownMessage, conn *websocket.Conn) error {
	log.Info("📋 Starting to handle user message")
	var payload UserMessagePayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal user message payload: %v", err)
		return fmt.Errorf("failed to unmarshal user message payload: %w", err)
	}

	// Send processing slack message notification that agent is starting to process
	if err := cr.sendProcessingSlackMessage(conn, payload.SlackMessageID); err != nil {
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
	response := UnknownMessage{
		Type: MessageTypeAssistantMessage,
		Payload: AssistantMessagePayload{
			JobID:          payload.JobID,
			Message:        claudeResult.Output,
			SlackMessageID: payload.SlackMessageID,
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}

	log.Info("🤖 Sent assistant response")

	// Send system message after assistant message for git activity
	if err := cr.sendGitActivitySystemMessage(conn, commitResult, payload.SlackMessageID); err != nil {
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

func (cr *CmdRunner) handleJobUnassigned(msg UnknownMessage, _ *websocket.Conn) error {
	log.Info("📋 Starting to handle job unassigned message")
	var payload JobUnassignedPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal job unassigned payload: %v", err)
		return fmt.Errorf("failed to unmarshal job unassigned payload: %w", err)
	}

	log.Info("🚫 Job has been unassigned from this agent")
	log.Info("📋 Completed successfully - handled job unassigned message")
	return nil
}

func (cr *CmdRunner) handleCheckIdleJobs(msg UnknownMessage, conn *websocket.Conn) error {
	log.Info("📋 Starting to handle check idle jobs message")
	var payload CheckIdleJobsPayload
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

func (cr *CmdRunner) handleHealthcheckCheck(msg UnknownMessage, conn *websocket.Conn) error {
	log.Info("📋 Starting to handle healthcheck check message")
	var payload HealthcheckCheckPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal healthcheck check payload: %v", err)
		return fmt.Errorf("failed to unmarshal healthcheck check payload: %w", err)
	}

	log.Info("💓 Received healthcheck ping from backend - sending ack")

	// Send healthcheck acknowledgment back to backend
	healthcheckAckMsg := UnknownMessage{
		Type:    MessageTypeHealthcheckAck,
		Payload: HealthcheckAckPayload{},
	}

	if err := conn.WriteJSON(healthcheckAckMsg); err != nil {
		log.Info("❌ Failed to send healthcheck ack: %v", err)
		return fmt.Errorf("failed to send healthcheck ack: %w", err)
	}

	log.Info("💓 Sent healthcheck ack to backend")
	log.Info("📋 Completed successfully - handled healthcheck check message")
	return nil
}

func (cr *CmdRunner) checkJobIdleness(jobID string, jobData models.JobData, conn *websocket.Conn) error {
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
		if err := cr.sendJobCompleteMessage(conn, jobID, reason); err != nil {
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

func (cr *CmdRunner) sendJobCompleteMessage(conn *websocket.Conn, jobID, reason string) error {
	log.Info("📋 Sending job complete message for job %s with reason: %s", jobID, reason)

	jobCompleteMsg := UnknownMessage{
		Type: MessageTypeJobComplete,
		Payload: JobCompletePayload{
			JobID:  jobID,
			Reason: reason,
		},
	}

	if err := conn.WriteJSON(jobCompleteMsg); err != nil {
		log.Error("❌ Failed to send job complete message: %v", err)
		return fmt.Errorf("failed to send job complete message: %w", err)
	}

	log.Info("📤 Sent job complete message for job: %s", jobID)
	return nil
}

func (cr *CmdRunner) sendSystemMessage(conn *websocket.Conn, message, slackMessageID string) error {
	systemMsg := UnknownMessage{
		Type: MessageTypeSystemMessage,
		Payload: SystemMessagePayload{
			Message:        message,
			SlackMessageID: slackMessageID,
		},
	}

	if err := conn.WriteJSON(systemMsg); err != nil {
		log.Info("❌ Failed to send system message: %v", err)
		return err
	}

	log.Info("⚙️ Sent system message: %s", message)
	return nil
}

// sendErrorMessage sends an error as a system message. The Claude service handles
// all error processing internally, so we just need to format and send the error.
func (cr *CmdRunner) sendErrorMessage(conn *websocket.Conn, err error, slackMessageID string) error {
	messageToSend := fmt.Sprintf("ccagent encountered error: %v", err)
	return cr.sendSystemMessage(conn, messageToSend, slackMessageID)
}

func (cr *CmdRunner) sendProcessingSlackMessage(conn *websocket.Conn, slackMessageID string) error {
	processingSlackMessageMsg := UnknownMessage{
		Type: MessageTypeProcessingSlackMessage,
		Payload: ProcessingSlackMessagePayload{
			SlackMessageID: slackMessageID,
		},
	}

	if err := conn.WriteJSON(processingSlackMessageMsg); err != nil {
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

func (cr *CmdRunner) sendGitActivitySystemMessage(conn *websocket.Conn, commitResult *usecases.AutoCommitResult, slackMessageID string) error {
	if commitResult == nil {
		return nil
	}

	if commitResult.JustCreatedPR && commitResult.PullRequestLink != "" {
		// New PR created
		message := fmt.Sprintf("Agent opened a <%s|pull request>", commitResult.PullRequestLink)
		if err := cr.sendSystemMessage(conn, message, slackMessageID); err != nil {
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

		if err := cr.sendSystemMessage(conn, message, slackMessageID); err != nil {
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
