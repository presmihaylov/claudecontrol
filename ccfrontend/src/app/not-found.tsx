import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import Link from "next/link";

export default function NotFound() {
	return (
		<div className="container mx-auto py-8 px-4 max-w-2xl">
			<div className="flex items-center justify-center min-h-[60vh]">
				<Card className="w-full max-w-md text-center">
					<CardHeader>
						<CardTitle className="text-4xl font-bold">404</CardTitle>
						<CardDescription className="text-lg">Page not found</CardDescription>
					</CardHeader>
					<CardContent>
						<p className="text-muted-foreground mb-6">
							The page you're looking for doesn't exist or has been moved.
						</p>
						<Button asChild>
							<Link href="/">Return to Dashboard</Link>
						</Button>
					</CardContent>
				</Card>
			</div>
		</div>
	);
}
