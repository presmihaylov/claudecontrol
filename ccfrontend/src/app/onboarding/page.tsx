"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { CheckCircle, ExternalLink, GitBranch, Loader2, Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

interface GitHubIntegration {
	id: string;
	github_installation_id: string;
	organization_id: string;
	created_at: string;
	updated_at: string;
}

export default function OnboardingPage() {
	const { getToken } = useAuth();
	const router = useRouter();
	const [loading, setLoading] = useState(true);
	const [integration, setIntegration] = useState<GitHubIntegration | null>(null);
	const [error, setError] = useState<string | null>(null);

	// Check for existing GitHub integration on mount
	useEffect(() => {
		checkExistingIntegration();
	}, []);

	const checkExistingIntegration = async () => {
		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				setLoading(false);
				return;
			}

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/github/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			if (!response.ok) {
				if (response.status === 404) {
					// No integrations found, that's ok
					setLoading(false);
					return;
				}
				throw new Error("Failed to fetch GitHub integrations");
			}

			const integrations: GitHubIntegration[] = await response.json();
			if (integrations.length > 0) {
				setIntegration(integrations[0]);
			}
		} catch (err) {
			console.error("Error checking existing integration:", err);
			setError("Failed to check existing integration");
		} finally {
			setLoading(false);
		}
	};

	const handleInstallGitHub = () => {
		// Redirect to GitHub App installation page
		const githubAppUrl = "https://github.com/apps/claude-control/installations/select_target";
		const redirectUri = `${window.location.origin}/github/redirect`;
		const state = `redirect_uri=${encodeURIComponent(redirectUri)}`;

		window.location.href = `${githubAppUrl}?state=${state}`;
	};

	const handleContinue = () => {
		router.push("/");
	};

	const handleDisconnect = async () => {
		if (!integration) return;

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
				`${env.CCBACKEND_BASE_URL}/github/integrations/${integration.id}`,
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

			// Clear the integration and show success
			setIntegration(null);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting GitHub integration:", err);
			setError("Failed to disconnect GitHub integration");
		} finally {
			setLoading(false);
		}
	};

	if (loading) {
		return (
			<div className="flex min-h-screen items-center justify-center">
				<Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
			</div>
		);
	}

	return (
		<div className="flex min-h-screen items-center justify-center p-4">
			<Card className="w-full max-w-2xl">
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<GitBranch className="h-6 w-6" />
						GitHub Integration Setup
					</CardTitle>
					<CardDescription>Connect your GitHub repositories to Claude Control</CardDescription>
				</CardHeader>
				<CardContent className="space-y-6">
					{error && (
						<div className="rounded-lg bg-destructive/10 p-4 text-destructive">{error}</div>
					)}

					{integration ? (
						<div className="space-y-4">
							<div className="flex items-center gap-2 text-green-600 dark:text-green-400">
								<CheckCircle className="h-5 w-5" />
								<span className="font-medium">GitHub integration active</span>
							</div>

							<div className="rounded-lg border bg-muted/50 p-4">
								<div className="flex items-start justify-between">
									<div className="flex-1">
										<h3 className="mb-2 font-medium">Installation Details</h3>
										<dl className="space-y-1 text-sm">
											<div>
												<dt className="inline font-medium text-muted-foreground">Installation ID:</dt>{" "}
												<dd className="inline font-mono">{integration.github_installation_id}</dd>
											</div>
											<div>
												<dt className="inline font-medium text-muted-foreground">Created:</dt>{" "}
												<dd className="inline">
													{new Date(integration.created_at).toLocaleDateString()}
												</dd>
											</div>
										</dl>
									</div>
									<Button
										variant="ghost"
										size="sm"
										onClick={handleDisconnect}
										disabled={loading}
										className="text-muted-foreground hover:text-destructive"
									>
										<Trash2 className="h-4 w-4 mr-2" />
										{loading ? "Disconnecting..." : "Disconnect"}
									</Button>
								</div>
							</div>

							<div className="flex justify-between">
								<Button
									variant="outline"
									onClick={() =>
										window.open("https://github.com/settings/installations", "_blank")
									}
								>
									<ExternalLink className="mr-2 h-4 w-4" />
									Manage on GitHub
								</Button>
								<Button onClick={handleContinue}>Continue to Dashboard</Button>
							</div>
						</div>
					) : (
						<div className="space-y-4">
							<p className="text-muted-foreground">
								To use Claude Control with your GitHub repositories, you need to install the GitHub
								App. This will allow Claude Control to:
							</p>

							<ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
								<li>Access repository metadata</li>
								<li>Read repository contents</li>
								<li>Create branches and pull requests</li>
								<li>Manage issues and comments</li>
							</ul>

							<div className="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-900 dark:bg-amber-950/20">
								<p className="text-sm text-amber-800 dark:text-amber-200">
									<strong>Note:</strong> You can choose which repositories to grant access to during
									the installation process.
								</p>
							</div>

							<div className="flex justify-center">
								<Button size="lg" onClick={handleInstallGitHub}>
									<GitBranch className="mr-2 h-5 w-5" />
									Install GitHub App
								</Button>
							</div>
						</div>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
