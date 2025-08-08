# Discord Bot Example

This is a simple Discord bot written in Go using the [DiscordGo](https://github.com/bwmarrin/discordgo) library. The bot demonstrates how to:

- Detect when the bot is mentioned in top-level channel messages
- Create public threads from messages that mention the bot
- Add emoji reactions to messages (thumbs up 👍)
- Respond to mentions in the created thread
- Handle additional commands like `!greet`

## Features

1. **Top-Level Mention Detection**: The bot detects when it's mentioned using `@BotName` in top-level channel messages (ignores mentions in existing threads)
2. **Thread Creation**: Automatically creates a public thread from the message that mentions the bot
3. **Automatic Reactions**: Adds a thumbs up (👍) emoji reaction to the original message
4. **Thread Response**: Replies with a personalized message in the newly created thread
5. **Additional Commands**: Supports a `!greet @user` command with wave emoji reaction (only in top-level channels)

## Prerequisites

1. **Go 1.22+**: Make sure you have Go installed
2. **Discord Application**: You need to create a Discord application and bot

## Setup Instructions

### 1. Create a Discord Application

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application" and give it a name
3. Go to the "Bot" section in the sidebar
4. Click "Add Bot"
5. Copy the bot token (you'll need this for the `DISCORD_BOT_TOKEN` environment variable)

### 2. Set Bot Permissions

In the "Bot" section, scroll down to "Privileged Gateway Intents" and enable:
- MESSAGE CONTENT INTENT (required to read message content)

### 3. Invite Bot to Server

1. Go to the "OAuth2" > "URL Generator" section
2. Select "bot" scope
3. Select the following bot permissions:
   - Send Messages
   - Add Reactions
   - Read Message History
   - View Channels
   - Create Public Threads
   - Send Messages in Threads
4. Copy the generated URL and open it in your browser to invite the bot to your server

### 4. Install Dependencies

```bash
cd examples/discord-bot
go mod tidy
```

### 5. Set Environment Variable

Set your bot token as an environment variable:

```bash
# Linux/macOS
export DISCORD_BOT_TOKEN="your_bot_token_here"

# Windows
set DISCORD_BOT_TOKEN=your_bot_token_here
```

## Running the Bot

```bash
go run main.go
```

You should see:
```
Discord bot is now running. Press CTRL-C to exit.
```

## Usage Examples

### Basic Mention (Top-Level Channel)
Type in any Discord channel where the bot has access (NOT in an existing thread):
```
@YourBot hello there!
```

**Result:**
- The bot adds a 👍 reaction to your message
- The bot creates a public thread named "Chat with YourUsername" from your message
- The bot replies in the thread: "Hello @YourUsername! You mentioned me in the main channel, so I created this thread for our conversation. What can I help you with?"

### Mention in Thread (Ignored)
If you mention the bot in an existing thread:
```
@YourBot hello there!
```

**Result:**
- The bot ignores the mention (no reaction, no response, no new thread created)
- This prevents thread spam and keeps conversations organized

### Greet Command (Top-Level Only)
```
!greet @SomeUser
```

**Result:**
- The bot adds a 👋 reaction to your message  
- The bot replies: "Hello @SomeUser! 👋 You were greeted by @YourUsername"
- Note: This command only works in top-level channels, not in threads

## Code Structure

### Main Components

1. **Bot Setup**: Creates Discord session with bot token
2. **Event Handler**: `messageCreate` function handles incoming messages
3. **Channel Type Check**: Uses `s.Channel()` to determine if message is in a thread
4. **Mention Detection**: Loops through `m.Mentions` to check if bot was mentioned
5. **Thread Creation**: Uses `MessageThreadStart` to create public threads
6. **Reaction Adding**: Uses `MessageReactionAdd` with Unicode emoji
7. **Thread Response**: Uses `ChannelMessageSend` to reply in the created thread

### Key DiscordGo Functions Used

- `discordgo.New("Bot " + token)` - Create Discord session
- `s.AddHandler(messageCreate)` - Register message event handler
- `s.Channel(channelID)` - Get channel information to check channel type
- `s.MessageThreadStart(channelID, messageID, name, duration)` - Create public thread from message
- `s.MessageReactionAdd(channelID, messageID, emoji)` - Add emoji reaction
- `s.ChannelMessageSend(channelID, message)` - Send message to channel or thread
- `m.Mentions` - Array of users mentioned in the message
- `m.Author.Bot` - Check if message author is a bot
- `isThreadChannel()` - Custom helper function to detect thread channel types

### Bot Safety Features

- **Bot Detection**: Ignores messages from other bots (including itself) using `m.Author.Bot`
- **Thread Spam Prevention**: Only responds to mentions in top-level channels, ignoring mentions in existing threads
- **Error Handling**: Logs errors for reactions, thread creation, and message sending
- **Fallback Response**: If thread creation fails, sends a fallback response in the original channel
- **Graceful Shutdown**: Handles CTRL-C to cleanly close the Discord connection

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DISCORD_BOT_TOKEN` | Your Discord bot's token from the Developer Portal | Yes |

## Troubleshooting

### Common Issues

1. **"Error creating Discord session"**: Check that your bot token is correct and properly set
2. **"Error opening connection"**: Verify your bot token and internet connection
3. **Bot doesn't respond**: Ensure the bot has proper permissions and MESSAGE CONTENT INTENT is enabled
4. **Reactions not working**: Check that the bot has "Add Reactions" permission in the channel
5. **Thread creation fails**: Ensure the bot has "Create Public Threads" and "Send Messages in Threads" permissions
6. **Bot ignores mentions**: Verify the mention is in a top-level channel, not an existing thread

### Debug Logs

The bot logs important events:
- When it's mentioned in top-level channel: `Bot mentioned by {username} in top-level channel {channelID}, creating thread`
- When it's mentioned in thread: `Bot mentioned in thread {channelID}, ignoring as per configuration`
- Thread creation: `Created thread '{name}' with ID {threadID}`
- Errors with reactions: `Error adding reaction: {error}`
- Errors with thread creation: `Error creating thread: {error}`
- Errors with messages: `Error sending message to thread: {error}`

## Extending the Bot

You can easily extend this bot by:

1. **Adding more commands**: Check for different prefixes in the message content
2. **More emoji reactions**: Use different Unicode emoji characters
3. **Database integration**: Store user interactions or preferences
4. **Slash commands**: Implement modern Discord slash commands
5. **Multiple server support**: Handle different server configurations

## Dependencies

- [github.com/bwmarrin/discordgo](https://github.com/bwmarrin/discordgo) v0.28.1 - Discord API bindings for Go

## License

This example is provided as-is for educational purposes.