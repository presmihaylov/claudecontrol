"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { CheckCircle, ExternalLink, GitBranch, Key, Loader2, MessageCircle, Server, Trash2, User } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

interface GitHubIntegration {
	id: string;
	github_installation_id: string;
	organization_id: string;
	created_at: string;
	updated_at: string;
}

interface AnthropicIntegration {
	id: string;
	has_api_key: boolean;
	has_oauth_token: boolean;
	organization_id: string;
	created_at: string;
	updated_at: string;
}

interface CCAgentContainerIntegration {
	id: string;
	instances_count: number;
	repo_url: string;
	organization_id: string;
	created_at: string;
	updated_at: string;
}

interface GitHubRepository {
	id: number;
	name: string;
	full_name: string;
	html_url: string;
	description?: string;
	private: boolean;
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

export default function OnboardingPage() {
	const { getToken } = useAuth();
	const router = useRouter();
	const [loading, setLoading] = useState(true);
	const [currentStep, setCurrentStep] = useState(1);
	const [githubIntegration, setGithubIntegration] = useState<GitHubIntegration | null>(null);
	const [anthropicIntegration, setAnthropicIntegration] = useState<AnthropicIntegration | null>(
		null,
	);
	const [ccAgentIntegration, setCCAgentIntegration] = useState<CCAgentContainerIntegration | null>(
		null,
	);
	const [repositories, setRepositories] = useState<GitHubRepository[]>([]);
	const [error, setError] = useState<string | null>(null);

	// Anthropic integration state
	const [apiKey, setApiKey] = useState("");
	const [oauthCode, setOauthCode] = useState("");
	const [integrationMethod, setIntegrationMethod] = useState<"api-key" | "oauth">("api-key");
	const [savingAnthropic, setSavingAnthropic] = useState(false);

	// CCAgent integration state
	const [selectedRepo, setSelectedRepo] = useState("");
	const [instancesCount] = useState(1);
	const [savingCCAgent, setSavingCCAgent] = useState(false);
	const [loadingRepos, setLoadingRepos] = useState(false);

	// Check for existing integrations on mount
	useEffect(() => {
		checkExistingIntegrations();
	}, []);

	// Load repositories when reaching step 3
	useEffect(() => {
		if (currentStep === 3 && githubIntegration && repositories.length === 0) {
			loadGitHubRepositories();
		}
	}, [currentStep, githubIntegration]);

	const checkExistingIntegrations = async () => {
		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				setLoading(false);
				return;
			}

			// Check GitHub integration
			const githubResponse = await fetch(`${env.CCBACKEND_BASE_URL}/github/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			if (githubResponse.ok) {
				const githubIntegrations: GitHubIntegration[] = await githubResponse.json();
				if (githubIntegrations.length > 0) {
					setGithubIntegration(githubIntegrations[0]);
					setCurrentStep(2);
				}
			}

			// Check Anthropic integration
			const anthropicResponse = await fetch(`${env.CCBACKEND_BASE_URL}/anthropic/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			if (anthropicResponse.ok) {
				const anthropicIntegrations: AnthropicIntegration[] = await anthropicResponse.json();
				if (anthropicIntegrations.length > 0) {
					setAnthropicIntegration(anthropicIntegrations[0]);
					if (githubIntegration) {
						setCurrentStep(3);
					}
				}
			}

			// Check CCAgent Container integration
			const ccAgentResponse = await fetch(`${env.CCBACKEND_BASE_URL}/ccagent-container/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			if (ccAgentResponse.ok) {
				const integration = await ccAgentResponse.json();
				if (integration) {
					setCCAgentIntegration(integration);
					if (githubIntegration && anthropicIntegration) {
						setCurrentStep(4);
					}
				}
			}
		} catch (err) {
			console.error("Error checking existing integrations:", err);
			setError("Failed to check existing integrations");
		} finally {
			setLoading(false);
		}
	};

	const handleInstallGitHub = () => {
		const githubAppUrl = "https://github.com/apps/claude-control/installations/select_target";
		const redirectUri = `${window.location.origin}/github/redirect`;
		const state = `redirect_uri=${encodeURIComponent(redirectUri)}`;
		window.location.href = `${githubAppUrl}?state=${state}`;
	};

	const handleDisconnectGitHub = async () => {
		if (!githubIntegration) return;

		const confirmed = window.confirm(
			"Are you sure you want to disconnect this GitHub integration? This will remove access to all connected repositories.",
		);

		if (!confirmed) return;

		setLoading(true);
		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/github/integrations/${githubIntegration.id}`,
				{
					method: "DELETE",
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			);

			if (!response.ok) {
				throw new Error("Failed to disconnect GitHub integration");
			}

			setGithubIntegration(null);
			setCurrentStep(1);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting GitHub integration:", err);
			setError("Failed to disconnect GitHub integration");
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
		setSavingAnthropic(true);
		setError(null);

		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			const body: { api_key?: string; oauth_token?: string } = {};
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
				body.oauth_token = oauthCode.trim();
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

			const integration: AnthropicIntegration = await response.json();
			setAnthropicIntegration(integration);
			setCurrentStep(3);
			setApiKey("");
			setOauthCode("");
		} catch (err) {
			console.error("Error saving Anthropic integration:", err);
			setError(err instanceof Error ? err.message : "Failed to save Anthropic integration");
		} finally {
			setSavingAnthropic(false);
		}
	};

	const handleDisconnectAnthropic = async () => {
		if (!anthropicIntegration) return;

		const confirmed = window.confirm(
			"Are you sure you want to disconnect this Anthropic integration?",
		);

		if (!confirmed) return;

		setLoading(true);
		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/anthropic/integrations/${anthropicIntegration.id}`,
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

			setAnthropicIntegration(null);
			setCurrentStep(githubIntegration ? 2 : 1);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting Anthropic integration:", err);
			setError("Failed to disconnect Anthropic integration");
		} finally {
			setLoading(false);
		}
	};

	const loadGitHubRepositories = async () => {
		setLoadingRepos(true);
		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/github/repositories`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			if (!response.ok) {
				throw new Error("Failed to load repositories");
			}

			const repos: GitHubRepository[] = await response.json();
			setRepositories(repos);
		} catch (err) {
			console.error("Error loading repositories:", err);
			setError("Failed to load GitHub repositories");
		} finally {
			setLoadingRepos(false);
		}
	};

	const handleSaveCCAgent = async () => {
		setSavingCCAgent(true);
		setError(null);

		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			if (!selectedRepo) {
				setError("Please select a repository");
				return;
			}

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/ccagent-container/integrations`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${token}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					instances_count: instancesCount,
					repo_url: selectedRepo,
				}),
			});

			if (!response.ok) {
				const errorText = await response.text();
				throw new Error(errorText || "Failed to create CCAgent integration");
			}

			const integration: CCAgentContainerIntegration = await response.json();
			setCCAgentIntegration(integration);
			setCurrentStep(4);
		} catch (err) {
			console.error("Error saving CCAgent integration:", err);
			setError(err instanceof Error ? err.message : "Failed to save CCAgent integration");
		} finally {
			setSavingCCAgent(false);
		}
	};

	const handleDisconnectCCAgent = async () => {
		if (!ccAgentIntegration) return;

		const confirmed = window.confirm(
			"Are you sure you want to disconnect this CCAgent integration?",
		);

		if (!confirmed) return;

		setLoading(true);
		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/ccagent-container/integrations/${ccAgentIntegration.id}`,
				{
					method: "DELETE",
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			);

			if (!response.ok) {
				throw new Error("Failed to disconnect CCAgent integration");
			}

			setCCAgentIntegration(null);
			setCurrentStep(githubIntegration && anthropicIntegration ? 3 : 2);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting CCAgent integration:", err);
			setError("Failed to disconnect CCAgent integration");
		} finally {
			setLoading(false);
		}
	};

	const handleContinueToDashboard = () => {
		router.push("/");
	};

	if (loading) {
		return (
			<div className="flex min-h-screen items-center justify-center">
				<Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
			</div>
		);
	}

	const isComplete = githubIntegration && anthropicIntegration && ccAgentIntegration;

	return (
		<div className="flex min-h-screen items-center justify-center p-4">
			<Card className="w-full max-w-4xl">
				<CardHeader>
					<CardTitle>Welcome to Claude Control</CardTitle>
					<CardDescription>Let's get you set up with your integrations</CardDescription>
				</CardHeader>
				<CardContent className="space-y-6">
					{error && (
						<div className="rounded-lg bg-destructive/10 p-4 text-destructive">{error}</div>
					)}

					{/* Stepper Progress */}
					<div className="flex items-center justify-between">
						<div className="flex items-center gap-2">
							<div
								className={`flex h-8 w-8 items-center justify-center rounded-full ${
									currentStep >= 1 ? "bg-primary text-primary-foreground" : "bg-muted"
								}`}
							>
								{githubIntegration ? <CheckCircle className="h-5 w-5" /> : "1"}
							</div>
							<span className="text-sm font-medium">GitHub</span>
						</div>
						<div className="h-px flex-1 bg-muted mx-4" />
						<div className="flex items-center gap-2">
							<div
								className={`flex h-8 w-8 items-center justify-center rounded-full ${
									currentStep >= 2 ? "bg-primary text-primary-foreground" : "bg-muted"
								}`}
							>
								{anthropicIntegration ? <CheckCircle className="h-5 w-5" /> : "2"}
							</div>
							<span className="text-sm font-medium">Anthropic</span>
						</div>
						<div className="h-px flex-1 bg-muted mx-4" />
						<div className="flex items-center gap-2">
							<div
								className={`flex h-8 w-8 items-center justify-center rounded-full ${
									currentStep >= 3 ? "bg-primary text-primary-foreground" : "bg-muted"
								}`}
							>
								{ccAgentIntegration ? <CheckCircle className="h-5 w-5" /> : "3"}
							</div>
							<span className="text-sm font-medium">CCAgent</span>
						</div>
						<div className="h-px flex-1 bg-muted mx-4" />
						<div className="flex items-center gap-2">
							<div
								className={`flex h-8 w-8 items-center justify-center rounded-full ${
									isComplete ? "bg-primary text-primary-foreground" : "bg-muted"
								}`}
							>
								{isComplete ? <CheckCircle className="h-5 w-5" /> : "4"}
							</div>
							<span className="text-sm font-medium">Complete</span>
						</div>
					</div>

					{/* Step 1: GitHub Integration */}
					{currentStep === 1 && !githubIntegration && (
						<Card>
							<CardHeader>
								<CardTitle className="flex items-center gap-2">
									<GitBranch className="h-5 w-5" />
									Connect GitHub
								</CardTitle>
								<CardDescription>
									Install the Claude Control GitHub App to access your repositories
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<p className="text-sm text-muted-foreground">This will allow Claude Control to:</p>
								<ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
									<li>Access repository metadata</li>
									<li>Read repository contents</li>
									<li>Create branches and pull requests</li>
									<li>Manage issues and comments</li>
								</ul>
								<Button onClick={handleInstallGitHub} className="w-full">
									<GitBranch className="mr-2 h-4 w-4" />
									Install GitHub App
								</Button>
							</CardContent>
						</Card>
					)}

					{/* GitHub Connected State */}
					{githubIntegration && currentStep === 1 && (
						<Card>
							<CardHeader>
								<CardTitle className="flex items-center gap-2 text-green-600 dark:text-green-400">
									<CheckCircle className="h-5 w-5" />
									GitHub Connected
								</CardTitle>
							</CardHeader>
							<CardContent className="space-y-4">
								<div className="rounded-lg border bg-muted/50 p-4">
									<div className="flex items-start justify-between">
										<div className="flex-1">
											<dl className="space-y-1 text-sm">
												<div>
													<dt className="inline font-medium text-muted-foreground">
														Installation ID:
													</dt>{" "}
													<dd className="inline font-mono">
														{githubIntegration.github_installation_id}
													</dd>
												</div>
												<div>
													<dt className="inline font-medium text-muted-foreground">Created:</dt>{" "}
													<dd className="inline">
														{new Date(githubIntegration.created_at).toLocaleDateString()}
													</dd>
												</div>
											</dl>
										</div>
										<Button
											variant="ghost"
											size="sm"
											onClick={handleDisconnectGitHub}
											disabled={loading}
											className="text-muted-foreground hover:text-destructive"
										>
											<Trash2 className="h-4 w-4 mr-2" />
											{loading ? "Disconnecting..." : "Disconnect"}
										</Button>
									</div>
								</div>
								<Button onClick={() => setCurrentStep(2)} className="w-full">
									Continue to Anthropic Setup
								</Button>
							</CardContent>
						</Card>
					)}

					{/* Step 2: Anthropic Integration */}
					{currentStep === 2 && !anthropicIntegration && (
						<Card>
							<CardHeader>
								<CardTitle className="flex items-center gap-2">
									<User className="h-5 w-5" />
									Connect Anthropic
								</CardTitle>
								<CardDescription>Choose how to connect your Anthropic account</CardDescription>
							</CardHeader>
							<CardContent>
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
											disabled={!apiKey.trim() || savingAnthropic}
											className="w-full"
										>
											{savingAnthropic ? (
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
													<strong>Step 1:</strong> Click the button below to authorize Claude
													Control
												</p>
												<Button
													onClick={handleOpenClaudeOAuth}
													className="mt-2 w-full"
													variant="outline"
												>
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
												disabled={!oauthCode.trim() || savingAnthropic}
												className="w-full"
											>
												{savingAnthropic ? (
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
					)}

					{/* Anthropic Connected State */}
					{anthropicIntegration && currentStep === 2 && (
						<Card>
							<CardHeader>
								<CardTitle className="flex items-center gap-2 text-green-600 dark:text-green-400">
									<CheckCircle className="h-5 w-5" />
									Anthropic Connected
								</CardTitle>
							</CardHeader>
							<CardContent className="space-y-4">
								<div className="rounded-lg border bg-muted/50 p-4">
									<div className="flex items-start justify-between">
										<div className="flex-1">
											<dl className="space-y-1 text-sm">
												<div>
													<dt className="inline font-medium text-muted-foreground">Type:</dt>{" "}
													<dd className="inline">
														{anthropicIntegration.has_api_key ? "API Key" : "OAuth Token"}
													</dd>
												</div>
												<div>
													<dt className="inline font-medium text-muted-foreground">Created:</dt>{" "}
													<dd className="inline">
														{new Date(anthropicIntegration.created_at).toLocaleDateString()}
													</dd>
												</div>
											</dl>
										</div>
										<Button
											variant="ghost"
											size="sm"
											onClick={handleDisconnectAnthropic}
											disabled={loading}
											className="text-muted-foreground hover:text-destructive"
										>
											<Trash2 className="h-4 w-4 mr-2" />
											{loading ? "Disconnecting..." : "Disconnect"}
										</Button>
									</div>
								</div>
								<Button onClick={() => setCurrentStep(3)} className="w-full">
									Continue to CCAgent Setup
								</Button>
							</CardContent>
						</Card>
					)}

					{/* Step 3: CCAgent Container Integration */}
					{currentStep === 3 && !ccAgentIntegration && (
						<Card>
							<CardHeader>
								<CardTitle className="flex items-center gap-2">
									<Server className="h-5 w-5" />
									Configure CCAgent
								</CardTitle>
								<CardDescription>
									Set up your CCAgent container to run automated tasks
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<div className="space-y-2">
									<Label htmlFor="repository">Repository</Label>
									<Select value={selectedRepo} onValueChange={setSelectedRepo} disabled={loadingRepos}>
										<SelectTrigger id="repository">
											<SelectValue placeholder={loadingRepos ? "Loading repositories..." : "Select a repository"} />
										</SelectTrigger>
										<SelectContent>
											{repositories.map((repo) => (
												<SelectItem key={repo.id} value={repo.html_url}>
													{repo.full_name}
													{repo.private && " 🔒"}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
									<p className="text-xs text-muted-foreground">
										Select the repository where CCAgent will work
									</p>
								</div>

								<div className="space-y-2">
									<Label>Instances</Label>
									<div className="space-y-2">
										<div className="flex items-center justify-between p-3 border rounded-lg">
											<div className="flex items-center gap-3">
												<input
													type="radio"
													id="instance-1"
													name="instances"
													value="1"
													checked={instancesCount === 1}
													readOnly
													className="h-4 w-4"
												/>
												<label htmlFor="instance-1" className="text-sm font-medium cursor-pointer">
													1 Instance
												</label>
											</div>
											<span className="text-xs text-muted-foreground">Default</span>
										</div>
										{[2, 3, 4, 5].map((count) => (
											<div key={count} className="flex items-center justify-between p-3 border rounded-lg opacity-50">
												<div className="flex items-center gap-3">
													<input
														type="radio"
														id={`instance-${count}`}
														name="instances"
														value={count.toString()}
														disabled
														className="h-4 w-4"
													/>
													<label htmlFor={`instance-${count}`} className="text-sm font-medium">
														{count} Instances
													</label>
												</div>
												<div className="flex items-center gap-2 text-xs text-muted-foreground">
													<MessageCircle className="h-3 w-3" />
													<span>Reach out if you need more</span>
												</div>
											</div>
										))}
									</div>
								</div>

								<Button
									onClick={handleSaveCCAgent}
									disabled={!selectedRepo || savingCCAgent}
									className="w-full"
								>
									{savingCCAgent ? (
										<>
											<Loader2 className="mr-2 h-4 w-4 animate-spin" />
											Saving...
										</>
									) : (
										<>
											<Server className="mr-2 h-4 w-4" />
											Save Configuration
										</>
									)}
								</Button>
							</CardContent>
						</Card>
					)}

					{/* CCAgent Connected State */}
					{ccAgentIntegration && currentStep === 3 && (
						<Card>
							<CardHeader>
								<CardTitle className="flex items-center gap-2 text-green-600 dark:text-green-400">
									<CheckCircle className="h-5 w-5" />
									CCAgent Configured
								</CardTitle>
							</CardHeader>
							<CardContent className="space-y-4">
								<div className="rounded-lg border bg-muted/50 p-4">
									<div className="flex items-start justify-between">
										<div className="flex-1">
											<dl className="space-y-1 text-sm">
												<div>
													<dt className="inline font-medium text-muted-foreground">Repository:</dt>{" "}
													<dd className="inline">{ccAgentIntegration.repo_url}</dd>
												</div>
												<div>
													<dt className="inline font-medium text-muted-foreground">Instances:</dt>{" "}
													<dd className="inline">{ccAgentIntegration.instances_count}</dd>
												</div>
												<div>
													<dt className="inline font-medium text-muted-foreground">Created:</dt>{" "}
													<dd className="inline">
														{new Date(ccAgentIntegration.created_at).toLocaleDateString()}
													</dd>
												</div>
											</dl>
										</div>
										<Button
											variant="ghost"
											size="sm"
											onClick={handleDisconnectCCAgent}
											disabled={loading}
											className="text-muted-foreground hover:text-destructive"
										>
											<Trash2 className="h-4 w-4 mr-2" />
											{loading ? "Disconnecting..." : "Disconnect"}
										</Button>
									</div>
								</div>
								<Button onClick={() => setCurrentStep(4)} className="w-full">
									View Summary
								</Button>
							</CardContent>
						</Card>
					)}

					{/* Step 4: Complete */}
					{currentStep === 4 && isComplete && (
						<Card>
							<CardHeader>
								<CardTitle className="flex items-center gap-2 text-green-600 dark:text-green-400">
									<CheckCircle className="h-5 w-5" />
									Onboarding Complete!
								</CardTitle>
								<CardDescription>You're all set up and ready to use Claude Control</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<div className="grid gap-4 md:grid-cols-3">
									<div className="rounded-lg border bg-muted/50 p-4">
										<div className="flex items-center gap-2 mb-2">
											<GitBranch className="h-4 w-4" />
											<span className="font-medium text-sm">GitHub</span>
										</div>
										<p className="text-xs text-muted-foreground">
											Installation ID: {githubIntegration?.github_installation_id}
										</p>
									</div>
									<div className="rounded-lg border bg-muted/50 p-4">
										<div className="flex items-center gap-2 mb-2">
											<User className="h-4 w-4" />
											<span className="font-medium text-sm">Anthropic</span>
										</div>
										<p className="text-xs text-muted-foreground">
											{anthropicIntegration?.has_api_key
												? "API Key configured"
												: "OAuth token configured"}
										</p>
									</div>
									<div className="rounded-lg border bg-muted/50 p-4">
										<div className="flex items-center gap-2 mb-2">
											<Server className="h-4 w-4" />
											<span className="font-medium text-sm">CCAgent</span>
										</div>
										<p className="text-xs text-muted-foreground">
											{ccAgentIntegration?.instances_count} instance(s) configured
										</p>
									</div>
								</div>
								<Button onClick={handleContinueToDashboard} size="lg" className="w-full">
									Continue to Dashboard
								</Button>
							</CardContent>
						</Card>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
