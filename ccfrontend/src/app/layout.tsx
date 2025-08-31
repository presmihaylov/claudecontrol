import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import PlainChatAuthenticated from "@/components/plain-chat-authenticated";
import { Toaster } from "@/components/ui/toaster";
import { ClerkProvider } from "@clerk/nextjs";

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
	icons: {
		icon: "/icon.svg",
		shortcut: "/icon.svg",
		apple: "/icon.svg",
	},
};

export default function RootLayout({
	children,
}: Readonly<{
	children: React.ReactNode;
}>) {
	return (
		<ClerkProvider>
			<html lang="en" className="h-full">
				<head>
					<link rel="icon" href="/icon.svg" type="image/svg+xml" />
					<link rel="shortcut icon" href="/icon.svg" />
				</head>
				<body
					className={`${geistSans.variable} ${geistMono.variable} antialiased min-h-full bg-background`}
				>
					{children}
					<PlainChatAuthenticated />
					<Toaster />
				</body>
			</html>
		</ClerkProvider>
	);
}
