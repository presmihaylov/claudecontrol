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

	// If the bot was mentioned, check if it's a top-level message and create thread
	if botMentioned {
		// Get channel information to check if message is in a thread
		channel, err := s.Channel(m.ChannelID)
		if err != nil {
			log.Printf("Error getting channel info: %v", err)
			return
		}

		// Check if the message is in a thread - if so, ignore it
		if isThreadChannel(channel.Type) {
			log.Printf("Bot mentioned in thread %s, ignoring as per configuration", m.ChannelID)
			return
		}

		// Message is top-level, proceed with thread creation
		log.Printf("Bot mentioned by %s in top-level channel %s, creating thread", m.Author.Username, m.ChannelID)

		// Add thumbs up emoji reaction to the original message
		err = s.MessageReactionAdd(m.ChannelID, m.ID, "👍")
		if err != nil {
			log.Printf("Error adding reaction: %v", err)
		}

		// Create a public thread from the message that mentioned the bot
		threadName := fmt.Sprintf("Chat with %s", m.Author.Username)
		thread, err := s.MessageThreadStart(m.ChannelID, m.ID, threadName, 1440) // 1440 minutes = 24 hours
		if err != nil {
			log.Printf("Error creating thread: %v", err)
			// Fallback: send response in the original channel
			responseMessage := fmt.Sprintf("Hello %s! You mentioned me. I tried to create a thread but couldn't. Thanks for the message!", m.Author.Mention())
			_, err = s.ChannelMessageSend(m.ChannelID, responseMessage)
			if err != nil {
				log.Printf("Error sending fallback message: %v", err)
			}
			return
		}

		log.Printf("Created thread '%s' with ID %s", thread.Name, thread.ID)

		// Send response in the newly created thread
		responseMessage := fmt.Sprintf("Hello %s! You mentioned me in the main channel, so I created this thread for our conversation. What can I help you with?", m.Author.Mention())
		_, err = s.ChannelMessageSend(thread.ID, responseMessage)
		if err != nil {
			log.Printf("Error sending message to thread: %v", err)
		}
	}

	// Optional: Handle additional commands with mentions (only in non-thread channels)
	if strings.HasPrefix(strings.ToLower(m.Content), "!greet") && len(m.Mentions) > 0 {
		// Get channel information to check if message is in a thread
		channel, err := s.Channel(m.ChannelID)
		if err != nil {
			log.Printf("Error getting channel info for greet command: %v", err)
			return
		}

		// Only process greet command in top-level channels
		if !isThreadChannel(channel.Type) {
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
}

// isThreadChannel checks if the given channel type is a thread
func isThreadChannel(channelType discordgo.ChannelType) bool {
	return channelType == discordgo.ChannelTypeGuildPublicThread ||
		   channelType == discordgo.ChannelTypeGuildPrivateThread ||
		   channelType == discordgo.ChannelTypeGuildNewsThread
}