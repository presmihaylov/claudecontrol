"use client";

import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";

function SlackRedirectContent() {
	const router = useRouter();
	const searchParams = useSearchParams();
	const { getToken } = useAuth();
	const [status, setStatus] = useState<"processing" | "success" | "error">("processing");
	const [errorMessage, setErrorMessage] = useState<string>("");

	useEffect(() => {
		const handleSlackIntegration = async () => {
			try {
				// Get the OAuth code from URL parameters
				const code = searchParams.get("code");
				const error = searchParams.get("error");

				if (error) {
					setStatus("error");
					setErrorMessage("Slack authorization was denied or failed.");
					return;
				}

				if (!code) {
					setStatus("error");
					setErrorMessage("No authorization code received from Slack.");
					return;
				}

				// Get authentication token
				const token = await getToken();
				if (!token) {
					setStatus("error");
					setErrorMessage("Authentication required. Please sign in.");
					return;
				}

				// Call backend API to create Slack integration
				const response = await fetch(`${env.CCBACKEND_BASE_URL}/slack/integrations`, {
					method: "POST",
					headers: {
						Authorization: `Bearer ${token}`,
						"Content-Type": "application/json",
					},
					body: JSON.stringify({
						slackAuthToken: code,
						redirectUrl: window.location.origin + "/slack/redirect",
					}),
				});

				if (!response.ok) {
					const errorData = await response.text();
					setStatus("error");
					setErrorMessage(`Failed to create Slack integration: ${errorData}`);
					return;
				}

				const integration = await response.json();
				console.log("Slack integration created successfully:", integration);
				setStatus("success");

				// Redirect to integration page after a short delay
				setTimeout(() => {
					router.push(`/integrations/${integration.id}`);
				}, 2000);
			} catch (error) {
				console.error("Error creating Slack integration:", error);
				setStatus("error");
				setErrorMessage("An unexpected error occurred. Please try again.");
			}
		};

		handleSlackIntegration();
	}, [searchParams, getToken, router]);

	if (status === "processing") {
		return (
			<div className="flex items-center justify-center min-h-screen">
				<div className="text-center">
					<h1 className="text-2xl font-bold mb-4">Processing...</h1>
					<p className="text-muted-foreground">Setting up your Slack integration. Please wait...</p>
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
						Your Slack integration has been set up successfully. Redirecting you back...
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
					className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
					type="button"
				>
					Return to Home
				</button>
			</div>
		</div>
	);
}

export default function SlackRedirect() {
	return (
		<Suspense
			fallback={
				<div className="flex items-center justify-center min-h-screen">
					<div className="text-center">
						<h1 className="text-2xl font-bold mb-4">Loading...</h1>
						<p className="text-muted-foreground">Preparing your Slack integration...</p>
					</div>
				</div>
			}
		>
			<SlackRedirectContent />
		</Suspense>
	);
}
