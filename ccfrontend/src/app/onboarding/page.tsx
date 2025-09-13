"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ClaudeControlIcon, DiscordIcon, SlackIcon } from "@/icons";
import { ClaudeIcon } from "@/icons/ClaudeIcon";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import {
	CheckCircle,
	ExternalLink,
	GitBranch,
	Key,
	Loader2,
	MessageCircle,
	Server,
	Trash2,
	User,
} from "lucide-react";
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

interface SlackIntegration {
	id: string;
	slack_team_id: string;
	slack_team_name: string;
	user_id: string;
	created_at: string;
	updated_at: string;
}

interface DiscordIntegration {
	id: string;
	discord_guild_id: string;
	discord_guild_name: string;
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
	const [slackIntegration, setSlackIntegration] = useState<SlackIntegration | null>(null);
	const [discordIntegration, setDiscordIntegration] = useState<DiscordIntegration | null>(null);
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

	// Deployment state
	const [deploying, setDeploying] = useState(false);
	const [deploymentMessage, setDeploymentMessage] = useState("");

	// Check onboarding status and redirect to main page if already completed
	useEffect(() => {
		const checkOnboardingStatus = async () => {
			if (!getToken) return;

			try {
				const token = await getToken();
				if (!token) return;

				const response = await fetch(`${env.CCBACKEND_BASE_URL}/settings/org-onboarding_finished`, {
					method: "GET",
					headers: {
						Authorization: `Bearer ${token}`,
						"Content-Type": "application/json",
					},
				});

				if (response.ok) {
					const data = await response.json();
					if (data.value) {
						router.push("/");
						return;
					}
				}
			} catch (error) {
				console.error("Error checking onboarding status:", error);
			}
		};

		checkOnboardingStatus();
	}, [getToken, router]);

	// Check for existing integrations on mount
	useEffect(() => {
		checkExistingIntegrations();
	}, []);

	// Load repositories when reaching step 4
	useEffect(() => {
		if (currentStep === 4 && githubIntegration && repositories.length === 0) {
			loadGitHubRepositories();
		}
	}, [currentStep, githubIntegration, repositories.length]);

	// Scroll to main content when step changes or page loads (mobile optimization)
	useEffect(() => {
		// Small delay to ensure DOM is updated
		const timer = setTimeout(() => {
			const mainContent = document.querySelector(".main-content");
			if (mainContent && window.innerWidth < 1024) {
				// Only scroll on mobile/tablet
				mainContent.scrollIntoView({ behavior: "smooth", block: "start" });
			}
		}, 300);

		return () => clearTimeout(timer);
	}, [currentStep, loading]); // Also trigger when loading changes

	// Deployment message rotation
	useEffect(() => {
		if (!deploying) return;

		const messages = [
			"🚀 Spinning up your Claude Control agent...",
			"⚙️ Configuring container environment...",
			"🔧 Installing dependencies...",
			"🐳 Building Docker image...",
			"☁️ Deploying to cloud infrastructure...",
			"🔗 Establishing secure connections...",
			"✨ Finalizing deployment...",
			"🎯 Almost ready...",
		];

		let messageIndex = 0;
		setDeploymentMessage(messages[messageIndex]);

		const scheduleNextMessage = () => {
			// Random delay between 1.5-3.5 seconds for realistic jitter
			const delay = Math.random() * 2000 + 1500;
			return setTimeout(() => {
				messageIndex = (messageIndex + 1) % messages.length;
				setDeploymentMessage(messages[messageIndex]);
				scheduleNextMessage();
			}, delay);
		};

		const timeout = scheduleNextMessage();

		return () => clearTimeout(timeout);
	}, [deploying]);

	const checkExistingIntegrations = async () => {
		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				setLoading(false);
				return;
			}

			// Check Slack integration
			const slackResponse = await fetch(`${env.CCBACKEND_BASE_URL}/slack/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			let hasSlackOrDiscord = false;
			if (slackResponse.ok) {
				const slackIntegrations: SlackIntegration[] = await slackResponse.json();
				if (slackIntegrations.length > 0) {
					setSlackIntegration(slackIntegrations[0]);
					hasSlackOrDiscord = true;
					setCurrentStep(2);
				}
			}

			// Check Discord integration
			const discordResponse = await fetch(`${env.CCBACKEND_BASE_URL}/discord/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			if (discordResponse.ok) {
				const discordIntegrations: DiscordIntegration[] = await discordResponse.json();
				if (discordIntegrations.length > 0) {
					setDiscordIntegration(discordIntegrations[0]);
					hasSlackOrDiscord = true;
					setCurrentStep(2);
				}
			}

			// Check GitHub integration
			const githubResponse = await fetch(`${env.CCBACKEND_BASE_URL}/github/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			let hasGithub = false;
			if (githubResponse.ok) {
				const githubIntegrations: GitHubIntegration[] = await githubResponse.json();
				if (githubIntegrations.length > 0) {
					setGithubIntegration(githubIntegrations[0]);
					hasGithub = true;
				}
			}

			// Update step based on what we have
			if (hasSlackOrDiscord && hasGithub) {
				setCurrentStep(3);
			} else if (hasSlackOrDiscord) {
				setCurrentStep(2);
			}

			// Check Anthropic integration
			const anthropicResponse = await fetch(`${env.CCBACKEND_BASE_URL}/anthropic/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			let hasAnthropic = false;
			if (anthropicResponse.ok) {
				const anthropicIntegrations: AnthropicIntegration[] = await anthropicResponse.json();
				if (anthropicIntegrations.length > 0) {
					setAnthropicIntegration(anthropicIntegrations[0]);
					hasAnthropic = true;
				}
			}

			// Check CCAgent Container integration
			const ccAgentResponse = await fetch(
				`${env.CCBACKEND_BASE_URL}/ccagent-container/integrations`,
				{
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			);

			let hasCCAgent = false;
			if (ccAgentResponse.ok) {
				const integrations: CCAgentContainerIntegration[] = await ccAgentResponse.json();
				if (integrations.length > 0) {
					setCCAgentIntegration(integrations[0]);
					hasCCAgent = true;
				}
			}

			// Final step determination
			if (hasSlackOrDiscord && hasGithub && hasAnthropic && hasCCAgent) {
				setCurrentStep(5);
			} else if (hasSlackOrDiscord && hasGithub && hasAnthropic) {
				setCurrentStep(4);
			} else if (hasSlackOrDiscord && hasGithub) {
				setCurrentStep(3);
			} else if (hasSlackOrDiscord) {
				setCurrentStep(2);
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

	const handleConnectSlack = () => {
		const scope =
			"app_mentions:read,channels:history,chat:write,commands,reactions:write,reactions:read,team:read";
		const userScope = "";

		const slackAuthUrl = `https://slack.com/oauth/v2/authorize?client_id=${env.SLACK_CLIENT_ID}&scope=${encodeURIComponent(scope)}&user_scope=${encodeURIComponent(userScope)}&redirect_uri=${encodeURIComponent(env.SLACK_REDIRECT_URI)}`;

		window.location.href = slackAuthUrl;
	};

	const handleConnectDiscord = () => {
		const discordAuthUrl = `https://discord.com/oauth2/authorize?client_id=1403408262338187264&permissions=34359740480&integration_type=0&scope=bot&redirect_uri=${encodeURIComponent(env.DISCORD_REDIRECT_URI)}&response_type=code`;

		window.location.href = discordAuthUrl;
	};

	const handleDisconnectSlack = async () => {
		if (!slackIntegration) return;

		const confirmed = window.confirm("Are you sure you want to disconnect this Slack integration?");

		if (!confirmed) return;

		setLoading(true);
		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/slack/integrations/${slackIntegration.id}`,
				{
					method: "DELETE",
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			);

			if (!response.ok) {
				throw new Error("Failed to disconnect Slack integration");
			}

			setSlackIntegration(null);
			setCurrentStep(1);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting Slack integration:", err);
			setError("Failed to disconnect Slack integration");
		} finally {
			setLoading(false);
		}
	};

	const handleDisconnectDiscord = async () => {
		if (!discordIntegration) return;

		const confirmed = window.confirm(
			"Are you sure you want to disconnect this Discord integration?",
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
				`${env.CCBACKEND_BASE_URL}/discord/integrations/${discordIntegration.id}`,
				{
					method: "DELETE",
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			);

			if (!response.ok) {
				throw new Error("Failed to disconnect Discord integration");
			}

			setDiscordIntegration(null);
			setCurrentStep(1);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting Discord integration:", err);
			setError("Failed to disconnect Discord integration");
		} finally {
			setLoading(false);
		}
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
			setCurrentStep(slackIntegration || discordIntegration ? 2 : 1);
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

			const integration: AnthropicIntegration = await response.json();
			setAnthropicIntegration(integration);
			setCurrentStep(4);
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
			setCurrentStep(githubIntegration ? 3 : slackIntegration || discordIntegration ? 2 : 1);
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
			setCurrentStep(5);
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
			setCurrentStep(
				anthropicIntegration
					? 4
					: githubIntegration
						? 3
						: slackIntegration || discordIntegration
							? 2
							: 1,
			);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting CCAgent integration:", err);
			setError("Failed to disconnect CCAgent integration");
		} finally {
			setLoading(false);
		}
	};

	const handleDeployClaudeControl = async () => {
		setDeploying(true);
		setError(null);

		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			// Trigger deployment
			if (!ccAgentIntegration) {
				throw new Error("CCAgent integration not found");
			}

			const deployResponse = await fetch(
				`${env.CCBACKEND_BASE_URL}/ccagents/${ccAgentIntegration.id}/redeploy`,
				{
					method: "POST",
					headers: {
						Authorization: `Bearer ${token}`,
						"Content-Type": "application/json",
					},
				},
			);

			if (!deployResponse.ok) {
				throw new Error("Failed to trigger deployment");
			}

			// Mark onboarding as completed
			await fetch(`${env.CCBACKEND_BASE_URL}/settings`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${token}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					key: "org-onboarding_finished",
					settingType: "bool",
					value: true,
				}),
			});

			// Wait a bit to show completion message
			setTimeout(() => {
				router.push("/");
			}, 2000);
		} catch (error) {
			console.error("Error deploying Claude Control:", error);
			setError("Failed to deploy Claude Control");
			setDeploying(false);
		}
	};

	if (loading) {
		return (
			<div className="flex items-center justify-center min-h-screen">
				<div className="animate-pulse">
					<div className="h-8 w-32 bg-muted rounded mb-4" />
					<div className="h-4 w-48 bg-muted rounded" />
				</div>
			</div>
		);
	}

	const isComplete =
		githubIntegration &&
		(slackIntegration || discordIntegration) &&
		anthropicIntegration &&
		ccAgentIntegration;

	return (
		<div className="min-h-screen p-4 lg:p-8 pb-16 lg:pb-32">
			<div className="mx-auto max-w-6xl pb-8 lg:pb-16">
				<div className="mb-8 text-center">
					<h1 className="text-3xl font-bold tracking-tight">Welcome to Claude Control</h1>
					<p className="text-lg text-muted-foreground mt-2">Let's get you set up!</p>
				</div>

				{error && (
					<div className="rounded-lg bg-destructive/10 p-4 text-destructive mb-8 mx-auto max-w-2xl">
						{error}
					</div>
				)}

				<div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
					{/* Vertical Stepper */}
					<div className="lg:col-span-3">
						<div className="sticky top-8">
							<nav aria-label="Setup progress" className="space-y-2">
								{[
									{
										step: 1,
										title: "Install Claude Control",
										description: "Set it up in Slack or Discord",
										icon: (
											<ClaudeControlIcon
												className="h-4 w-4"
												primaryColor="white"
												secondaryColor="black"
											/>
										),
										isCompleted: !!(slackIntegration || discordIntegration),
										isCurrent: currentStep === 1,
									},
									{
										step: 2,
										title: "Link Github Account",
										description: "So we can access your repos",
										icon: <GitBranch className="h-4 w-4" />,
										isCompleted: !!githubIntegration,
										isCurrent: currentStep === 2,
									},
									{
										step: 3,
										title: "Link Claude Integration",
										description: "So we can run Claude Code",
										icon: <ClaudeIcon className="h-4 w-4" />,
										isCompleted: !!anthropicIntegration,
										isCurrent: currentStep === 3,
									},
									{
										step: 4,
										title: "Deploy Background Agent",
										description: "Configure the remote container",
										icon: <Server className="h-4 w-4" />,
										isCompleted: !!ccAgentIntegration,
										isCurrent: currentStep === 4,
									},
									{
										step: 5,
										title: "Done 👌",
										description: "Setup complete",
										icon: <CheckCircle className="h-4 w-4" />,
										isCompleted: isComplete,
										isCurrent: currentStep === 5,
									},
								].map((item, index, array) => (
									<div key={item.step} className="relative">
										<div
											className={`flex items-center gap-3 rounded-lg p-3 transition-colors ${
												item.isCurrent
													? "bg-primary/10 border border-primary/20"
													: item.isCompleted
														? "bg-green-50 dark:bg-green-900/20"
														: "bg-muted/50"
											}`}
										>
											<div
												className={`flex h-8 w-8 items-center justify-center rounded-full shrink-0 ${
													item.isCompleted
														? "bg-green-600 text-white"
														: item.isCurrent
															? "bg-primary text-primary-foreground"
															: "bg-muted-foreground/20"
												}`}
											>
												{item.isCompleted ? (
													<CheckCircle className="h-4 w-4" />
												) : item.isCurrent ? (
													item.icon
												) : (
													<span className="text-sm font-medium">{item.step}</span>
												)}
											</div>
											<div className="min-w-0 flex-1">
												<div
													className={`text-sm font-medium ${
														item.isCurrent
															? "text-primary"
															: item.isCompleted
																? "text-green-700 dark:text-green-400"
																: "text-muted-foreground"
													}`}
												>
													{item.title}
												</div>
												<div className="text-xs text-muted-foreground">{item.description}</div>
											</div>
										</div>
										{/* Connector line */}
										{index < array.length - 1 && (
											<div
												className={`absolute left-[26px] top-14 h-6 w-px ${
													item.isCompleted ? "bg-green-600" : "bg-muted"
												}`}
											/>
										)}
									</div>
								))}
							</nav>
						</div>
					</div>

					{/* Main Content */}
					<div className="lg:col-span-9 main-content pb-4 lg:pb-8">
						{/* Step 1: Slack/Discord Integration */}
						{currentStep === 1 && !slackIntegration && !discordIntegration && (
							<Card className="w-full">
								<CardHeader>
									<CardTitle>Install the Claude Control App</CardTitle>
									<CardDescription>Install it in your Slack or Discord</CardDescription>
								</CardHeader>
								<CardContent className="space-y-4">
									<div className="flex flex-col sm:flex-row gap-4 justify-center">
										<Button
											size="lg"
											className="flex items-center gap-2 w-full sm:w-auto"
											onClick={handleConnectSlack}
										>
											<SlackIcon className="h-5 w-5" color="white" />
											Connect Slack
										</Button>
										<Button
											size="lg"
											className="flex items-center gap-2 w-full sm:w-auto"
											onClick={handleConnectDiscord}
										>
											<DiscordIcon className="h-5 w-5" color="white" />
											Connect Discord
										</Button>
									</div>
								</CardContent>
							</Card>
						)}

						{/* Slack/Discord Connected State */}
						{(slackIntegration || discordIntegration) && currentStep === 1 && (
							<Card className="w-full">
								<CardHeader>
									<CardTitle className="flex items-center gap-2 text-green-600 dark:text-green-400">
										<CheckCircle className="h-5 w-5" />
										Chat Platform Connected
									</CardTitle>
								</CardHeader>
								<CardContent className="space-y-4">
									<div className="rounded-lg border bg-muted/50 p-4">
										<div className="flex items-start justify-between">
											<div className="flex-1">
												<dl className="space-y-1 text-sm">
													<div>
														<dt className="inline font-medium text-muted-foreground">Platform:</dt>{" "}
														<dd className="inline">
															{slackIntegration
																? `Slack (${slackIntegration.slack_team_name})`
																: `Discord (${discordIntegration?.discord_guild_name})`}
														</dd>
													</div>
													<div>
														<dt className="inline font-medium text-muted-foreground">Connected:</dt>{" "}
														<dd className="inline">
															{new Date(
																(slackIntegration || discordIntegration)?.created_at || "",
															).toLocaleDateString()}
														</dd>
													</div>
												</dl>
											</div>
											<Button
												variant="destructive"
												size="sm"
												onClick={slackIntegration ? handleDisconnectSlack : handleDisconnectDiscord}
												disabled={loading}
											>
												<Trash2 className="h-4 w-4 mr-2" />
												{loading ? "Disconnecting..." : "Disconnect"}
											</Button>
										</div>
									</div>
									<Button onClick={() => setCurrentStep(2)} className="w-full">
										Continue to GitHub Setup
									</Button>
								</CardContent>
							</Card>
						)}

						{/* Step 2: GitHub Integration */}
						{currentStep === 2 && !githubIntegration && (
							<Card className="w-full">
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
									<p className="text-sm text-muted-foreground">
										This will allow Claude Control to:
									</p>
									<ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
										<li>Read repository contents</li>
										<li>Create branches and pull requests</li>
									</ul>
									<Button onClick={handleInstallGitHub} className="w-full">
										<GitBranch className="mr-2 h-4 w-4" />
										Install GitHub App
									</Button>
								</CardContent>
							</Card>
						)}

						{/* GitHub Connected State */}
						{githubIntegration && currentStep === 2 && (
							<Card className="w-full">
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
												variant="destructive"
												size="sm"
												onClick={handleDisconnectGitHub}
												disabled={loading}
											>
												<Trash2 className="h-4 w-4 mr-2" />
												{loading ? "Disconnecting..." : "Disconnect"}
											</Button>
										</div>
									</div>
									<Button onClick={() => setCurrentStep(3)} className="w-full">
										Continue to Anthropic Setup
									</Button>
								</CardContent>
							</Card>
						)}

						{/* Step 3: Anthropic Integration */}
						{currentStep === 3 && !anthropicIntegration && (
							<Card className="w-full">
								<CardHeader>
									<CardTitle>Connect your Claude account</CardTitle>
									<CardDescription>Choose how to connect your Claude account</CardDescription>
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
														After authorizing, you'll be redirected to a URL with a{" "}
														<code>code</code> parameter. Copy and paste that code here.
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
														"Save OAuth Token"
													)}
												</Button>
											</div>
										</TabsContent>
									</Tabs>
								</CardContent>
							</Card>
						)}

						{/* Anthropic Connected State */}
						{anthropicIntegration && currentStep === 3 && (
							<Card className="w-full">
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
												variant="destructive"
												size="sm"
												onClick={handleDisconnectAnthropic}
												disabled={loading}
											>
												<Trash2 className="h-4 w-4 mr-2" />
												{loading ? "Disconnecting..." : "Disconnect"}
											</Button>
										</div>
									</div>
									<Button onClick={() => setCurrentStep(4)} className="w-full">
										Continue to Background Agent Setup
									</Button>
								</CardContent>
							</Card>
						)}

						{/* Step 4: CCAgent Container Integration */}
						{currentStep === 4 && !ccAgentIntegration && (
							<Card className="w-full">
								<CardHeader>
									<CardTitle className="flex items-center gap-2">
										<Server className="h-5 w-5" />
										Configure your Claude Code Container
									</CardTitle>
									<CardDescription>
										Deploy a background agent so that claude code can work on your repository.
									</CardDescription>
								</CardHeader>
								<CardContent className="space-y-4">
									<div className="space-y-2">
										<Label htmlFor="repository">Repository</Label>
										<Select
											value={selectedRepo}
											onValueChange={setSelectedRepo}
											disabled={loadingRepos}
										>
											<SelectTrigger id="repository">
												<SelectValue
													placeholder={
														loadingRepos ? "Loading repositories..." : "Select a repository"
													}
												/>
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
													<label
														htmlFor="instance-1"
														className="text-sm font-medium cursor-pointer"
													>
														1 Instance
													</label>
												</div>
												<span className="text-xs text-muted-foreground">Default</span>
											</div>
											<div className="flex items-center justify-between p-3 border rounded-lg bg-muted/30">
												<div className="flex items-center gap-3">
													<span className="text-sm font-medium text-muted-foreground">
														More instances
													</span>
												</div>
												<div className="flex items-center gap-2 text-xs text-muted-foreground">
													<MessageCircle className="h-3 w-3" />
													<span>Contact me if you need this</span>
												</div>
											</div>
										</div>
									</div>

									<div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
										<p className="text-sm text-blue-800">
											If you prefer to self-host instead of using a managed service, please reach
											out to me for setup instructions.
										</p>
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
						{ccAgentIntegration && currentStep === 4 && (
							<Card className="w-full">
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
														<dt className="inline font-medium text-muted-foreground">
															Repository:
														</dt>{" "}
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
												variant="destructive"
												size="sm"
												onClick={handleDisconnectCCAgent}
												disabled={loading}
											>
												<Trash2 className="h-4 w-4 mr-2" />
												{loading ? "Disconnecting..." : "Disconnect"}
											</Button>
										</div>
									</div>
									<Button onClick={() => setCurrentStep(5)} className="w-full">
										View Summary
									</Button>
								</CardContent>
							</Card>
						)}

						{/* Step 5: Complete */}
						{currentStep === 5 && isComplete && (
							<Card className="w-full">
								<CardHeader>
									<CardTitle className="flex items-center gap-2 text-green-600 dark:text-green-400">
										<CheckCircle className="h-5 w-5" />
										Onboarding Complete!
									</CardTitle>
									<CardDescription>
										You're all set up and ready to use Claude Control
									</CardDescription>
								</CardHeader>
								<CardContent className="space-y-4">
									<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
										<div className="rounded-lg border bg-muted/50 p-4">
											<div className="flex items-center gap-2 mb-2">
												{slackIntegration ? (
													<SlackIcon className="h-4 w-4" />
												) : (
													<DiscordIcon className="h-4 w-4" />
												)}
												<span className="font-medium text-sm">App Installed</span>
											</div>
											<p className="text-xs text-muted-foreground">
												{slackIntegration
													? `Slack: ${slackIntegration.slack_team_name}`
													: `Discord: ${discordIntegration?.discord_guild_name}`}
											</p>
										</div>
										<div className="rounded-lg border bg-muted/50 p-4">
											<div className="flex items-center gap-2 mb-2">
												<GitBranch className="h-4 w-4" />
												<span className="font-medium text-sm">Repository</span>
											</div>
											<p className="text-xs text-muted-foreground">
												Installation ID: {githubIntegration?.github_installation_id}
											</p>
										</div>
										<div className="rounded-lg border bg-muted/50 p-4">
											<div className="flex items-center gap-2 mb-2">
												<User className="h-4 w-4" />
												<span className="font-medium text-sm">Claude Integration</span>
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
												<span className="font-medium text-sm">Background Agent</span>
											</div>
											<p className="text-xs text-muted-foreground">
												{ccAgentIntegration?.instances_count} instance(s) configured
											</p>
										</div>
									</div>
									<Button
										onClick={handleDeployClaudeControl}
										size="lg"
										className="w-full"
										disabled={deploying}
									>
										{deploying ? (
											<>
												<Loader2 className="mr-2 h-4 w-4 animate-spin" />
												Deploy Claude Control
											</>
										) : (
											"Deploy Claude Control"
										)}
									</Button>
									{deploying && (
										<div className="text-center text-sm text-muted-foreground mt-3">
											{deploymentMessage}
										</div>
									)}
								</CardContent>
							</Card>
						)}
					</div>
				</div>
			</div>
		</div>
	);
}
