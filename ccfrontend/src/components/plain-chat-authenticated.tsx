"use client";

import { useAuth } from "@clerk/nextjs";
import { useEffect, useState } from "react";

declare global {
	interface Window {
		Plain?: {
			init: (config: {
				appId: string;
				customerDetails?: {
					email: string;
					emailHash: string;
					fullName: string;
					shortName: string;
				};
				theme?: string;
				style?: {
					brandColor?: string;
					brandBackgroundColor?: string;
					launcherBackgroundColor?: string;
					launcherIconColor?: string;
				};
				position?: {
					right?: string;
					bottom?: string;
				};
				links?: Array<{
					icon: string;
					text: string;
					url: string;
				}>;
			}) => void;
		};
	}
}

interface ChatAuthData {
	email: string;
	emailHash: string;
	fullName: string;
	shortName: string;
}

export default function PlainChatAuthenticated() {
	const { isSignedIn } = useAuth();
	const [_isLoading, setIsLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		if (!isSignedIn) {
			setIsLoading(false);
			return;
		}

		let scriptAdded = false;

		const initializeChat = async () => {
			try {
				// Fetch chat authentication data
				const response = await fetch("/api/user/chat-auth");

				if (!response.ok) {
					throw new Error("Failed to get chat authentication data");
				}

				const authData: ChatAuthData = await response.json();

				// Load Plain chat script if not already loaded
				if (!document.querySelector('script[src="https://chat.cdn-plain.com/index.js"]')) {
					const script = document.createElement("script");
					script.async = false;
					script.onload = () => {
						if (window.Plain) {
							window.Plain.init({
								appId: "liveChatApp_01K2D7E0M9ZWWB6CZXBT1VX54X",

								// Manual email verification with authenticated user
								customerDetails: {
									email: authData.email,
									emailHash: authData.emailHash,
									fullName: authData.fullName,
									shortName: authData.shortName,
								},

								// Styling with black background and white text
								theme: "dark",
								style: {
									brandColor: "#ffffff",
									brandBackgroundColor: "#000000",
									launcherBackgroundColor: "#000000",
									launcherIconColor: "#ffffff",
								},

								// Position the chat widget
								position: {
									right: "20px",
									bottom: "20px",
								},

								// Add helpful links
								links: [
									{
										icon: "email",
										text: "Email Support",
										url: "mailto:support@pmihaylov.com",
									},
								],
							});
						}
					};
					script.onerror = () => {
						setError("Failed to load chat widget");
					};
					script.src = "https://chat.cdn-plain.com/index.js";
					document.getElementsByTagName("head")[0].appendChild(script);
					scriptAdded = true;
				}

				setIsLoading(false);
			} catch (err) {
				console.error("Error initializing chat:", err);
				setError("Failed to initialize chat");
				setIsLoading(false);
			}
		};

		initializeChat();

		// Cleanup on unmount
		return () => {
			if (scriptAdded) {
				const existingScript = document.querySelector(
					'script[src="https://chat.cdn-plain.com/index.js"]',
				);
				if (existingScript) {
					existingScript.remove();
				}
			}
		};
	}, [isSignedIn]);

	// Don't render anything if user is not signed in or if there's an error
	if (!isSignedIn || error) {
		return null;
	}

	return null; // This component doesn't render anything visible
}
