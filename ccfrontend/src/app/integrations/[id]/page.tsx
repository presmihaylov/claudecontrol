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
import { Input } from "@/components/ui/input";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { ArrowLeft, Copy, Download, Key, RefreshCw, Slack } from "lucide-react";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

interface SlackIntegration {
	id: string;
	slack_team_id: string;
	slack_team_name: string;
	user_id: string;
	ccagent_secret_key_generated_at: string | null;
	created_at: string;
	updated_at: string;
}

interface CCAgentSecretKeyResponse {
	secret_key: string;
}

export default function IntegrationDetail() {
	const params = useParams();
	const router = useRouter();
	const { isLoaded, isSignedIn, getToken, signOut } = useAuth();
	const [integration, setIntegration] = useState<SlackIntegration | null>(null);
	const [loading, setLoading] = useState(true);
	const [generatingKey, setGeneratingKey] = useState(false);
	const [regenerateDialogOpen, setRegenerateDialogOpen] = useState(false);
	const [secretKeyDialogOpen, setSecretKeyDialogOpen] = useState(false);
	const [generatedSecretKey, setGeneratedSecretKey] = useState<string>("");
	const [copySuccess, setCopySuccess] = useState(false);

	const integrationId = params.id as string;

	useEffect(() => {
		const fetchIntegration = async () => {
			if (!isLoaded || !isSignedIn) return;

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

				const integrations: SlackIntegration[] = await response.json();
				const foundIntegration = integrations.find((i) => i.id === integrationId);

				if (!foundIntegration) {
					router.push("/");
					return;
				}

				setIntegration(foundIntegration);
			} catch (error) {
				console.error("Error fetching integration:", error);
			} finally {
				setLoading(false);
			}
		};

		fetchIntegration();
	}, [isLoaded, isSignedIn, getToken, integrationId, router]);

	const handleGenerateSecretKey = async () => {
		setGeneratingKey(true);
		setRegenerateDialogOpen(false);

		try {
			const token = await getToken();
			if (!token) return;

			const response = await fetch(
				`${env.CCBACKEND_BASE_URL}/slack/integrations/${integrationId}/ccagent_secret_key`,
				{
					method: "POST",
					headers: {
						Authorization: `Bearer ${token}`,
						"Content-Type": "application/json",
					},
				},
			);

			if (!response.ok) {
				console.error("Failed to generate secret key:", response.statusText);
				alert("Failed to generate secret key. Please try again.");
				return;
			}

			const data: CCAgentSecretKeyResponse = await response.json();
			setGeneratedSecretKey(data.secret_key);
			setSecretKeyDialogOpen(true);

			// Update the integration to reflect the new timestamp
			if (integration) {
				setIntegration({
					...integration,
					ccagent_secret_key_generated_at: new Date().toISOString(),
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

	if (!integration) {
		return (
			<div className="flex items-center justify-center min-h-screen">
				<div className="text-muted-foreground">Integration not found</div>
			</div>
		);
	}

	const hasSecretKey = integration.ccagent_secret_key_generated_at !== null;

	return (
		<div className="min-h-screen bg-background">
			<header className="border-b">
				<div className="container mx-auto px-4 py-4 flex items-center justify-between">
					<Button variant="ghost" size="sm" onClick={() => router.back()}>
						<ArrowLeft className="h-4 w-4 mr-2" />
						Back
					</Button>
					<h1 className="text-2xl font-semibold">Claude Control</h1>
					<Button variant="outline" size="sm" onClick={() => signOut()}>
						Logout
					</Button>
				</div>
			</header>

			<div className="container mx-auto px-4 py-8 max-w-4xl">
				<div className="space-y-6">
					{/* Integration Info */}
					<Card>
						<CardHeader>
							<div className="flex items-center gap-3">
								<Slack className="h-8 w-8 text-black" />
								<div>
									<CardTitle>{integration.slack_team_name}</CardTitle>
									<CardDescription>
										Connected on {new Date(integration.created_at).toLocaleDateString()}
									</CardDescription>
								</div>
							</div>
						</CardHeader>
					</Card>

					{/* ccagent Secret Key Section */}
					<Card>
						<CardHeader>
							<CardTitle className="flex items-center gap-2">
								<Key className="h-5 w-5" />
								ccagent Secret Key
							</CardTitle>
							<CardDescription>
								This secret key is used by the ccagent CLI to authenticate with your Slack
								workspace.
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							{/* Download ccagent Button */}
							<div className="flex items-center justify-between p-4 border rounded-lg bg-muted/50">
								<div className="space-y-1">
									<h4 className="font-medium">Download ccagent</h4>
									<p className="text-sm text-muted-foreground">
										Download the ccagent CLI tool to start using Claude Control with your Slack
										workspace.
									</p>
								</div>
								<Button
									variant="outline"
									onClick={() =>
										window.open(
											"https://drive.google.com/drive/folders/12M6c9Ql9PObqKBPWbrHCMcNCcvXHMLFH?usp=sharing",
											"_blank",
										)
									}
									className="flex items-center gap-2"
								>
									<Download className="h-4 w-4" />
									Download
								</Button>
							</div>
							{hasSecretKey ? (
								<div className="space-y-2">
									<p className="text-sm text-muted-foreground">
										Secret key generated on{" "}
										{integration.ccagent_secret_key_generated_at &&
											new Date(
												integration.ccagent_secret_key_generated_at,
											).toLocaleDateString()}{" "}
										at{" "}
										{integration.ccagent_secret_key_generated_at &&
											new Date(integration.ccagent_secret_key_generated_at).toLocaleTimeString()}
									</p>
									<Button
										variant="outline"
										onClick={() => setRegenerateDialogOpen(true)}
										disabled={generatingKey}
										className="flex items-center gap-2"
									>
										<RefreshCw className={`h-4 w-4 ${generatingKey ? "animate-spin" : ""}`} />
										{generatingKey ? "Regenerating..." : "Regenerate Secret Key"}
									</Button>
								</div>
							) : (
								<div className="space-y-2">
									<p className="text-sm text-muted-foreground">
										No secret key has been generated yet. Generate one to start using ccagent with
										this workspace.
									</p>
									<Button
										onClick={handleGenerateSecretKey}
										disabled={generatingKey}
										className="flex items-center gap-2"
									>
										<Key className={`h-4 w-4 ${generatingKey ? "animate-spin" : ""}`} />
										{generatingKey ? "Generating..." : "Generate Secret Key"}
									</Button>
								</div>
							)}
						</CardContent>
					</Card>
				</div>

				{/* Regenerate Warning Dialog */}
				<Dialog open={regenerateDialogOpen} onOpenChange={setRegenerateDialogOpen}>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Regenerate Secret Key</DialogTitle>
							<DialogDescription>
								Are you sure you want to regenerate the secret key for "
								{integration.slack_team_name}"?
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
