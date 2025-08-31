"use client";

import { env } from "@/lib/env";
import { useAuth } from "@clerk/nextjs";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect } from "react";

function GitHubRedirectContent() {
	const { getToken } = useAuth();
	const router = useRouter();
	const searchParams = useSearchParams();

	useEffect(() => {
		const code = searchParams.get("code");
		const installationId = searchParams.get("installation_id");
		const setupAction = searchParams.get("setup_action");

		if (!code || !installationId) {
			router.push("/onboarding?error=github-callback-invalid");
			return;
		}

		if (setupAction === "install") {
			handleGitHubIntegration(code, installationId);
		} else {
			router.push("/onboarding?error=github-setup-invalid");
		}
	}, [searchParams, router, getToken]);

	const handleGitHubIntegration = async (code: string, installationId: string) => {
		try {
			const token = await getToken();
			if (!token) {
				router.push("/onboarding?error=auth-required");
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
				console.error("GitHub integration failed:", errorText);
				router.push("/onboarding?error=github-integration-failed");
				return;
			}

			// Success - redirect to main page or onboarding
			router.push("/");
		} catch (err) {
			console.error("Error creating GitHub integration:", err);
			router.push("/onboarding?error=github-integration-failed");
		}
	};

	return null; // Blank transient page
}

export default function GitHubRedirectPage() {
	return (
		<Suspense fallback={null}>
			<GitHubRedirectContent />
		</Suspense>
	);
}