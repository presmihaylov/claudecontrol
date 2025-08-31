"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { CheckCircle, GitBranch, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";

interface GitHubIntegration {
	id: string;
	github_installation_id: string;
	organization_id: string;
	created_at: string;
	updated_at: string;
}

interface GitHubIntegrationCardProps {
	onIntegrationChange?: (integration: GitHubIntegration | null) => void;
}

export function GitHubIntegrationCard({ onIntegrationChange }: GitHubIntegrationCardProps) {
	const { getToken } = useAuth();
	const [integration, setIntegration] = useState<GitHubIntegration | null>(null);
	const [loading, setLoading] = useState(true);
	const [deleting, setDeleting] = useState(false);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		checkIntegrationStatus();
	}, []);

	const checkIntegrationStatus = async () => {
		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/github/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			if (response.ok) {
				const integrations: GitHubIntegration[] = await response.json();
				const currentIntegration = integrations.length > 0 ? integrations[0] : null;
				setIntegration(currentIntegration);
				onIntegrationChange?.(currentIntegration);
			}
		} catch (err) {
			console.error("Error checking GitHub integration:", err);
			setError("Failed to load GitHub integration status");
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
		if (!integration) return;

		const confirmed = window.confirm(
			"Are you sure you want to disconnect this GitHub integration? This will remove access to all connected repositories.",
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

			setIntegration(null);
			onIntegrationChange?.(null);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting GitHub integration:", err);
			setError("Failed to disconnect GitHub integration");
		} finally {
			setDeleting(false);
		}
	};

	if (loading) {
		return (
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<GitBranch className="h-5 w-5" />
						GitHub Integration
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

	if (error) {
		return (
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<GitBranch className="h-5 w-5" />
						GitHub Integration
					</CardTitle>
				</CardHeader>
				<CardContent>
					<div className="text-sm text-destructive">{error}</div>
				</CardContent>
			</Card>
		);
	}

	if (!integration) {
		return (
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<GitBranch className="h-5 w-5" />
						GitHub Integration
					</CardTitle>
					<CardDescription>Connect your GitHub account to access repositories</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					<p className="text-sm text-muted-foreground">This will allow Claude Control to:</p>
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
		);
	}

	return (
		<Card>
			<CardHeader>
				<CardTitle className="flex items-center gap-2">
					<GitBranch className="h-5 w-5" />
					GitHub Integration
				</CardTitle>
			</CardHeader>
			<CardContent className="space-y-4">
				<div className="rounded-lg border bg-muted/50 p-4">
					<div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
						<div className="flex-1 min-w-0">
							<dl className="space-y-1 text-sm">
								<div>
									<dt className="inline font-medium text-muted-foreground">Installation ID:</dt>{" "}
									<dd className="inline font-mono break-all">
										{integration.github_installation_id}
									</dd>
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
							onClick={handleDisconnectGitHub}
							disabled={deleting}
							className="text-foreground hover:text-destructive self-start sm:self-center flex-shrink-0"
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
