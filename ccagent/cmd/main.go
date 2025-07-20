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

	"github.com/gorilla/websocket"
	"github.com/jessevdk/go-flags"
)

type CmdRunner struct {
	configService  *services.ConfigService
	sessionService *services.SessionService
	claudeClient   *clients.ClaudeClient
}

func NewCmdRunner() *CmdRunner {
	log.Info("📋 Starting to initialize CmdRunner")
	configService := services.NewConfigService()
	sessionService := services.NewSessionService()
	claudeClient := clients.NewClaudeClient()

	log.Info("📋 Completed successfully - initialized CmdRunner with all services")
	return &CmdRunner{
		configService:  configService,
		sessionService: sessionService,
		claudeClient:   claudeClient,
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
		conn, connected := cr.connectWithRetry(u.String(), retryIntervals)
		if conn == nil {
			log.Info("❌ All retry attempts exhausted, shutting down")
			return fmt.Errorf("failed to connect after all retry attempts")
		}

		if !connected {
			continue // Retry loop will handle reconnection
		}

		log.Info("✅ Connected to WebSocket server")

		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)

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

func (cr *CmdRunner) connectWithRetry(serverURL string, retryIntervals []time.Duration) (*websocket.Conn, bool) {
	log.Info("🔌 Attempting to connect to WebSocket server at %s", serverURL)

	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err == nil {
		return conn, true
	}

	log.Info("❌ Initial connection failed: %v", err)
	log.Info("🔄 Starting retry sequence with exponential backoff...")

	for attempt, interval := range retryIntervals {
		log.Info("⏱️ Waiting %v before retry attempt %d/%d", interval, attempt+1, len(retryIntervals))
		time.Sleep(interval)

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
		cr.handleStartConversation(msg, conn)
	case MessageTypeUserMessage:
		cr.handleUserMessage(msg, conn)
	case MessageTypePing:
		cr.handlePing(conn)
	default:
		log.Info("⚠️ Unhandled message type: %s", msg.Type)
	}
}

func (cr *CmdRunner) handleStartConversation(msg UnknownMessage, conn *websocket.Conn) {
	log.Info("📋 Starting to handle start conversation message")
	var payload StartConversationPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal start conversation payload: %v", err)
		return
	}

	log.Info("🚀 Starting new conversation with message: %s", payload.Message)

	output, err := cr.claudeClient.StartNewSession(payload.Message)
	if err != nil {
		log.Info("❌ Error starting Claude session: %v", err)
		return
	}

	// Send assistant response back
	response := UnknownMessage{
		Type: MessageTypeAssistantMessage,
		Payload: AssistantMessagePayload{
			Message: output,
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
	} else {
		log.Info("🤖 Sent assistant response")
		log.Info("📋 Completed successfully - handled start conversation message")
	}
}

func (cr *CmdRunner) handleUserMessage(msg UnknownMessage, conn *websocket.Conn) {
	log.Info("📋 Starting to handle user message")
	var payload UserMessagePayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal user message payload: %v", err)
		return
	}

	log.Info("💬 Continuing conversation with message: %s", payload.Message)

	// For now, we'll use a dummy session ID since ContinueSession isn't working properly
	// according to the comment in claude.go
	output, err := cr.claudeClient.ContinueSession("dummy-session", payload.Message)
	if err != nil {
		log.Info("❌ Error continuing Claude session: %v", err)
		return
	}

	// Send assistant response back
	response := UnknownMessage{
		Type: MessageTypeAssistantMessage,
		Payload: AssistantMessagePayload{
			Message: output,
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
	} else {
		log.Info("🤖 Sent assistant response")
		log.Info("📋 Completed successfully - handled user message")
	}
}

func (cr *CmdRunner) handlePing(conn *websocket.Conn) {
	log.Info("📋 Starting to handle ping message")
	log.Info("🏓 Received ping, sending pong")

	response := UnknownMessage{
		Type:    MessageTypePong,
		Payload: PongPayload{},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Info("❌ Failed to send pong: %v", err)
	} else {
		log.Info("🏓 Sent pong response")
		log.Info("📋 Completed successfully - handled ping message")
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

