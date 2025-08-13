"use client";

import Link from "next/link";
import Image from "next/image";
import { AnimateOnScroll } from "./components/animate-on-scroll";
import PlainChat from "./components/plain-chat";

export default function Home() {
	return (
		<main className="flex min-h-screen flex-col items-center justify-between md:pl-12 md:pr-12 pb-0">
			<PlainChat />
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
						<p className="p-4 md:p-0 text-xl leading-7 mb-8 max-w-2xl text-gray-600">
							Open pull requests, ask questions about your codebase, and connect
							any MCP server. Self-hosted and open source.
						</p>
					</AnimateOnScroll>
					<AnimateOnScroll delay={0.2}>
						<Link
							href="https://app.claudecontrol.com"
							className="cursor-pointer inline-flex h-12 items-center justify-center rounded-md bg-black text-white px-8 py-3 text-lg font-medium shadow transition-all hover:bg-gray-800"
						>
							Join Open Beta
						</Link>
					</AnimateOnScroll>

					{/* Platform Preview */}
					<AnimateOnScroll delay={0.4}>
						<div className="mt-16 w-full max-w-7xl">
							<div className="grid grid-cols-1 md:grid-cols-2 gap-12 items-start">
								{/* Slack Preview */}
								<div className="md:rounded-lg md:overflow-hidden md:h-[700px] md:bg-gray-100">
									<Image
										src="/slack-example.jpeg"
										alt="Claude Control Slack Integration Example"
										width={800}
										height={600}
										className="w-full h-auto md:h-full md:object-contain rounded-lg md:rounded-none"
									/>
								</div>

								{/* Discord Preview */}
								<div className="md:rounded-lg md:overflow-hidden md:h-[700px] md:bg-gray-100">
									<Image
										src="/discord-example.jpeg"
										alt="Claude Control Discord Integration Example"
										width={800}
										height={600}
										className="w-full h-auto md:h-full md:object-contain rounded-lg md:rounded-none"
									/>
								</div>
							</div>
						</div>
					</AnimateOnScroll>
				</section>

				{/* See it in action */}
				<hr className="border-gray-300 w-full max-w-4xl mx-auto" />
				<section className="py-12 rounded-lg pb-8 pt-8 max-w-4xl mx-auto">
					<AnimateOnScroll>
						<h2 className="text-4xl font-bold text-center mb-8">
							See it in action
						</h2>
					</AnimateOnScroll>

					{/* YouTube Video Embed */}
					<AnimateOnScroll delay={0.2}>
						<div className="mb-12 max-w-7xl mx-auto">
							<div
								className="relative w-full"
								style={{ paddingBottom: "56.25%" }}
							>
								<iframe
									className="absolute top-0 left-0 w-full h-full rounded-lg"
									src="https://www.youtube.com/embed/mZZu9h-980A"
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
				<hr className="border-gray-300 w-full max-w-4xl mx-auto" />
				<section className="py-12 rounded-lg pb-8 pt-8 max-w-4xl mx-auto">
					<AnimateOnScroll>
						<h2 className="text-4xl font-bold text-center mb-8">FAQ</h2>
					</AnimateOnScroll>

					<div className="space-y-8 max-w-3xl mx-auto">
						{/* First FAQ */}
						<AnimateOnScroll delay={0.1}>
							<div className="p-4 md:p-0">
								<h3 className="text-xl font-semibold mb-4 text-gray-900">
									How does it work?
								</h3>
								<p className="text-gray-600 leading-7">
									Deploy the{" "}
									<a
										href="https://github.com/presmihaylov/ccagent"
										target="_blank"
										rel="noopener noreferrer"
										className="text-blue-600 hover:text-blue-800 underline transition-colors"
									>
										ccagent binary
									</a>{" "}
									on your laptop or server. The agent communicates with the CC
									server, which sends messages to Slack and Discord.
									<br />
									Your code and data never leaves your machine.
								</p>
							</div>
						</AnimateOnScroll>

						{/* Second FAQ */}
						<AnimateOnScroll delay={0.2}>
							<div className="p-4 md:p-0">
								<h3 className="text-xl font-semibold mb-4 text-gray-900">
									How much does it cost?
								</h3>
								<p className="text-gray-600 leading-7">
									It's free during beta.
								</p>
							</div>
						</AnimateOnScroll>

						{/* Third FAQ */}
						<AnimateOnScroll delay={0.3}>
							<div className="p-4 md:p-0">
								<h3 className="text-xl font-semibold mb-4 text-gray-900">
									How can I share feedback?
								</h3>
								<p className="text-gray-600 leading-7">
									Contact us at{" "}
									<a
										href="mailto:support@claudecontrol.com"
										className="text-blue-600 hover:text-blue-800 underline transition-colors"
									>
										support@claudecontrol.com
									</a>{" "}
									or use the chat widget in the app.
								</p>
							</div>
						</AnimateOnScroll>
					</div>
				</section>
			</div>

			{/* Footer */}
			<footer className="w-full max-w-5xl mx-auto border-t border-gray-300 p-6">
				<div className="flex justify-between items-center text-sm text-gray-600">
					<div>© 2025 Claude Control. All rights reserved.</div>
					<div className="flex gap-6">
						<Link
							href="/privacy"
							className="hover:text-black transition-colors"
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
