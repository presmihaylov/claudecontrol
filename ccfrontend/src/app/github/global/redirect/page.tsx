"use client";

import { useEffect } from "react";

export default function GitHubGlobalRedirect() {
	useEffect(() => {
		// Parse query parameters from current URL
		const urlParams = new URLSearchParams(window.location.search);
		const state = urlParams.get("state");
		
		if (!state) {
			console.error("No state parameter found in GitHub callback");
			window.location.href = "/";
			return;
		}

		// Parse the state parameter to extract redirect_uri
		let targetRedirectUri: string;
		try {
			// The state format is "redirect_uri=<encoded_uri>"
			if (state.startsWith("redirect_uri=")) {
				const encodedUri = state.substring("redirect_uri=".length);
				targetRedirectUri = decodeURIComponent(encodedUri);
			} else {
				throw new Error("Invalid state format");
			}
		} catch (error) {
			console.error("Failed to parse state parameter:", error);
			window.location.href = "/";
			return;
		}

		// Create new URL with the target redirect URI
		const targetUrl = new URL(targetRedirectUri);
		
		// Preserve all query parameters except 'state' and add them to target URL
		urlParams.delete("state");
		for (const [key, value] of urlParams.entries()) {
			targetUrl.searchParams.set(key, value);
		}

		// Redirect to the target URL with all preserved parameters
		console.log("Redirecting to:", targetUrl.toString());
		window.location.href = targetUrl.toString();
	}, []);

	return (
		<div className="flex items-center justify-center min-h-screen">
			<div className="text-center">
				<div className="animate-pulse">
					<div className="h-8 w-32 bg-muted rounded mb-4 mx-auto" />
					<div className="h-4 w-48 bg-muted rounded mx-auto" />
				</div>
				<p className="mt-4 text-muted-foreground">Processing GitHub installation...</p>
			</div>
		</div>
	);
}