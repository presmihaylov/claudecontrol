package main

import (
	"fmt"
	"log/slog"
	"os"

	"ccagent/clients"
	"ccagent/core/log"
	"ccagent/services"

	"github.com/jessevdk/go-flags"
)

type NewCommand struct {
}

type ContinueCommand struct {
	SessionID string `long:"session-id" short:"s" required:"true" description:"Session ID to continue"`
	Prompt    string `short:"p" long:"prompt" required:"true" description:"Prompt to send to Claude"`
}

type CmdRunner struct {
	configService  *services.ConfigService
	sessionService *services.SessionService
	claudeClient   *clients.ClaudeClient
}

func NewCmdRunner() *CmdRunner {
	configService := services.NewConfigService()
	sessionService := services.NewSessionService()
	claudeClient := clients.NewClaudeClient()

	return &CmdRunner{
		configService:  configService,
		sessionService: sessionService,
		claudeClient:   claudeClient,
	}
}

type Options struct {
	Verbose  bool            `short:"v" long:"verbose" description:"Enable verbose logging"`
	New      NewCommand      `command:"new" description:"Start a new session"`
	Continue ContinueCommand `command:"continue" description:"Continue an existing session"`
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

	command := parser.Active
	switch command.Name {
	case "new":
		cmdRunner.handleNewCommand(&opts.New)
	case "continue":
		cmdRunner.handleContinueCommand(&opts.Continue)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command\n")
		os.Exit(1)
	}
}

func (cr *CmdRunner) handleNewCommand(cmd *NewCommand) {
	_ = cmd
	session := cr.sessionService.GenerateSession()
	fmt.Println(session.ID)
	
	output, err := cr.claudeClient.StartNewSession("hello")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting Claude session: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println(output)
}

func (cr *CmdRunner) handleContinueCommand(cmd *ContinueCommand) {
	if cmd.SessionID == "" {
		fmt.Fprintln(os.Stderr, "Session ID is required for continue command")
		os.Exit(1)
	}

	output, err := cr.claudeClient.ContinueSession(cmd.SessionID, cmd.Prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing Claude command: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(output)
}

