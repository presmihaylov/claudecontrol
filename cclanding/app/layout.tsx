import type { Metadata } from "next";
import { Geist, Geist_Mono, Orbitron } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const orbitron = Orbitron({
  variable: "--font-orbitron",
  subsets: ["latin"],
  weight: ["400", "700", "900"],
});

export const metadata: Metadata = {
  title: "Claude Control - AI Agent for Your Team",
  description: "Deploy AI agents that interact with your codebase directly from Slack and Discord. Open pull requests, ask questions, and connect MCP servers with your AI co-worker.",
  icons: {
    icon: '/icon.svg',
    shortcut: '/icon.svg',
    apple: '/icon.svg',
  },
  openGraph: {
    title: "Claude Control - AI Agent for Your Team",
    description: "Deploy AI agents that interact with your codebase directly from Slack and Discord. Open pull requests, ask questions, and connect MCP servers with your AI co-worker.",
    images: ['/ogimage.png'],
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: "Claude Control - AI Agent for Your Team",
    description: "Deploy AI agents that interact with your codebase directly from Slack and Discord. Open pull requests, ask questions, and connect MCP servers with your AI co-worker.",
    images: ['/ogimage.png'],
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full">
      <head>
        <link rel="icon" href="/icon.svg" type="image/svg+xml" />
        <link rel="shortcut icon" href="/icon.svg" />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} ${orbitron.variable} antialiased min-h-full bg-white text-black font-sans`}
      >
        {children}
      </body>
    </html>
  );
}
