import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { ClerkProvider, SignedIn, UserButton } from "@clerk/nextjs";

const geistSans = Geist({
	variable: "--font-geist-sans",
	subsets: ["latin"],
});

const geistMono = Geist_Mono({
	variable: "--font-geist-mono",
	subsets: ["latin"],
});

export const metadata: Metadata = {
	title: "Claude Control",
	description: "Claude Control Dashboard",
};

export default function RootLayout({
	children,
}: Readonly<{
	children: React.ReactNode;
}>) {
	return (
		<ClerkProvider>
			<html lang="en" className="h-full">
				<body
					className={`${geistSans.variable} ${geistMono.variable} antialiased min-h-full bg-background`}
				>
					<div className="min-h-screen bg-gradient-to-b from-background to-muted/20">
						<header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
							<div className="container flex h-14 max-w-screen-2xl items-center justify-end pr-4">
								<SignedIn>
									<UserButton
										appearance={{
											elements: {
												avatarBox: "w-8 h-8",
												userButtonAvatarBox: "w-8 h-8",
											},
										}}
									/>
								</SignedIn>
							</div>
						</header>
						<main className="flex-1">{children}</main>
					</div>
				</body>
			</html>
		</ClerkProvider>
	);
}
