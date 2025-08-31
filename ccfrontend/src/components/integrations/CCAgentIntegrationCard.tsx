"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useToast } from "@/hooks/use-toast";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import {
	AlertTriangle,
	CheckCircle,
	Loader2,
	MessageCircle,
	RefreshCw,
	Server,
	Trash2,
} from "lucide-react";
import { useEffect, useState } from "react";

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

interface CCAgentIntegrationCardProps {
	onIntegrationChange?: (integration: CCAgentContainerIntegration | null) => void;
	githubIntegration?: GitHubIntegration | null;
	anthropicIntegration?: AnthropicIntegration | null;
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

export function CCAgentIntegrationCard({
	onIntegrationChange,
	githubIntegration,
	anthropicIntegration,
}: CCAgentIntegrationCardProps) {
	const { getToken } = useAuth();
	const { toast } = useToast();
	const [integration, setIntegration] = useState<CCAgentContainerIntegration | null>(null);
	const [repositories, setRepositories] = useState<GitHubRepository[]>([]);
	const [loading, setLoading] = useState(true);
	const [loadingRepos, setLoadingRepos] = useState(false);
	const [deleting, setDeleting] = useState(false);
	const [saving, setSaving] = useState(false);
	const [deploying, setDeploying] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [redeployRequired, setRedeployRequired] = useState(false);

	// Form states
	const [selectedRepo, setSelectedRepo] = useState("");
	const [instancesCount] = useState(1);

	useEffect(() => {
		checkIntegrationStatus();
		checkRedeployRequiredSetting();
	}, []);

	useEffect(() => {
		if (!integration && githubIntegration && repositories.length === 0) {
			loadGitHubRepositories();
		}
	}, [githubIntegration, integration, repositories.length]);

	// Monitor dependency changes and set redeploy required setting
	useEffect(() => {
		if (!integration) return; // Only monitor when agent is configured

		const hasMissingDependencies = !githubIntegration || !anthropicIntegration;

		if (hasMissingDependencies && !redeployRequired) {
			// Set redeploy required when dependencies are missing
			setRedeployRequiredSetting(true);
		} else if (!hasMissingDependencies && redeployRequired) {
			// Keep the redeploy required flag until user manually redeploys
			// This ensures user knows they need to redeploy after fixing integrations
		}
	}, [githubIntegration, anthropicIntegration, integration, redeployRequired]);

	const checkIntegrationStatus = async () => {
		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/ccagent-container/integrations`, {
				headers: {
					Authorization: `Bearer ${token}`,
				},
			});

			if (response.ok) {
				const integrations: CCAgentContainerIntegration[] = await response.json();
				const currentIntegration = integrations.length > 0 ? integrations[0] : null;
				setIntegration(currentIntegration);
				onIntegrationChange?.(currentIntegration);
			}
		} catch (err) {
			console.error("Error checking CCAgent integration:", err);
			setError("Failed to load background agent status");
		} finally {
			setLoading(false);
		}
	};

	const checkRedeployRequiredSetting = async () => {
		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/settings/org-ccagent_redeploy_required`,
				{
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			);

			if (response.ok) {
				const data = await response.json();
				setRedeployRequired(data.value || false);
			}
		} catch (err) {
			console.error("Error checking redeploy required setting:", err);
		}
	};

	const setRedeployRequiredSetting = async (required: boolean) => {
		try {
			const token = await getToken();
			if (!token) return;

			await fetch(`${env.CCBACKEND_BASE_URL}/settings/org-ccagent_redeploy_required`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${token}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify({ value: required }),
			});

			setRedeployRequired(required);
		} catch (err) {
			console.error("Error setting redeploy required setting:", err);
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
		setSaving(true);
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

			const newIntegration: CCAgentContainerIntegration = await response.json();
			setIntegration(newIntegration);
			onIntegrationChange?.(newIntegration);
			
			// Set redeploy required when creating new agent integration
			await setRedeployRequiredSetting(true);
			
			setError(null);
		} catch (err) {
			console.error("Error saving CCAgent integration:", err);
			setError(err instanceof Error ? err.message : "Failed to save CCAgent integration");
		} finally {
			setSaving(false);
		}
	};

	const handleDisconnectCCAgent = async () => {
		if (!integration) return;

		const confirmed = window.confirm(
			"Are you sure you want to disconnect this background agent integration?",
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
				`${env.CCBACKEND_BASE_URL}/ccagent-container/integrations/${integration.id}`,
				{
					method: "DELETE",
					headers: {
						Authorization: `Bearer ${token}`,
					},
				},
			);

			if (!response.ok) {
				throw new Error("Failed to disconnect background agent integration");
			}

			setIntegration(null);
			onIntegrationChange?.(null);
			setError(null);
		} catch (err) {
			console.error("Error disconnecting CCAgent integration:", err);
			setError("Failed to disconnect background agent integration");
		} finally {
			setDeleting(false);
		}
	};

	const handleRedeploy = async () => {
		if (!integration) return;

		setDeploying(true);
		setError(null);

		try {
			const token = await getToken();
			if (!token) {
				setError("Authentication required");
				return;
			}

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/ccagents/${integration.id}/redeploy`,
				{
					method: "POST",
					headers: {
						Authorization: `Bearer ${token}`,
						"Content-Type": "application/json",
					},
				},
			);

			if (!response.ok) {
				throw new Error("Failed to redeploy background agent");
			}

			// Show success toast
			toast({
				title: "Agent redeployed successfully",
				description: "Your background agent has been redeployed and is ready to work.",
				variant: "success",
			});

			// Clear the redeploy required setting after successful redeploy
			await setRedeployRequiredSetting(false);

			setDeploying(false);
		} catch (err) {
			console.error("Error redeploying background agent:", err);
			setError("Failed to redeploy background agent");
			setDeploying(false);
		}
	};

	if (loading) {
		return (
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<Server className="h-5 w-5" />
						Background Agents
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
						<Server className="h-5 w-5" />
						Background Agents
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
		if (!githubIntegration) {
			return (
				<Card>
					<CardHeader>
						<CardTitle className="flex items-center gap-2">
							<Server className="h-5 w-5" />
							Background Agents
						</CardTitle>
						<CardDescription>Connect GitHub first to set up background agents</CardDescription>
					</CardHeader>
					<CardContent>
						<p className="text-sm text-muted-foreground">
							You need to connect your GitHub account before you can configure background agents.
						</p>
					</CardContent>
				</Card>
			);
		}

		return (
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<Server className="h-5 w-5" />
						Background Agents
					</CardTitle>
					<CardDescription>Deploy a background agent to work on your repository</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					{error && <div className="text-sm text-destructive mb-4">{error}</div>}

					<div className="space-y-2">
						<Label htmlFor="repository">Repository</Label>
						<Select value={selectedRepo} onValueChange={setSelectedRepo} disabled={loadingRepos}>
							<SelectTrigger id="repository">
								<SelectValue
									placeholder={loadingRepos ? "Loading repositories..." : "Select a repository"}
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
							Select the repository where the background agent will work
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
							<div className="flex items-center justify-between p-3 border rounded-lg bg-muted/30">
								<div className="flex items-center gap-3">
									<span className="text-sm font-medium text-muted-foreground">More instances</span>
								</div>
								<div className="flex items-center gap-2 text-xs text-muted-foreground">
									<MessageCircle className="h-3 w-3" />
									<span>Contact me if you need this</span>
								</div>
							</div>
						</div>
					</div>

					<Button onClick={handleSaveCCAgent} disabled={!selectedRepo || saving} className="w-full">
						{saving ? (
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
		);
	}

	return (
		<Card>
			<CardHeader>
				<CardTitle className="flex items-center gap-2">
					<Server className="h-5 w-5" />
					Background Agents
				</CardTitle>
			</CardHeader>
			<CardContent className="space-y-4">
				<div className="rounded-lg border bg-muted/50 p-4">
					<div className="flex items-start justify-between">
						<div className="flex-1">
							<dl className="space-y-1 text-sm">
								<div>
									<dt className="inline font-medium text-muted-foreground">Repository:</dt>{" "}
									<dd className="inline">{integration.repo_url}</dd>
								</div>
								<div>
									<dt className="inline font-medium text-muted-foreground">Configured:</dt>{" "}
									<dd className="inline">
										{new Date(integration.created_at).toLocaleDateString()}
									</dd>
								</div>
							</dl>
						</div>
						<Button
							variant="ghost"
							size="sm"
							onClick={handleDisconnectCCAgent}
							disabled={deleting || deploying}
							className="text-foreground hover:text-destructive"
						>
							<Trash2 className="h-4 w-4 mr-2" />
							{deleting ? "Disconnecting..." : "Disconnect"}
						</Button>
					</div>
				</div>
				{error && <div className="text-sm text-destructive mb-4">{error}</div>}

				{/* Warning for missing dependencies */}
				{(!githubIntegration || !anthropicIntegration) && (
					<div className="mb-4 p-3 bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 rounded-md">
						<div className="flex items-start gap-2">
							<AlertTriangle className="h-4 w-4 text-yellow-600 dark:text-yellow-400 mt-0.5 flex-shrink-0" />
							<div className="text-sm">
								<p className="font-medium text-yellow-800 dark:text-yellow-200 mb-1">
									Missing dependencies
								</p>
								<p className="text-yellow-700 dark:text-yellow-300">
									Your background agent requires both GitHub and Anthropic integrations to work
									properly.
								</p>
							</div>
						</div>
					</div>
				)}

				{/* Warning for redeploy required */}
				{githubIntegration && anthropicIntegration && redeployRequired && (
					<div className="mb-4 p-3 bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 rounded-md">
						<div className="flex items-start gap-2">
							<RefreshCw className="h-4 w-4 text-yellow-600 dark:text-yellow-400 mt-0.5 flex-shrink-0" />
							<div className="text-sm">
								<p className="font-medium text-yellow-800 dark:text-yellow-200 mb-1">
									Redeploy required
								</p>
								<p className="text-yellow-700 dark:text-yellow-300">
									The background agent won't work properly because you've recently changed one of
									your integrations. Please redeploy your agent to apply the changes.
								</p>
							</div>
						</div>
					</div>
				)}

				<div className="flex justify-center">
					<Button
						onClick={handleRedeploy}
						size="lg"
						disabled={deploying || deleting || !githubIntegration || !anthropicIntegration}
						className="px-6"
					>
						{deploying ? (
							<>
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
								Redeploying...
							</>
						) : (
							"Redeploy Agent"
						)}
					</Button>
				</div>
			</CardContent>
		</Card>
	);
}
