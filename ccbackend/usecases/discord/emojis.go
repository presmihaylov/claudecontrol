package discord

// Discord Unicode emoji constants used throughout the Discord usecase
const (
	// Status emojis
	EmojiHourglass  = "⏳" // Processing/queued status
	EmojiEyes       = "👀" // Agent is looking at/processing the message
	EmojiCheckMark  = "✅" // Completed successfully
	EmojiRaisedHand = "✋" // Agent waiting for next steps
	EmojiCrossMark  = "❌" // Error/failed status

	// System message prefix
	EmojiGear = ":gear:" // System message indicator
)

// Emoji arrays for batch operations
var (
	// AllStatusEmojis contains all emojis used for message status reactions
	AllStatusEmojis = []string{
		EmojiHourglass,
		EmojiEyes,
		EmojiCheckMark,
		EmojiRaisedHand,
		EmojiCrossMark,
	}
)
