"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { Settings, Slack, Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

interface SlackIntegration {
	id: string;
	slack_team_id: string;
	slack_team_name: string;
	user_id: string;
	organization_id: string;
	created_at: string;
	updated_at: string;
}

export default function Home() {
	const router = useRouter();
	const { isLoaded, isSignedIn, getToken, signOut } = useAuth();
	const [integrations, setIntegrations] = useState<SlackIntegration[]>([]);
	const [loading, setLoading] = useState(true);
	const [backendAuthenticated, setBackendAuthenticated] = useState(false);
	const [authError, setAuthError] = useState<string | null>(null);
	const [deleting, setDeleting] = useState<string | null>(null);
	const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
	const [integrationToDelete, setIntegrationToDelete] = useState<SlackIntegration | null>(null);

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

				// Then fetch their Slack integrations
				await fetchIntegrations();
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

	const handleAddToSlack = () => {
		const scope =
			"app_mentions:read,channels:history,chat:write,commands,reactions:write,reactions:read,team:read";
		const userScope = "";

		const slackAuthUrl = `https://slack.com/oauth/v2/authorize?client_id=${env.SLACK_CLIENT_ID}&scope=${encodeURIComponent(scope)}&user_scope=${encodeURIComponent(userScope)}&redirect_uri=${encodeURIComponent(env.SLACK_REDIRECT_URI)}`;

		window.location.href = slackAuthUrl;
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
						<div />
						<h1 className="text-2xl font-semibold">Claude Control</h1>
						<Button variant="outline" size="sm" onClick={() => signOut()}>
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
					<div />
					<h1 className="text-2xl font-semibold">Claude Control</h1>
					<Button variant="outline" size="sm" onClick={() => signOut()}>
						Logout
					</Button>
				</div>
			</header>
			<div className="container mx-auto px-4 py-8 max-w-4xl">
				{integrations.length === 0 ? (
					// Show "Add to Slack" when no integrations exist
					<div className="flex flex-col items-center justify-center min-h-[60vh]">
						<p className="text-lg text-muted-foreground mb-6 text-center">
							Connect your Slack workspace to get started with Claude Control
						</p>
						<Button
							size="lg"
							className="flex items-center gap-2 cursor-pointer"
							onClick={handleAddToSlack}
						>
							<Slack className="h-5 w-5" />
							Add to Slack
						</Button>
					</div>
				) : (
					// Show list of integrations with "Connect another workspace" button
					<div className="space-y-6">
						<div>
							<h2 className="text-2xl font-semibold mb-4">Connected Workspaces</h2>
							<div className="grid gap-4">
								{integrations.map((integration) => (
									<Card key={integration.id} className="p-4">
										<div className="flex items-center justify-between w-full">
											<div className="flex items-center gap-3">
												<Slack className="h-6 w-6 text-black" />
												<div>
													<h3 className="font-semibold">{integration.slack_team_name}</h3>
													<p className="text-sm text-muted-foreground">
														Connected on {new Date(integration.created_at).toLocaleDateString()}
													</p>
												</div>
											</div>
											<div className="flex items-center gap-2">
												<Button
													variant="outline"
													size="sm"
													onClick={() => router.push(`/integrations/${integration.id}`)}
													className="flex items-center gap-2"
												>
													<Settings className="h-4 w-4" />
													Manage
												</Button>
												<Button
													variant="destructive"
													size="sm"
													onClick={() => handleDeleteIntegration(integration)}
													disabled={deleting === integration.id}
													className="flex items-center gap-2"
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

						{/* Connect another workspace button */}
						<div className="flex justify-center pt-4">
							<Button size="lg" className="flex items-center gap-2" onClick={handleAddToSlack}>
								<Slack className="h-5 w-5" />
								Connect another workspace
							</Button>
						</div>
					</div>
				)}

				{/* Delete confirmation dialog */}
				<Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Disconnect Workspace</DialogTitle>
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
								variant="destructive"
								onClick={confirmDeleteIntegration}
								disabled={deleting === integrationToDelete?.id}
							>
								{deleting === integrationToDelete?.id ? "Disconnecting..." : "Disconnect"}
							</Button>
						</DialogFooter>
					</DialogContent>
				</Dialog>
			</div>
		</div>
	);
}
