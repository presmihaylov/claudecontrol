package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// Get bot token from environment variable
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("No DISCORD_BOT_TOKEN environment variable provided. Please set it to your bot's token.")
	}

	// Create a new Discord session using the provided bot token
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events
	dg.AddHandler(messageCreate)

	// We only care about receiving message events
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

	// Open a websocket connection to Discord and begin listening
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}

	// Wait here until CTRL-C or other term signal is received
	fmt.Println("Discord bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session
	dg.Close()
}

// messageCreate handles incoming messages
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots (including this bot)
	if m.Author.Bot {
		return
	}

	// Check if the bot was mentioned
	botMentioned := false
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			botMentioned = true
			break
		}
	}

	// If the bot was mentioned, respond and add thumbs up reaction
	if botMentioned {
		// Add thumbs up emoji reaction to the message
		err := s.MessageReactionAdd(m.ChannelID, m.ID, "👍")
		if err != nil {
			log.Printf("Error adding reaction: %v", err)
		}

		// Respond to the user who mentioned the bot
		responseMessage := fmt.Sprintf("Hello %s! You mentioned me. Thanks for the message!", m.Author.Mention())
		_, err = s.ChannelMessageSend(m.ChannelID, responseMessage)
		if err != nil {
			log.Printf("Error sending message: %v", err)
		}

		log.Printf("Bot mentioned by %s in channel %s", m.Author.Username, m.ChannelID)
	}

	// Optional: Handle additional commands with mentions
	if strings.HasPrefix(strings.ToLower(m.Content), "!greet") && len(m.Mentions) > 0 {
		// Greet the first mentioned user
		mentionedUser := m.Mentions[0]
		greetMessage := fmt.Sprintf("Hello %s! 👋 You were greeted by %s", 
			mentionedUser.Mention(), m.Author.Mention())
		
		_, err := s.ChannelMessageSend(m.ChannelID, greetMessage)
		if err != nil {
			log.Printf("Error sending greet message: %v", err)
		}

		// Add wave emoji reaction to the original message
		err = s.MessageReactionAdd(m.ChannelID, m.ID, "👋")
		if err != nil {
			log.Printf("Error adding wave reaction: %v", err)
		}
	}
}