import Link from "next/link";
import Image from "next/image";

export default function Home() {
	return (
		<main className="flex min-h-screen flex-col items-center justify-between md:pl-12 md:pr-12 pb-0">

			{/* Hero Section */}
			<div className="z-10 w-full max-w-5xl mx-auto items-center justify-center font-sans">
				<section className="flex flex-col items-center justify-center text-center py-12 md:pb-20 md:pt-32">
					<h1 className="p-4 md:p-0 scroll-m-20 text-4xl font-extrabold tracking-tight lg:text-5xl mb-4">
						Deploy Claude Code in your Slack & Discord
					</h1>
					<p className="p-4 md:p-0 text-xl leading-7 mb-8 max-w-2xl">
						Open pull requests, ask questions about your codebase, and connect
						any MCP server.
						<br />
						Self-hosted and open source - your whole team can use Claude Code
						without your data leaving your server.
					</p>
					<Link
						href="https://app.claudecontrol.com"
						className="cursor-pointer inline-flex h-12 items-center justify-center rounded-md bg-black text-white px-8 py-3 text-lg font-medium shadow transition-all hover:bg-gray-800"
					>
						Get Started
					</Link>

					{/* Platform Preview */}
					<div className="mt-16 w-full max-w-4xl">
						<div className="grid grid-cols-1 md:grid-cols-2 gap-8 items-start">
							{/* Slack Preview */}
							<div className="rounded-lg overflow-hidden">
								<Image
									src="/slack-example.jpeg"
									alt="Claude Control Slack Integration Example"
									width={600}
									height={400}
									className="w-full h-auto"
								/>
							</div>

							{/* Discord Preview */}
							<div className="rounded-lg overflow-hidden">
								<Image
									src="/discord-example.jpeg"
									alt="Claude Control Discord Integration Example"
									width={600}
									height={400}
									className="w-full h-auto"
								/>
							</div>
						</div>
					</div>
				</section>

				{/* How It Works */}
				<hr className="border-gray-200 w-full max-w-4xl mx-auto" />
				<section className="py-12 rounded-lg pb-8 pt-8 max-w-4xl mx-auto">
					<h2 className="text-3xl font-bold text-center mb-8">How it works</h2>

					{/* High-level description */}
					<div className="mb-12 max-w-2xl mx-auto">
						<p className="p-4 md:p-0 text-xl leading-7 text-center">
							Deploy our ccagent binary on your infra. The agent communicates
							with our server, which sends requests to Slack and Discord.
							<br />
							Your code and data never leave your machine.
						</p>
					</div>
				</section>
			</div>

			{/* Footer */}
			<footer className="w-full max-w-5xl mx-auto border-t border-gray-200 p-6">
				<div className="flex justify-between items-center text-sm text-gray-600">
					<div>© 2025 Claude Control. All rights reserved.</div>
					<div className="flex gap-6">
						<Link
							href="/privacy"
							className="hover:text-black transition-colors"
						>
							Privacy Policy
						</Link>
						<Link href="/terms" className="hover:text-black transition-colors">
							Terms of Service
						</Link>
					</div>
				</div>
			</footer>
		</main>
	);
}
