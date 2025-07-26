"use client";

import { Button } from "@/components/ui/button";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { Slack } from "lucide-react";
import { useEffect } from "react";

export default function Home() {
	const { isLoaded, isSignedIn, getToken } = useAuth();

	// Authenticate user with backend when they first sign in
	useEffect(() => {
		const authenticateUser = async () => {
			if (!isLoaded || !isSignedIn) return;

			try {
				const token = await getToken();
				if (!token) return;

				const response = await fetch(
					`${env.CCBACKEND_BASE_URL}/users/authenticate`,
					{
						method: "POST",
						headers: {
							Authorization: `Bearer ${token}`,
							"Content-Type": "application/json",
						},
					},
				);

				if (!response.ok) {
					console.error("Failed to authenticate user:", response.statusText);
					return;
				}

				const user = await response.json();
				console.log("User authenticated successfully:", user);
			} catch (error) {
				console.error("Error authenticating user:", error);
			}
		};

		authenticateUser();
	}, [isLoaded, isSignedIn, getToken]);

	const handleAddToSlack = () => {
		const scope =
			"app_mentions:read,channels:history,chat:write,commands,reactions:write,reactions:read,team:read";
		const userScope = "";

		const slackAuthUrl = `https://slack.com/oauth/v2/authorize?client_id=${env.SLACK_CLIENT_ID}&scope=${encodeURIComponent(scope)}&user_scope=${encodeURIComponent(userScope)}&redirect_uri=${encodeURIComponent(env.SLACK_REDIRECT_URI)}`;

		window.location.href = slackAuthUrl;
	};

	if (!isLoaded) {
		return (
			<div className="flex items-center justify-center min-h-screen">
				<div className="animate-pulse">
					<div className="h-8 w-32 bg-muted rounded mb-4" />
					<div className="h-4 w-48 bg-muted rounded" />
				</div>
			</div>
		);
	}

	if (!isSignedIn) {
		return (
			<div className="flex items-center justify-center min-h-screen">
				<div className="text-muted-foreground">Redirecting to sign in...</div>
			</div>
		);
	}

	return (
		<div className="flex flex-col items-center justify-center min-h-[80vh] px-4">
			<h1 className="text-4xl font-bold text-center mb-8">
				Welcome to Claude Control
			</h1>
			<Button
				size="lg"
				className="flex items-center gap-2 cursor-pointer"
				onClick={handleAddToSlack}
			>
				<Slack className="h-5 w-5" />
				Add to Slack
			</Button>
		</div>
	);
}
