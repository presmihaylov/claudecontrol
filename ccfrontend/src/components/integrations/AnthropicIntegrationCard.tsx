"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { CheckCircle, ExternalLink, Key, Loader2, Trash2, User } from "lucide-react";
import { useEffect, useState } from "react";

interface AnthropicIntegration {
	id: string;
	has_api_key: boolean;
	has_oauth_token: boolean;
	organization_id: string;
	created_at: string;
	updated_at: string;
}

interface AnthropicIntegrationCardProps {
	onIntegrationChange?: (integration: AnthropicIntegration | null) => void;
}

// OAuth configuration from ccoauth example
const CLAUDE_AUTH_URL = "https://claude.ai/oauth/authorize";
const CLAUDE_CLIENT_ID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e";
const CLAUDE_REDIRECT_URI = "https://console.anthropic.com/oauth/code/callback";
const CLAUDE_SCOPE = "org:create_api_key user:profile user:inference";

// PKCE helper functions
function randomURLSafe(n: number): string {
	const array = new Uint8Array(n);
	crypto.getRandomValues(array);
	return btoa(String.fromCharCode(...array))
		.replace(/\+/g, "-")
		.replace(/\//g, "_")
		.replace(/=+$/, "");
}

async function pkceS256(verifier: string): Promise<string> {
	const encoder = new TextEncoder();
	const data = encoder.encode(verifier);
	const hash = await crypto.subtle.digest("SHA-256", data);
	return btoa(String.fromCharCode(...new Uint8Array(hash)))
		.replace(/\+/g, "-")
		.replace(/\//g, "_")
		.replace(/=+$/, "");
}

export function AnthropicIntegrationCard({ onIntegrationChange }: AnthropicIntegrationCardProps) {
	const { getToken } = useAuth();
	const [integration, setIntegration] = useState<AnthropicIntegration | null>(null);
	const [loading, setLoading] = useState(true);
	const [deleting, setDeleting] = useState(false);
	const [saving, setSaving] = useState(false);
	const [error, setError] = useState<string | null>(null);

	// Form states
	const [apiKey, setApiKey] = useState("");
	const [oauthCode, setOauthCode] = useState("");
	const [integrationMethod, setIntegrationMethod] = useState<"api-key" | "oauth">("api-key");

	useEffect(() => {
		checkIntegrationStatus();
	}, []);

	const checkIntegrationStatus = async () => {
		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/anthropic/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			if (response.ok) {
				const integrations: AnthropicIntegration[] = await response.json();
				const currentIntegration = integrations.length > 0 ? integrations[0] : null;
				setIntegration(currentIntegration);
				onIntegrationChange?.(currentIntegration);
			}
		} catch (err) {
			console.error("Error checking Anthropic integration:", err);
			setError("Failed to load Anthropic integration status");
		} finally {
			setLoading(false);
		}
	};

	const buildClaudeOAuthURL = async () => {
		const codeVerifier = randomURLSafe(64);
		const codeChallenge = await pkceS256(codeVerifier);
		const state = randomURLSafe(24);

		// Store verifier for later use
		sessionStorage.setItem("claude_code_verifier", codeVerifier);
		sessionStorage.setItem("claude_state", state);

		const params = new URLSearchParams({
			code: "true",
			response_type: "code",
			client_id: CLAUDE_CLIENT_ID,
			redirect_uri: CLAUDE_REDIRECT_URI,
			scope: CLAUDE_SCOPE,
			state: state,
			code_challenge: codeChallenge,
			code_challenge_method: "S256",
		});

		return `${CLAUDE_AUTH_URL}?${params.toString()}`;
	};

	const handleOpenClaudeOAuth = async () => {
		const url = await buildClaudeOAuthURL();
		window.open(url, "_blank");
	};

	const handleSaveAnthropic = async () => {
		setSaving(true);
		setError(null);

		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			const body: { api_key?: string; oauth_token?: string; code_verifier?: string } = {};
			if (integrationMethod === "api-key") {
				if (!apiKey.trim()) {
					setError("Please enter an API key");
					return;
				}
				body.api_key = apiKey.trim();
			} else {
				if (!oauthCode.trim()) {
					setError("Please paste the OAuth code");
					return;
				}
				// Get the code verifier from sessionStorage
				const codeVerifier = sessionStorage.getItem("claude_code_verifier");
				if (!codeVerifier) {
					setError("Code verifier not found. Please restart the OAuth flow.");
					return;
				}
				body.oauth_token = oauthCode.trim();
				body.code_verifier = codeVerifier;
			}

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/anthropic/integrations`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${token}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify(body),
			});

			if (!response.ok) {
				const errorText = await response.text();
				throw new Error(errorText || "Failed to create Anthropic integration");
			}

			const newIntegration: AnthropicIntegration = await response.json();
			setIntegration(newIntegration);
			onIntegrationChange?.(newIntegration);
			setApiKey("");
			setOauthCode("");
			setError(null);
		} catch (err) {
			console.error("Error saving Anthropic integration:", err);
			setError(err instanceof Error ? err.message : "Failed to save Anthropic integration");
		} finally {
			setSaving(false);
		}
	};

	const handleDisconnectAnthropic = async () => {
		if (!integration) return;

		const confirmed = window.confirm(
			"Are you sure you want to disconnect this Anthropic integration?",
		);

		if (!confirmed) return;

		setDeleting(true);
		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/anthropic/integrations/${integration.id}`,
				{
					method: "DELETE",
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			);

			if (!response.ok) {
				throw new Error("Failed to disconnect Anthropic integration");
			}

			setIntegration(null);
			onIntegrationChange?.(null);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting Anthropic integration:", err);
			setError("Failed to disconnect Anthropic integration");
		} finally {
			setDeleting(false);
		}
	};

	if (loading) {
		return (
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<User className="h-5 w-5" />
						Anthropic Account
					</CardTitle>
				</CardHeader>
				<CardContent>
					<div className="flex items-center justify-center py-4">
						<div className="animate-pulse text-sm text-muted-foreground">Loading...</div>
					</div>
				</CardContent>
			</Card>
		);
	}

	if (error && integration) {
		return (
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<User className="h-5 w-5" />
						Anthropic Account
					</CardTitle>
				</CardHeader>
				<CardContent>
					<div className="text-sm text-destructive mb-4">{error}</div>
					<Button onClick={() => setError(null)} variant="outline">
						Try Again
					</Button>
				</CardContent>
			</Card>
		);
	}

	if (!integration) {
		return (
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<User className="h-5 w-5" />
						Anthropic Account
					</CardTitle>
					<CardDescription>Connect your Anthropic account to use Claude Code</CardDescription>
				</CardHeader>
				<CardContent>
					{error && <div className="text-sm text-destructive mb-4">{error}</div>}
					<Tabs
						value={integrationMethod}
						onValueChange={(v) => setIntegrationMethod(v as "api-key" | "oauth")}
					>
						<TabsList className="grid w-full grid-cols-2">
							<TabsTrigger value="api-key">API Key</TabsTrigger>
							<TabsTrigger value="oauth">Claude Pro/Max Plan</TabsTrigger>
						</TabsList>
						<TabsContent value="api-key" className="space-y-4">
							<div className="space-y-2">
								<Label htmlFor="api-key">Anthropic API Key</Label>
								<Input
									id="api-key"
									type="password"
									placeholder="sk-ant-..."
									value={apiKey}
									onChange={(e) => setApiKey(e.target.value)}
								/>
								<p className="text-xs text-muted-foreground">
									You can find your API key in the{" "}
									<a
										href="https://console.anthropic.com/settings/keys"
										target="_blank"
										rel="noopener noreferrer"
										className="underline"
									>
										Anthropic Console
									</a>
								</p>
							</div>
							<Button
								onClick={handleSaveAnthropic}
								disabled={!apiKey.trim() || saving}
								className="w-full"
							>
								{saving ? (
									<>
										<Loader2 className="mr-2 h-4 w-4 animate-spin" />
										Saving...
									</>
								) : (
									<>
										<Key className="mr-2 h-4 w-4" />
										Save API Key
									</>
								)}
							</Button>
						</TabsContent>
						<TabsContent value="oauth" className="space-y-4">
							<div className="space-y-4">
								<div className="rounded-lg border bg-muted/50 p-4">
									<p className="text-sm">
										<strong>Step 1:</strong> Click the button below to authorize Claude Control
									</p>
									<Button onClick={handleOpenClaudeOAuth} className="mt-2 w-full" variant="outline">
										<ExternalLink className="mr-2 h-4 w-4" />
										Open Claude OAuth
									</Button>
								</div>
								<div className="space-y-2">
									<Label htmlFor="oauth-code">
										<strong>Step 2:</strong> Paste the code from the redirect URL
									</Label>
									<Input
										id="oauth-code"
										type="text"
										placeholder="Paste the code parameter from the URL"
										value={oauthCode}
										onChange={(e) => setOauthCode(e.target.value)}
									/>
									<p className="text-xs text-muted-foreground">
										After authorizing, you'll be redirected to a URL with a <code>code</code>{" "}
										parameter. Copy and paste that code here.
									</p>
								</div>
								<Button
									onClick={handleSaveAnthropic}
									disabled={!oauthCode.trim() || saving}
									className="w-full"
								>
									{saving ? (
										<>
											<Loader2 className="mr-2 h-4 w-4 animate-spin" />
											Saving...
										</>
									) : (
										<>
											<User className="mr-2 h-4 w-4" />
											Save OAuth Token
										</>
									)}
								</Button>
							</div>
						</TabsContent>
					</Tabs>
				</CardContent>
			</Card>
		);
	}

	return (
		<Card>
			<CardHeader>
				<CardTitle className="flex items-center gap-2">
					<User className="h-5 w-5" />
					Anthropic Account
				</CardTitle>
			</CardHeader>
			<CardContent className="space-y-4">
				<div className="rounded-lg border bg-muted/50 p-4">
					<div className="flex items-start justify-between">
						<div className="flex-1">
							<dl className="space-y-1 text-sm">
								<div>
									<dt className="inline font-medium text-muted-foreground">Type:</dt>{" "}
									<dd className="inline">{integration.has_api_key ? "API Key" : "OAuth Token"}</dd>
								</div>
								<div>
									<dt className="inline font-medium text-muted-foreground">Connected:</dt>{" "}
									<dd className="inline">
										{new Date(integration.created_at).toLocaleDateString()}
									</dd>
								</div>
							</dl>
						</div>
						<Button
							variant="ghost"
							size="sm"
							onClick={handleDisconnectAnthropic}
							disabled={deleting}
							className="text-muted-foreground hover:text-destructive"
						>
							<Trash2 className="h-4 w-4 mr-2" />
							{deleting ? "Disconnecting..." : "Disconnect"}
						</Button>
					</div>
				</div>
			</CardContent>
		</Card>
	);
}
