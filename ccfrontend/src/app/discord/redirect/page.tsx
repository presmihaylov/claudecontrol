"use client";

import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";

function DiscordRedirectContent() {
	const router = useRouter();
	const searchParams = useSearchParams();
	const { getToken } = useAuth();
	const [status, setStatus] = useState<"processing" | "success" | "error">("processing");
	const [errorMessage, setErrorMessage] = useState<string>("");

	useEffect(() => {
		const handleDiscordIntegration = async () => {
			try {
				// Get the OAuth parameters from URL
				const code = searchParams.get("code");
				const guildId = searchParams.get("guild_id");
				const error = searchParams.get("error");

				if (error) {
					setStatus("error");
					setErrorMessage("Discord authorization was denied or failed.");
					return;
				}

				if (!code) {
					setStatus("error");
					setErrorMessage("No authorization code received from Discord.");
					return;
				}

				if (!guildId) {
					setStatus("error");
					setErrorMessage("No guild ID received from Discord.");
					return;
				}

				// Get authentication token
				const token = await getToken();
				if (!token) {
					setStatus("error");
					setErrorMessage("Authentication required. Please sign in.");
					return;
				}

				// Call backend API to create Discord integration
				const response = await fetch(`${env.CCBACKEND_BASE_URL}/discord/integrations`, {
					method: "POST",
					headers: {
						Authorization: `Bearer ${token}`,
						"Content-Type": "application/json",
					},
					body: JSON.stringify({
						code: code,
						guild_id: guildId,
						redirect_url: window.location.origin + "/discord/redirect",
					}),
				});

				if (!response.ok) {
					const errorData = await response.text();
					setStatus("error");
					setErrorMessage(`Failed to create Discord integration: ${errorData}`);
					return;
				}

				const integration = await response.json();
				console.log("Discord integration created successfully:", integration);
				setStatus("success");

				// Redirect to home page after a short delay
				setTimeout(() => {
					router.push("/");
				}, 2000);
			} catch (error) {
				console.error("Error creating Discord integration:", error);
				setStatus("error");
				setErrorMessage("An unexpected error occurred. Please try again.");
			}
		};

		handleDiscordIntegration();
	}, [searchParams, getToken, router]);

	if (status === "processing") {
		return (
			<div className="flex items-center justify-center min-h-screen">
				<div className="text-center">
					<h1 className="text-2xl font-bold mb-4">Processing...</h1>
					<p className="text-muted-foreground">Setting up your Discord integration. Please wait...</p>
				</div>
			</div>
		);
	}

	if (status === "success") {
		return (
			<div className="flex items-center justify-center min-h-screen">
				<div className="text-center">
					<h1 className="text-2xl font-bold mb-4 text-green-600">Success!</h1>
					<p className="text-muted-foreground">
						Your Discord integration has been set up successfully. Redirecting you back...
					</p>
				</div>
			</div>
		);
	}

	return (
		<div className="flex items-center justify-center min-h-screen">
			<div className="text-center">
				<h1 className="text-2xl font-bold mb-4 text-red-600">Error</h1>
				<p className="text-muted-foreground mb-4">{errorMessage}</p>
				<button
					onClick={() => router.push("/")}
					className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 cursor-pointer"
					type="button"
				>
					Return to Home
				</button>
			</div>
		</div>
	);
}

export default function DiscordRedirect() {
	return (
		<Suspense
			fallback={
				<div className="flex items-center justify-center min-h-screen">
					<div className="text-center">
						<h1 className="text-2xl font-bold mb-4">Loading...</h1>
						<p className="text-muted-foreground">Preparing your Discord integration...</p>
					</div>
				</div>
			}
		>
			<DiscordRedirectContent />
		</Suspense>
	);
}