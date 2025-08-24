"use client";

import { Card, CardContent } from "@/components/ui/card";
import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { CheckCircle, Loader2, XCircle } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";

function GitHubRedirectContent() {
	const { getToken } = useAuth();
	const router = useRouter();
	const searchParams = useSearchParams();
	const [status, setStatus] = useState<"processing" | "success" | "error">("processing");
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		const code = searchParams.get("code");
		const installationId = searchParams.get("installation_id");
		const setupAction = searchParams.get("setup_action");

		if (!code || !installationId) {
			setStatus("error");
			setError("Invalid GitHub callback - missing required parameters");
			return;
		}

		if (setupAction === "install") {
			handleGitHubIntegration(code, installationId);
		} else {
			setStatus("error");
			setError("Unsupported setup action");
		}
	}, [searchParams]);

	const handleGitHubIntegration = async (code: string, installationId: string) => {
		try {
			const token = await getToken();
			if (!token) {
				setStatus("error");
				setError("Authentication required");
				return;
			}

			const response = await fetch(`${env.CCBACKEND_BASE_URL}/github/integrations`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${token}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					code,
					installation_id: installationId,
				}),
			});

			if (!response.ok) {
				const errorText = await response.text();
				if (response.status === 401) {
					throw new Error("Failed to verify GitHub installation. Please try again.");
				}
				throw new Error(errorText || "Failed to create GitHub integration");
			}

			setStatus("success");

			// Redirect to main page after a short delay
			setTimeout(() => {
				router.push("/");
			}, 2000);
		} catch (err) {
			console.error("Error creating GitHub integration:", err);
			setStatus("error");
			setError(err instanceof Error ? err.message : "Failed to create GitHub integration");
		}
	};

	return (
		<div className="flex min-h-screen items-center justify-center p-4">
			<Card className="w-full max-w-md">
				<CardContent className="pt-6">
					{status === "processing" && (
						<div className="flex flex-col items-center space-y-4">
							<Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
							<p className="text-center text-muted-foreground">Verifying GitHub installation...</p>
						</div>
					)}

					{status === "success" && (
						<div className="flex flex-col items-center space-y-4">
							<CheckCircle className="h-8 w-8 text-green-600" />
							<div className="text-center">
								<p className="font-medium">GitHub integration successful!</p>
								<p className="mt-2 text-sm text-muted-foreground">Redirecting to dashboard...</p>
							</div>
						</div>
					)}

					{status === "error" && (
						<div className="flex flex-col items-center space-y-4">
							<XCircle className="h-8 w-8 text-destructive" />
							<div className="text-center">
								<p className="font-medium">Integration failed</p>
								<p className="mt-2 text-sm text-muted-foreground">{error}</p>
								<a href="/onboarding" className="mt-4 inline-block text-sm text-primary underline">
									Try again
								</a>
							</div>
						</div>
					)}
				</CardContent>
			</Card>
		</div>
	);
}

export default function GitHubRedirectPage() {
	return (
		<Suspense
			fallback={
				<div className="flex min-h-screen items-center justify-center p-4">
					<Card className="w-full max-w-md">
						<CardContent className="pt-6">
							<div className="flex flex-col items-center space-y-4">
								<Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
								<p className="text-center text-muted-foreground">Loading...</p>
							</div>
						</CardContent>
					</Card>
				</div>
			}
		>
			<GitHubRedirectContent />
		</Suspense>
	);
}
