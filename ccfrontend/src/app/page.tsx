"use client";

import { AnthropicIntegrationCard } from "@/components/integrations/AnthropicIntegrationCard";
import { CCAgentIntegrationCard } from "@/components/integrations/CCAgentIntegrationCard";
import { GitHubIntegrationCard } from "@/components/integrations/GitHubIntegrationCard";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { DiscordIcon, SlackIcon } from "@/icons";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

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

export default function Home() {
	const { isLoaded, isSignedIn, getToken, signOut } = useAuth();
	const router = useRouter();
	const [integrations, setIntegrations] = useState<SlackIntegration[]>([]);
	const [discordIntegrations, setDiscordIntegrations] = useState<DiscordIntegration[]>([]);
	const [loading, setLoading] = useState(true);
	const [backendAuthenticated, setBackendAuthenticated] = useState(false);
	const [authError, setAuthError] = useState<string | null>(null);
	const [deleting, setDeleting] = useState<string | null>(null);
	const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
	const [integrationToDelete, setIntegrationToDelete] = useState<SlackIntegration | null>(null);
	const [discordDeleteDialogOpen, setDiscordDeleteDialogOpen] = useState(false);
	const [discordIntegrationToDelete, setDiscordIntegrationToDelete] =
		useState<DiscordIntegration | null>(null);

	// Integration states for child components
	const [githubIntegration, setGithubIntegration] = useState<GitHubIntegration | null>(null);
	const [anthropicIntegration, setAnthropicIntegration] = useState<AnthropicIntegration | null>(
		null,
	);
	const [ccAgentIntegration, setCCAgentIntegration] = useState<CCAgentContainerIntegration | null>(
		null,
	);

	// Check onboarding status and redirect if needed
	useEffect(() => {
		const checkOnboardingStatus = async () => {
			if (!isLoaded || !isSignedIn) return;

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
					if (!data.value) {
						router.push("/onboarding");
						return;
					}
				}
			} catch (error) {
				console.error("Error checking onboarding status:", error);
			}
		};

		checkOnboardingStatus();
	}, [isLoaded, isSignedIn, getToken, router]);

	// Authenticate user with backend and fetch integrations when they first sign in
	useEffect(() => {
		const authenticateUserAndFetchIntegrations = async () => {
			if (!isLoaded || !isSignedIn) return;

			try {
				const token = await getToken();
				if (!token) return;

				// First authenticate the user
				const authResponse = await fetch(`${env.CCBACKEND_BASE_URL}/users/authenticate`, {
					method: "POST",
					headers: {
						Authorization: `Bearer ${token}`,
						"Content-Type": "application/json",
					},
				});

				if (!authResponse.ok) {
					console.error("Failed to authenticate user:", authResponse.statusText);
					setAuthError(`Authentication failed: ${authResponse.statusText}`);
					setBackendAuthenticated(false);
					return;
				}

				const user = await authResponse.json();
				console.log("User authenticated successfully:", user);
				setBackendAuthenticated(true);
				setAuthError(null);

				// Then fetch their integrations
				await fetchSlackIntegrations();
				await fetchDiscordIntegrations();
			} catch (error) {
				console.error("Error authenticating user:", error);
				setAuthError(`Authentication error: ${error}`);
				setBackendAuthenticated(false);
			} finally {
				setLoading(false);
			}
		};

		authenticateUserAndFetchIntegrations();
	}, [isLoaded, isSignedIn, getToken]);

	const fetchSlackIntegrations = async () => {
		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/slack/integrations`, {
				method: "GET",
				headers: {
					Authorization: `Bearer ${token}`,
					"Content-Type": "application/json",
				},
			});

			if (!response.ok) {
				console.error("Failed to fetch integrations:", response.statusText);
				return;
			}

			const integrationsData = await response.json();
			setIntegrations(integrationsData || []);
		} catch (error) {
			console.error("Error fetching integrations:", error);
		}
	};

	const fetchDiscordIntegrations = async () => {
		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/discord/integrations`, {
				method: "GET",
				headers: {
					Authorization: `Bearer ${token}`,
					"Content-Type": "application/json",
				},
			});

			if (!response.ok) {
				console.error("Failed to fetch Discord integrations:", response.statusText);
				return;
			}

			const integrationsData = await response.json();
			setDiscordIntegrations(integrationsData || []);
		} catch (error) {
			console.error("Error fetching Discord integrations:", error);
		}
	};

	const handleAddToSlack = () => {
		const scope =
			"app_mentions:read,channels:history,chat:write,commands,reactions:write,reactions:read,team:read";
		const userScope = "";

		const slackAuthUrl = `https://slack.com/oauth/v2/authorize?client_id=${env.SLACK_CLIENT_ID}&scope=${encodeURIComponent(scope)}&user_scope=${encodeURIComponent(userScope)}&redirect_uri=${encodeURIComponent(env.SLACK_REDIRECT_URI)}`;

		window.location.href = slackAuthUrl;
	};

	const handleAddToDiscord = () => {
		const discordAuthUrl = `https://discord.com/oauth2/authorize?client_id=1403408262338187264&permissions=34359740480&integration_type=0&scope=bot&redirect_uri=${encodeURIComponent(env.DISCORD_REDIRECT_URI)}&response_type=code`;

		window.location.href = discordAuthUrl;
	};

	const handleDeleteIntegration = (integration: SlackIntegration) => {
		setIntegrationToDelete(integration);
		setDeleteDialogOpen(true);
	};

	const confirmDeleteIntegration = async () => {
		if (!integrationToDelete) return;

		setDeleting(integrationToDelete.id);
		setDeleteDialogOpen(false);

		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/slack/integrations/${integrationToDelete.id}`,
				{
					method: "DELETE",
					headers: {
						Authorization: `Bearer ${token}`,
						"Content-Type": "application/json",
					},
				},
			);

			if (!response.ok) {
				console.error("Failed to delete integration:", response.statusText);
				alert("Failed to delete integration. Please try again.");
				return;
			}

			// Remove the integration from local state
			setIntegrations((prev) =>
				prev.filter((integration) => integration.id !== integrationToDelete.id),
			);
		} catch (error) {
			console.error("Error deleting integration:", error);
			alert("Failed to delete integration. Please try again.");
		} finally {
			setDeleting(null);
			setIntegrationToDelete(null);
		}
	};

	const handleDeleteDiscordIntegration = (integration: DiscordIntegration) => {
		setDiscordIntegrationToDelete(integration);
		setDiscordDeleteDialogOpen(true);
	};

	const confirmDeleteDiscordIntegration = async () => {
		if (!discordIntegrationToDelete) return;

		setDeleting(discordIntegrationToDelete.id);
		setDiscordDeleteDialogOpen(false);

		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/discord/integrations/${discordIntegrationToDelete.id}`,
				{
					method: "DELETE",
					headers: {
						Authorization: `Bearer ${token}`,
						"Content-Type": "application/json",
					},
				},
			);

			if (!response.ok) {
				console.error("Failed to delete Discord integration:", response.statusText);
				alert("Failed to delete Discord integration. Please try again.");
				return;
			}

			// Remove the integration from local state
			setDiscordIntegrations((prev) =>
				prev.filter((integration) => integration.id !== discordIntegrationToDelete.id),
			);
		} catch (error) {
			console.error("Error deleting Discord integration:", error);
			alert("Failed to delete Discord integration. Please try again.");
		} finally {
			setDeleting(null);
			setDiscordIntegrationToDelete(null);
		}
	};

	if (!isLoaded || loading) {
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

	// Show error if backend authentication failed
	if (!loading && authError) {
		return (
			<div className="min-h-screen bg-background w-full overflow-x-hidden">
				<header className="border-b">
					<div className="container mx-auto px-4 py-4 flex items-center justify-between min-w-0">
						<div className="hidden sm:block" />
						<h1 className="text-xl sm:text-2xl font-semibold text-center sm:text-left">
							Claude Control
						</h1>
						<Button variant="secondary" size="sm" onClick={() => signOut()}>
							Logout
						</Button>
					</div>
				</header>
				<div className="container mx-auto px-4 py-8 max-w-4xl min-w-0 overflow-hidden">
					<div className="flex flex-col items-center justify-center min-h-[60vh]">
						<div className="text-center space-y-4">
							<h2 className="text-xl font-semibold text-destructive">Authentication Failed</h2>
							<p className="text-muted-foreground max-w-md">
								Unable to authenticate with the backend server. Please try refreshing the page or
								contact support if the issue persists.
							</p>
							<div className="text-sm text-muted-foreground bg-muted p-3 rounded-md font-mono">
								{authError}
							</div>
							<div className="space-x-2">
								<Button onClick={() => window.location.reload()}>Refresh Page</Button>
								<Button variant="outline" onClick={() => signOut()}>
									Sign Out
								</Button>
							</div>
						</div>
					</div>
				</div>
			</div>
		);
	}

	// Only show main UI if backend authentication succeeded
	if (!loading && !backendAuthenticated) {
		return (
			<div className="flex items-center justify-center min-h-screen">
				<div className="text-muted-foreground">Authenticating with backend...</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen bg-background w-full overflow-x-hidden">
			<header className="border-b">
				<div className="container mx-auto px-4 py-4 flex items-center justify-between min-w-0">
					<div className="hidden sm:block" />
					<h1 className="text-xl sm:text-2xl font-semibold text-center sm:text-left">
						Claude Control
					</h1>
					<Button variant="secondary" size="sm" onClick={() => signOut()}>
						Logout
					</Button>
				</div>
			</header>
			<div className="container mx-auto px-4 py-8 max-w-4xl min-w-0 overflow-hidden">
				<div className="space-y-6 w-full">
					{/* App Installations Section */}
					<Card className="w-full overflow-hidden">
						<CardHeader>
							<CardTitle>App Installations</CardTitle>
						</CardHeader>
						<CardContent className="space-y-4">
							{integrations.length === 0 && discordIntegrations.length === 0 ? (
								<div className="text-center space-y-4">
									<p className="text-sm text-muted-foreground">
										Connect Slack or Discord to get started
									</p>
									<div className="flex flex-col sm:flex-row gap-4 justify-center">
										<Button
											size="lg"
											className="flex items-center gap-2 w-full sm:w-auto"
											onClick={handleAddToSlack}
										>
											<SlackIcon className="h-5 w-5" color="white" />
											Connect Slack
										</Button>
										<Button
											size="lg"
											className="flex items-center gap-2 w-full sm:w-auto"
											onClick={handleAddToDiscord}
										>
											<DiscordIcon className="h-5 w-5" color="white" />
											Connect Discord
										</Button>
									</div>
								</div>
							) : (
								<div className="space-y-3">
									{integrations.map((integration) => (
										<div
											key={integration.id}
											className="flex flex-col sm:flex-row sm:items-center sm:justify-between p-3 border rounded-lg bg-muted/50 gap-3"
										>
											<div className="flex items-center gap-3">
												<SlackIcon className="h-5 w-5" />
												<div>
													<h4 className="font-medium text-sm">{integration.slack_team_name}</h4>
													<p className="text-xs text-muted-foreground">
														Connected {new Date(integration.created_at).toLocaleDateString()}
													</p>
												</div>
											</div>
											<Button
												variant="ghost"
												size="sm"
												onClick={() => handleDeleteIntegration(integration)}
												disabled={deleting === integration.id}
												className="text-foreground hover:text-destructive self-start sm:self-center"
											>
												<Trash2 className="h-4 w-4" />
												Disconnect
											</Button>
										</div>
									))}
									{discordIntegrations.map((integration) => (
										<div
											key={integration.id}
											className="flex flex-col sm:flex-row sm:items-center sm:justify-between p-3 border rounded-lg bg-muted/50 gap-3"
										>
											<div className="flex items-center gap-3">
												<DiscordIcon className="h-5 w-5" />
												<div>
													<h4 className="font-medium text-sm">{integration.discord_guild_name}</h4>
													<p className="text-xs text-muted-foreground">
														Connected {new Date(integration.created_at).toLocaleDateString()}
													</p>
												</div>
											</div>
											<Button
												variant="ghost"
												size="sm"
												onClick={() => handleDeleteDiscordIntegration(integration)}
												disabled={deleting === integration.id}
												className="text-foreground hover:text-destructive self-start sm:self-center"
											>
												<Trash2 className="h-4 w-4" />
												Disconnect
											</Button>
										</div>
									))}
									{/* Add more connections */}
									<div className="pt-2 flex flex-col sm:flex-row gap-4 justify-center">
										<Button
											size="lg"
											className="flex items-center gap-2 w-full sm:w-auto"
											onClick={handleAddToSlack}
										>
											<SlackIcon className="h-5 w-5" color="white" />
											Connect Slack
										</Button>
										<Button
											size="lg"
											className="flex items-center gap-2 w-full sm:w-auto"
											onClick={handleAddToDiscord}
										>
											<DiscordIcon className="h-5 w-5" color="white" />
											Connect Discord
										</Button>
									</div>
								</div>
							)}
						</CardContent>
					</Card>

					{/* GitHub Integration Section */}
					<GitHubIntegrationCard onIntegrationChange={setGithubIntegration} />

					{/* Claude Integration Section */}
					<AnthropicIntegrationCard onIntegrationChange={setAnthropicIntegration} />

					{/* Background Agents Section */}
					<CCAgentIntegrationCard
						onIntegrationChange={setCCAgentIntegration}
						githubIntegration={githubIntegration}
						anthropicIntegration={anthropicIntegration}
					/>
				</div>

				{/* Delete confirmation dialogs */}
				<Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Disconnect Slack Workspace</DialogTitle>
							<DialogDescription>
								Are you sure you want to disconnect "{integrationToDelete?.slack_team_name}" from
								Claude Control? This action cannot be undone.
							</DialogDescription>
						</DialogHeader>
						<DialogFooter>
							<Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>
								Cancel
							</Button>
							<Button
								variant="secondary"
								onClick={confirmDeleteIntegration}
								disabled={deleting === integrationToDelete?.id}
							>
								{deleting === integrationToDelete?.id ? "Disconnecting..." : "Disconnect"}
							</Button>
						</DialogFooter>
					</DialogContent>
				</Dialog>

				<Dialog open={discordDeleteDialogOpen} onOpenChange={setDiscordDeleteDialogOpen}>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Disconnect Discord Server</DialogTitle>
							<DialogDescription>
								Are you sure you want to disconnect "
								{discordIntegrationToDelete?.discord_guild_name}" from Claude Control? This action
								cannot be undone.
							</DialogDescription>
						</DialogHeader>
						<DialogFooter>
							<Button variant="outline" onClick={() => setDiscordDeleteDialogOpen(false)}>
								Cancel
							</Button>
							<Button
								variant="secondary"
								onClick={confirmDeleteDiscordIntegration}
								disabled={deleting === discordIntegrationToDelete?.id}
							>
								{deleting === discordIntegrationToDelete?.id ? "Disconnecting..." : "Disconnect"}
							</Button>
						</DialogFooter>
					</DialogContent>
				</Dialog>
			</div>
		</div>
	);
}
