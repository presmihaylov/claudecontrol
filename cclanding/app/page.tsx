"use client";

import Link from "next/link";
import Image from "next/image";
import { AnimateOnScroll } from "./components/animate-on-scroll";

export default function Home() {
	return (
		<main className="flex min-h-screen flex-col items-center justify-between md:pl-12 md:pr-12 pb-0">
			{/* Hero Section */}
			<div className="z-10 w-full max-w-5xl mx-auto items-center justify-center font-sans">
				<section className="flex flex-col items-center justify-center text-center py-12 md:pb-20 md:pt-32">
					<AnimateOnScroll>
						<h1 className="p-4 md:p-0 scroll-m-20 text-4xl font-extrabold tracking-tight lg:text-5xl mb-4">
							Deploy{" "}
							<span style={{ color: "rgb(217, 119, 87)" }}>Claude Code</span> in
							your Slack & Discord
						</h1>
					</AnimateOnScroll>
					<AnimateOnScroll delay={0.1}>
						<p className="p-4 md:p-0 text-xl leading-7 mb-8 max-w-2xl text-gray-300">
							Open pull requests, ask questions about your codebase, and connect
							any MCP server.
							<br />
							Self-hosted and open source - your whole team can use Claude Code
							without your data leaving your server.
						</p>
					</AnimateOnScroll>
					<AnimateOnScroll delay={0.2}>
						<Link
							href="https://app.claudecontrol.com"
							className="cursor-pointer inline-flex h-12 items-center justify-center rounded-md bg-white text-black px-8 py-3 text-lg font-medium shadow transition-all hover:bg-gray-200"
						>
							Get Started
						</Link>
					</AnimateOnScroll>

					{/* Platform Preview */}
					<AnimateOnScroll delay={0.4}>
						<div className="mt-16 w-full max-w-6xl">
							<div className="grid grid-cols-1 md:grid-cols-2 gap-8 items-start">
								{/* Slack Preview */}
								<div className="rounded-lg overflow-hidden h-96 md:h-[500px] bg-gray-900">
									<Image
										src="/slack-example.jpeg"
										alt="Claude Control Slack Integration Example"
										width={600}
										height={400}
										className="w-full h-full object-contain"
									/>
								</div>

								{/* Discord Preview */}
								<div className="rounded-lg overflow-hidden h-96 md:h-[500px] bg-gray-900">
									<Image
										src="/discord-example.jpeg"
										alt="Claude Control Discord Integration Example"
										width={600}
										height={400}
										className="w-full h-full object-contain"
									/>
								</div>
							</div>
						</div>
					</AnimateOnScroll>
				</section>

				{/* See it in action */}
				<hr className="border-gray-800 w-full max-w-4xl mx-auto" />
				<section className="py-12 rounded-lg pb-8 pt-8 max-w-4xl mx-auto">
					<AnimateOnScroll>
						<h2 className="text-4xl font-bold text-center mb-8">See it in action</h2>
					</AnimateOnScroll>
					
					{/* YouTube Video Embed */}
					<AnimateOnScroll delay={0.2}>
						<div className="mb-12 max-w-4xl mx-auto">
							<div className="relative w-full" style={{paddingBottom: '56.25%'}}>
								<iframe
									className="absolute top-0 left-0 w-full h-full rounded-lg"
									src="https://www.youtube.com/embed/dQw4w9WgXcQ"
									title="Claude Control Demo"
									frameBorder="0"
									allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
									allowFullScreen
								></iframe>
							</div>
						</div>
					</AnimateOnScroll>
				</section>

				{/* FAQ */}
				<hr className="border-gray-800 w-full max-w-4xl mx-auto" />
				<section className="py-12 rounded-lg pb-8 pt-8 max-w-4xl mx-auto">
					<AnimateOnScroll>
						<h2 className="text-4xl font-bold text-center mb-8">FAQ</h2>
					</AnimateOnScroll>

					<div className="space-y-8 max-w-3xl mx-auto">
						{/* First FAQ */}
						<AnimateOnScroll delay={0.1}>
							<div className="p-4 md:p-0">
								<h3 className="text-xl font-semibold mb-4 text-gray-200">How does it work?</h3>
								<p className="text-gray-300 leading-7">
									Deploy the <a href="https://github.com/presmihaylov/ccagent" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:text-blue-300 underline transition-colors">ccagent binary</a> on your infra. The agent communicates
									with our server, which sends requests to Slack and Discord.
									<br />
									Your code and data never leave your machine.
								</p>
							</div>
						</AnimateOnScroll>

						{/* Second FAQ */}
						<AnimateOnScroll delay={0.2}>
							<div className="p-4 md:p-0">
								<h3 className="text-xl font-semibold mb-4 text-gray-200">How much does it cost?</h3>
								<p className="text-gray-300 leading-7">
									It's free during beta.
								</p>
							</div>
						</AnimateOnScroll>

						{/* Third FAQ */}
						<AnimateOnScroll delay={0.3}>
							<div className="p-4 md:p-0">
								<h3 className="text-xl font-semibold mb-4 text-gray-200">How can I share feedback and feature requests?</h3>
								<p className="text-gray-300 leading-7">
									Contact me at <a href="mailto:support@pmihaylov.com" className="text-blue-400 hover:text-blue-300 underline transition-colors">support@pmihaylov.com</a> or use the chat widget in the app.
								</p>
							</div>
						</AnimateOnScroll>
					</div>
				</section>
			</div>

			{/* Footer */}
			<footer className="w-full max-w-5xl mx-auto border-t border-gray-800 p-6">
				<div className="flex justify-between items-center text-sm text-gray-400">
					<div>© 2025 Claude Control. All rights reserved.</div>
					<div className="flex gap-6">
						<Link
							href="/privacy"
							className="hover:text-white transition-colors"
						>
							Privacy Policy
						</Link>
						<Link href="/terms" className="hover:text-white transition-colors">
							Terms of Service
						</Link>
					</div>
				</div>
			</footer>
		</main>
	);
}
