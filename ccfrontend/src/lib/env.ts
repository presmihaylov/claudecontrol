// Environment variable validation
const requiredEnvVars = {
	NEXT_PUBLIC_SLACK_CLIENT_ID: process.env.NEXT_PUBLIC_SLACK_CLIENT_ID,
	NEXT_PUBLIC_SLACK_REDIRECT_URI: process.env.NEXT_PUBLIC_SLACK_REDIRECT_URI,
	NEXT_PUBLIC_DISCORD_REDIRECT_URI: process.env.NEXT_PUBLIC_DISCORD_REDIRECT_URI,
	NEXT_PUBLIC_CCBACKEND_BASE_URL: process.env.NEXT_PUBLIC_CCBACKEND_BASE_URL,
	PLAIN_CHAT_SECRET: process.env.PLAIN_CHAT_SECRET,
} as const;

// Validate all required environment variables
for (const [key, value] of Object.entries(requiredEnvVars)) {
	if (!value) {
		throw new Error(`Missing required environment variable: ${key}`);
	}
}

// Export validated environment variables
export const env = {
	SLACK_CLIENT_ID: requiredEnvVars.NEXT_PUBLIC_SLACK_CLIENT_ID as string,
	SLACK_REDIRECT_URI: requiredEnvVars.NEXT_PUBLIC_SLACK_REDIRECT_URI as string,
	DISCORD_REDIRECT_URI: requiredEnvVars.NEXT_PUBLIC_DISCORD_REDIRECT_URI as string,
	CCBACKEND_BASE_URL: requiredEnvVars.NEXT_PUBLIC_CCBACKEND_BASE_URL as string,
	PLAIN_CHAT_SECRET: requiredEnvVars.PLAIN_CHAT_SECRET as string,
} as const;
