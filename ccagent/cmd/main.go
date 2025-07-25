package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"time"

	"ccagent/clients"
	"ccagent/core/log"
	"ccagent/services"
	"ccagent/usecases"

	"github.com/gorilla/websocket"
	"github.com/jessevdk/go-flags"
)

type CmdRunner struct {
	configService  *services.ConfigService
	sessionService *services.SessionService
	claudeClient   *clients.ClaudeClient
	gitUseCase     *usecases.GitUseCase
}

func NewCmdRunner() *CmdRunner {
	log.Info("📋 Starting to initialize CmdRunner")
	configService := services.NewConfigService()
	sessionService := services.NewSessionService()
	claudeClient := clients.NewClaudeClient()
	gitClient := clients.NewGitClient()
	gitUseCase := usecases.NewGitUseCase(gitClient, claudeClient)

	log.Info("📋 Completed successfully - initialized CmdRunner with all services")
	return &CmdRunner{
		configService:  configService,
		sessionService: sessionService,
		claudeClient:   claudeClient,
		gitUseCase:     gitUseCase,
	}
}

type Options struct {
	Verbose bool   `short:"v" long:"verbose" description:"Enable verbose logging"`
	URL     string `short:"u" long:"url" default:"ws://localhost:8080/ws" description:"WebSocket server URL"`
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

	if opts.Verbose {
		log.SetLevel(slog.LevelInfo)
	}

	cmdRunner := NewCmdRunner()

	_, err = cmdRunner.configService.GetOrCreateConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Validate Git environment before starting
	err = cmdRunner.gitUseCase.ValidateGitEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Git environment validation failed: %v\n", err)
		os.Exit(1)
	}

	// Start WebSocket client
	err = cmdRunner.startWebSocketClient(opts.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting WebSocket client: %v\n", err)
		os.Exit(1)
	}
}

func (cr *CmdRunner) startWebSocketClient(serverURL string) error {
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
		conn, connected := cr.connectWithRetry(u.String(), retryIntervals, interrupt)
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

		// Start message reading goroutine
		go func() {
			defer close(done)
			for {
				var msg UnknownMessage
				err := conn.ReadJSON(&msg)
				if err != nil {
					log.Info("❌ Read error: %v", err)
					cr.sendSystemMessage(conn, fmt.Sprintf("ccagent encountered error: %v", err))
					close(reconnect)
					return
				}

				log.Info("📨 Received message type: %s", msg.Type)
				cr.handleMessage(msg, conn)
			}
		}()

		// Main loop for this connection
		shouldExit := false
		for {
			select {
			case <-done:
				// Connection closed, trigger reconnection
				conn.Close()
				log.Info("🔄 Connection lost, attempting to reconnect...")
				break
			case <-reconnect:
				// Connection lost from read goroutine, trigger reconnection
				conn.Close()
				log.Info("🔄 Connection lost, attempting to reconnect...")
				break
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
			break
		}

		if shouldExit {
			return nil
		}
	}
}

func (cr *CmdRunner) connectWithRetry(serverURL string, retryIntervals []time.Duration, interrupt <-chan os.Signal) (*websocket.Conn, bool) {
	log.Info("🔌 Attempting to connect to WebSocket server at %s", serverURL)

	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
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
		conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
		if err == nil {
			log.Info("✅ Successfully connected on retry attempt %d/%d", attempt+1, len(retryIntervals))
			return conn, true
		}

		log.Info("❌ Retry attempt %d/%d failed: %v", attempt+1, len(retryIntervals), err)
	}

	log.Info("💀 All %d retry attempts failed, giving up", len(retryIntervals))
	return nil, false
}

func (cr *CmdRunner) handleMessage(msg UnknownMessage, conn *websocket.Conn) {
	switch msg.Type {
	case MessageTypeStartConversation:
		if err := cr.handleStartConversation(msg, conn); err != nil {
			cr.sendSystemMessage(conn, fmt.Sprintf("ccagent encountered error: %v", err))
		}
	case MessageTypeUserMessage:
		if err := cr.handleUserMessage(msg, conn); err != nil {
			cr.sendSystemMessage(conn, fmt.Sprintf("ccagent encountered error: %v", err))
		}
	case MessageTypeJobUnassigned:
		if err := cr.handleJobUnassigned(msg, conn); err != nil {
			cr.sendSystemMessage(conn, fmt.Sprintf("ccagent encountered error: %v", err))
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

	log.Info("🚀 Starting new conversation with message: %s", payload.Message)

	// Prepare Git environment for new conversation - FAIL if this doesn't work
	if err := cr.gitUseCase.PrepareForNewConversation(payload.Message); err != nil {
		log.Error("❌ Failed to prepare Git environment: %v", err)
		return fmt.Errorf("failed to prepare Git environment: %w", err)
	}

	behaviourInstructions := `You are a claude code instance which will be referred to by the user as "Claude Control" for this session. When someone calls you claude control, they refer to you.

You are being interacted with over Slack (the software). I want you to adjust your responses to account for this. In particular:
- keep your responses more succinct than usual because it's hard to read very long messages in slack. Use long messages only if user asks for it
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
`

	_, err := cr.claudeClient.StartNewSession(behaviourInstructions)
	if err != nil {
		log.Info("❌ Error starting Claude session with behaviour prompt: %v", err)
		return fmt.Errorf("error starting Claude session with behaviour prompt: %w", err)
	}

	output, err := cr.claudeClient.ContinueSession("dummy-session", payload.Message)
	if err != nil {
		log.Info("❌ Error starting Claude session: %v", err)
		return fmt.Errorf("error starting Claude session: %w", err)
	}

	// Send assistant response back
	response := UnknownMessage{
		Type: MessageTypeAssistantMessage,
		Payload: AssistantMessagePayload{
			Message:        output,
			SlackMessageID: payload.SlackMessageID,
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}

	log.Info("🤖 Sent assistant response")
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

	log.Info("💬 Continuing conversation with message: %s", payload.Message)

	// For now, we'll use a dummy session ID since ContinueSession isn't working properly
	// according to the comment in claude.go
	output, err := cr.claudeClient.ContinueSession("dummy-session", payload.Message)
	if err != nil {
		log.Info("❌ Error continuing Claude session: %v", err)
		return fmt.Errorf("error continuing Claude session: %w", err)
	}

	// Send assistant response back
	response := UnknownMessage{
		Type: MessageTypeAssistantMessage,
		Payload: AssistantMessagePayload{
			Message:        output,
			SlackMessageID: payload.SlackMessageID,
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}

	log.Info("🤖 Sent assistant response")
	log.Info("📋 Completed successfully - handled user message")
	return nil
}

func (cr *CmdRunner) handleJobUnassigned(msg UnknownMessage, conn *websocket.Conn) error {
	log.Info("📋 Starting to handle job unassigned message")
	var payload JobUnassignedPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal job unassigned payload: %v", err)
		return fmt.Errorf("failed to unmarshal job unassigned payload: %w", err)
	}

	log.Info("🚫 Job has been unassigned from this agent")

	// Complete job and create PR - FAIL if this doesn't work
	if err := cr.gitUseCase.CompleteJobAndCreatePR(); err != nil {
		log.Error("❌ Failed to complete job and create PR: %v", err)
		return fmt.Errorf("failed to complete job and create PR: %w", err)
	}

	log.Info("📋 Completed successfully - handled job unassigned message")
	return nil
}

func (cr *CmdRunner) sendSystemMessage(conn *websocket.Conn, message string) {
	log.Info("📋 Sending system message: %s", message)
	response := UnknownMessage{
		Type: MessageTypeSystemMessage,
		Payload: SystemMessagePayload{
			Message: message,
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Info("❌ Failed to send system message: %v", err)
	} else {
		log.Info("⚙️ Sent system message")
	}
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
