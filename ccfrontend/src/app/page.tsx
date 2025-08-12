"use client";

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
import { Input } from "@/components/ui/input";
import { DiscordIcon, SlackIcon } from "@/icons";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { Copy, Download, Key, RefreshCw, Trash2 } from "lucide-react";
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

interface Organization {
	id: string;
	ccagent_secret_key_generated_at: string | null;
	created_at: string;
	updated_at: string;
}

interface CCAgentSecretKeyResponse {
	secret_key: string;
	generated_at: string;
}

export default function Home() {
	const { isLoaded, isSignedIn, getToken, signOut } = useAuth();
	const [integrations, setIntegrations] = useState<SlackIntegration[]>([]);
	const [discordIntegrations, setDiscordIntegrations] = useState<DiscordIntegration[]>([]);
	const [organization, setOrganization] = useState<Organization | null>(null);
	const [loading, setLoading] = useState(true);
	const [backendAuthenticated, setBackendAuthenticated] = useState(false);
	const [authError, setAuthError] = useState<string | null>(null);
	const [deleting, setDeleting] = useState<string | null>(null);
	const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
	const [integrationToDelete, setIntegrationToDelete] = useState<SlackIntegration | null>(null);
	const [discordDeleteDialogOpen, setDiscordDeleteDialogOpen] = useState(false);
	const [discordIntegrationToDelete, setDiscordIntegrationToDelete] =
		useState<DiscordIntegration | null>(null);
	const [generatingKey, setGeneratingKey] = useState(false);
	const [regenerateDialogOpen, setRegenerateDialogOpen] = useState(false);
	const [secretKeyDialogOpen, setSecretKeyDialogOpen] = useState(false);
	const [generatedSecretKey, setGeneratedSecretKey] = useState<string>("");
	const [copySuccess, setCopySuccess] = useState(false);

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

				// Then fetch their integrations and organization
				await fetchIntegrations();
				await fetchDiscordIntegrations();
				await fetchOrganization();
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

	const fetchIntegrations = async () => {
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

	const fetchOrganization = async () => {
		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/organizations`, {
				method: "GET",
				headers: {
					Authorization: `Bearer ${token}`,
					"Content-Type": "application/json",
				},
			});

			if (!response.ok) {
				console.error("Failed to fetch organization:", response.statusText);
				return;
			}

			const organizationData = await response.json();
			setOrganization(organizationData);
		} catch (error) {
			console.error("Error fetching organization:", error);
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

	const handleGenerateSecretKey = async () => {
		setGeneratingKey(true);
		setRegenerateDialogOpen(false);

		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/organizations/ccagent_secret_key`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${token}`,
					"Content-Type": "application/json",
				},
			});

			if (!response.ok) {
				console.error("Failed to generate secret key:", response.statusText);
				alert("Failed to generate secret key. Please try again.");
				return;
			}

			const data: CCAgentSecretKeyResponse = await response.json();
			setGeneratedSecretKey(data.secret_key);
			setSecretKeyDialogOpen(true);

			// Update the organization to reflect the new timestamp
			if (organization && data.generated_at) {
				setOrganization({
					...organization,
					ccagent_secret_key_generated_at: data.generated_at,
				});
			}
		} catch (error) {
			console.error("Error generating secret key:", error);
			alert("Failed to generate secret key. Please try again.");
		} finally {
			setGeneratingKey(false);
		}
	};

	const handleCopyToClipboard = async () => {
		try {
			await navigator.clipboard.writeText(generatedSecretKey);
			setCopySuccess(true);
			setTimeout(() => setCopySuccess(false), 2000);
		} catch (error) {
			console.error("Failed to copy to clipboard:", error);
		}
	};

	const handleCloseSecretKeyDialog = () => {
		setSecretKeyDialogOpen(false);
		setGeneratedSecretKey("");
		setCopySuccess(false);
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
			<div className="min-h-screen bg-background">
				<header className="border-b">
					<div className="container mx-auto px-4 py-4 flex items-center justify-between">
						<div className="hidden sm:block" />
						<h1 className="text-xl sm:text-2xl font-semibold text-center sm:text-left">Claude Control</h1>
						<Button variant="secondary" size="sm" onClick={() => signOut()}>
							Logout
						</Button>
					</div>
				</header>
				<div className="container mx-auto px-4 py-8 max-w-4xl">
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
		<div className="min-h-screen bg-background">
			<header className="border-b">
				<div className="container mx-auto px-4 py-4 flex items-center justify-between">
					<div className="hidden sm:block" />
					<h1 className="text-xl sm:text-2xl font-semibold text-center sm:text-left">Claude Control</h1>
					<Button variant="secondary" size="sm" onClick={() => signOut()}>
						Logout
					</Button>
				</div>
			</header>
			<div className="container mx-auto px-4 py-8 max-w-4xl">
				{integrations.length === 0 && discordIntegrations.length === 0 ? (
					// Show ccagent Secret Key section first, then "Add to Slack"
					<div className="space-y-6">
						{/* ccagent Secret Key Section - always show */}
						{organization && (
							<Card>
								<CardHeader>
									<CardTitle>Control Panel</CardTitle>
								</CardHeader>
								<CardContent className="space-y-4">
									{/* Setup Tutorial Link */}
									<div className="flex flex-col sm:flex-row sm:items-center justify-between p-4 border rounded-lg bg-muted/50 gap-4">
										<div className="space-y-1">
											<h4 className="font-medium">Getting Started</h4>
											<p className="text-sm text-muted-foreground">
												How to set up and use Claude Control
											</p>
										</div>
										<Button
											variant="outline"
											onClick={() =>
												window.open(
													"https://drive.google.com/file/d/11G1btpviFYzehqx0-ji3o1QhKmTR991U/view?usp=sharing",
													"_blank",
												)
											}
											className="flex items-center gap-2 w-full sm:w-auto"
										>
											📺 Watch Tutorial
										</Button>
									</div>

									{/* Download ccagent Button */}
									<div className="flex flex-col sm:flex-row sm:items-center justify-between p-4 border rounded-lg bg-muted/50 gap-4">
										<div className="space-y-1">
											<h4 className="font-medium">Download CCAgent</h4>
											<p className="text-sm text-muted-foreground">
												Download the ccagent CLI tool to start using Claude Control with your Slack
												workspaces.
											</p>
										</div>
										<Button
											variant="outline"
											onClick={() =>
												window.open(
													"https://github.com/presmihaylov/ccagent#installation",
													"_blank",
												)
											}
											className="flex items-center gap-2 w-full sm:w-auto"
										>
											<Download className="h-4 w-4" />
											Download
										</Button>
									</div>

									{/* CCAgent API Key Section */}
									<div className="flex flex-col sm:flex-row sm:items-center justify-between p-4 border rounded-lg bg-muted/50 gap-4">
										<div className="space-y-1">
											<h4 className="font-medium">CCAgent API Key</h4>
											<p className="text-sm text-muted-foreground">
												The secret key used to authenticate ccagent against your organization
											</p>
										</div>
										<div className="flex gap-2 w-full sm:w-auto">
											{organization.ccagent_secret_key_generated_at ? (
												<Button
													variant="outline"
													onClick={() => setRegenerateDialogOpen(true)}
													disabled={generatingKey}
													className="flex items-center gap-2 w-full sm:w-auto"
												>
													<RefreshCw className={`h-4 w-4 ${generatingKey ? "animate-spin" : ""}`} />
													{generatingKey ? "Regenerating..." : "Regenerate"}
												</Button>
											) : (
												<Button
													onClick={handleGenerateSecretKey}
													disabled={generatingKey}
													className="flex items-center gap-2 w-full sm:w-auto"
												>
													<Key className={`h-4 w-4 ${generatingKey ? "animate-spin" : ""}`} />
													{generatingKey ? "Generating..." : "Generate"}
												</Button>
											)}
										</div>
									</div>
								</CardContent>
							</Card>
						)}

						<div className="flex flex-col items-center justify-center">
							<p className="text-base sm:text-lg text-muted-foreground mb-6 text-center px-4">
								Connect your workspace to get started with Claude Control
							</p>
							<div className="flex flex-col sm:flex-row gap-4 w-full sm:w-auto">
								<Button
									size="lg"
									className="flex items-center gap-2 cursor-pointer w-full sm:w-auto"
									onClick={handleAddToSlack}
								>
									<SlackIcon className="h-5 w-5" color="white" />
									Add to Slack
								</Button>
								<Button
									size="lg"
									className="flex items-center gap-2 cursor-pointer w-full sm:w-auto"
									onClick={handleAddToDiscord}
								>
									<DiscordIcon className="h-5 w-5" color="white" />
									Add to Discord
								</Button>
							</div>
						</div>
					</div>
				) : (
					// Show secret key section and list of integrations
					<div className="space-y-6">
						{/* ccagent Secret Key Section */}
						{organization && (
							<Card>
								<CardHeader>
									<CardTitle>Control Panel</CardTitle>
								</CardHeader>
								<CardContent className="space-y-4">
									{/* Setup Tutorial Link */}
									<div className="flex flex-col sm:flex-row sm:items-center justify-between p-4 border rounded-lg bg-muted/50 gap-4">
										<div className="space-y-1">
											<h4 className="font-medium">Getting Started</h4>
											<p className="text-sm text-muted-foreground">
												How to set up and use Claude Control
											</p>
										</div>
										<Button
											variant="outline"
											onClick={() =>
												window.open(
													"https://drive.google.com/file/d/11G1btpviFYzehqx0-ji3o1QhKmTR991U/view?usp=sharing",
													"_blank",
												)
											}
											className="flex items-center gap-2 w-full sm:w-auto"
										>
											📺 Watch Tutorial
										</Button>
									</div>

									{/* Download ccagent Button */}
									<div className="flex flex-col sm:flex-row sm:items-center justify-between p-4 border rounded-lg bg-muted/50 gap-4">
										<div className="space-y-1">
											<h4 className="font-medium">Download CCAgent</h4>
											<p className="text-sm text-muted-foreground">
												Download the ccagent CLI tool to start using Claude Control with your Slack
												workspaces.
											</p>
										</div>
										<Button
											variant="outline"
											onClick={() =>
												window.open(
													"https://github.com/presmihaylov/ccagent#installation",
													"_blank",
												)
											}
											className="flex items-center gap-2 w-full sm:w-auto"
										>
											<Download className="h-4 w-4" />
											Download
										</Button>
									</div>

									{/* CCAgent API Key Section */}
									<div className="flex flex-col sm:flex-row sm:items-center justify-between p-4 border rounded-lg bg-muted/50 gap-4">
										<div className="space-y-1">
											<h4 className="font-medium">CCAgent API Key</h4>
											<p className="text-sm text-muted-foreground">
												The secret key used to authenticate ccagent against your organization
											</p>
										</div>
										<div className="flex gap-2 w-full sm:w-auto">
											{organization.ccagent_secret_key_generated_at ? (
												<Button
													variant="outline"
													onClick={() => setRegenerateDialogOpen(true)}
													disabled={generatingKey}
													className="flex items-center gap-2 w-full sm:w-auto"
												>
													<RefreshCw className={`h-4 w-4 ${generatingKey ? "animate-spin" : ""}`} />
													{generatingKey ? "Regenerating..." : "Regenerate"}
												</Button>
											) : (
												<Button
													onClick={handleGenerateSecretKey}
													disabled={generatingKey}
													className="flex items-center gap-2 w-full sm:w-auto"
												>
													<Key className={`h-4 w-4 ${generatingKey ? "animate-spin" : ""}`} />
													{generatingKey ? "Generating..." : "Generate"}
												</Button>
											)}
										</div>
									</div>
								</CardContent>
							</Card>
						)}

						<div>
							<h2 className="text-2xl font-semibold mb-4">Connected Workspaces</h2>
							<div className="grid gap-4">
								{integrations.map((integration) => (
									<Card key={integration.id} className="p-4">
										<div className="flex flex-col sm:flex-row sm:items-center justify-between w-full gap-4">
											<div className="flex items-center gap-3">
												<SlackIcon className="h-6 w-6" color="black" />
												<div>
													<h3 className="font-semibold">{integration.slack_team_name}</h3>
													<p className="text-sm text-muted-foreground">
														Slack • Connected on{" "}
														{new Date(integration.created_at).toLocaleDateString()}
													</p>
												</div>
											</div>
											<div className="flex items-center gap-2">
												<Button
													variant="secondary"
													size="sm"
													onClick={() => handleDeleteIntegration(integration)}
													disabled={deleting === integration.id}
													className="flex items-center gap-2 w-full sm:w-auto"
												>
													<Trash2 className="h-4 w-4" />
													{deleting === integration.id ? "Disconnecting..." : "Disconnect"}
												</Button>
											</div>
										</div>
									</Card>
								))}
								{discordIntegrations.map((integration) => (
									<Card key={integration.id} className="p-4">
										<div className="flex flex-col sm:flex-row sm:items-center justify-between w-full gap-4">
											<div className="flex items-center gap-3">
												<DiscordIcon className="h-6 w-6" color="black" />
												<div>
													<h3 className="font-semibold">{integration.discord_guild_name}</h3>
													<p className="text-sm text-muted-foreground">
														Discord • Connected on{" "}
														{new Date(integration.created_at).toLocaleDateString()}
													</p>
												</div>
											</div>
											<div className="flex items-center gap-2">
												<Button
													variant="secondary"
													size="sm"
													onClick={() => handleDeleteDiscordIntegration(integration)}
													disabled={deleting === integration.id}
													className="flex items-center gap-2 w-full sm:w-auto"
												>
													<Trash2 className="h-4 w-4" />
													{deleting === integration.id ? "Disconnecting..." : "Disconnect"}
												</Button>
											</div>
										</div>
									</Card>
								))}
							</div>
						</div>

						{/* Connect another workspace buttons */}
						<div className="flex justify-center pt-4">
							<div className="flex flex-col sm:flex-row gap-4 w-full sm:w-auto">
								<Button size="lg" className="flex items-center gap-2 w-full sm:w-auto" onClick={handleAddToSlack}>
									<SlackIcon className="h-5 w-5" color="white" />
									Connect Slack
								</Button>
								<Button size="lg" className="flex items-center gap-2 w-full sm:w-auto" onClick={handleAddToDiscord}>
									<DiscordIcon className="h-5 w-5" color="white" />
									Connect Discord
								</Button>
							</div>
						</div>
					</div>
				)}

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

				{/* Regenerate Warning Dialog */}
				<Dialog open={regenerateDialogOpen} onOpenChange={setRegenerateDialogOpen}>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Regenerate Secret Key</DialogTitle>
							<DialogDescription>
								Are you sure you want to regenerate the secret key for your organization?
								<br />
								<br />
								<strong>Warning:</strong> This will invalidate the old key and any running ccagent
								instances using the old key will stop working until you update them with the new
								key.
							</DialogDescription>
						</DialogHeader>
						<DialogFooter>
							<Button variant="outline" onClick={() => setRegenerateDialogOpen(false)}>
								Cancel
							</Button>
							<Button onClick={handleGenerateSecretKey} disabled={generatingKey}>
								{generatingKey ? "Regenerating..." : "Regenerate Key"}
							</Button>
						</DialogFooter>
					</DialogContent>
				</Dialog>

				{/* Secret Key Display Dialog */}
				<Dialog open={secretKeyDialogOpen} onOpenChange={setSecretKeyDialogOpen}>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Your ccagent Secret Key</DialogTitle>
							<DialogDescription>
								Copy this secret key and save it somewhere safe. You won't be able to see it again
								after closing this dialog.
							</DialogDescription>
						</DialogHeader>
						<div className="space-y-4">
							<div className="space-y-2">
								<label htmlFor="secret-key-input" className="text-sm font-medium">
									Secret Key
								</label>
								<div className="flex gap-2">
									<Input
										id="secret-key-input"
										type="text"
										value={generatedSecretKey}
										readOnly
										className="font-mono text-sm"
										onClick={(e) => e.currentTarget.select()}
									/>
									<Button
										variant="outline"
										size="sm"
										onClick={handleCopyToClipboard}
										className="flex items-center gap-2"
									>
										<Copy className="h-4 w-4" />
										{copySuccess ? "Copied!" : "Copy"}
									</Button>
								</div>
							</div>
							{copySuccess && (
								<p className="text-sm text-green-600">
									Secret key copied to clipboard successfully!
								</p>
							)}
						</div>
						<DialogFooter>
							<Button onClick={handleCloseSecretKeyDialog}>Close</Button>
						</DialogFooter>
					</DialogContent>
				</Dialog>
			</div>
		</div>
	);
}
