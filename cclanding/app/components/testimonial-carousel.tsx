"use client";

import Image from "next/image";
import { motion } from "motion/react";
import { animate } from "motion";

const testimonials = [
	{
		image: "/testimonials/testimonial1.png",
		link: "https://x.com/CodyDetails/status/1954974595305574909",
		alt: "Cody's testimonial about Claude Control",
	},
	{
		image: "/testimonials/testimonial2.png",
		link: "https://x.com/khoi_danny/status/1953622574157770987",
		alt: "Khoi Nguyen's testimonial about productivity boost",
	},
	{
		image: "/testimonials/testimonial3.png",
		link: "https://x.com/T_Zahil/status/1953455072748208224",
		alt: "T_Zahil's testimonial about Claude Control",
	},
	{
		image: "/testimonials/testimonial4.png",
		link: "https://x.com/lioloc_dev/status/1953734925347098827",
		alt: "Lioloc's testimonial about Claude Control",
	},
	{
		image: "/testimonials/testimonial5.png",
		link: "https://x.com/PaulRBerg/status/1956015087279337486",
		alt: "Paul Berg's testimonial about Claude Control",
	},
	{
		image: "/testimonials/testimonial6.png",
		link: "#",
		alt: "Community testimonial about Claude Control",
	},
];

export default function TestimonialCarousel() {
	// Calculate total width for smooth infinite scroll
	const testimonialWidth = 460 + 70; // 460px + 70px margin
	const repeatTimes = 5;
	const animateDuration = 15;
	const totalWidth = testimonials.length * testimonialWidth;
	const carousel = Array(10).fill(testimonials).flat();

	return (
		<div className="w-full overflow-hidden">
			<motion.div
				className="flex"
				animate={{
					x: [0, -totalWidth * repeatTimes],
				}}
				transition={{
					duration: animateDuration * repeatTimes,
					repeat: Infinity,
					ease: "linear",
				}}
			>
				{/* Render testimonials twice for seamless loop */}
				{carousel.map((testimonial, index) => (
					<div key={index} className="flex-shrink-0 w-[460px] mx-2">
						<a
							href={testimonial.link}
							target="_blank"
							rel="noopener noreferrer"
							className="block hover:opacity-90 transition-opacity"
						>
							<div className="rounded-lg bg-white overflow-hidden h-64 flex items-center justify-center p-2">
								<Image
									src={testimonial.image}
									alt={testimonial.alt}
									width={480}
									height={240}
									className="max-w-full max-h-full object-contain"
								/>
							</div>
						</a>
					</div>
				))}
			</motion.div>
		</div>
	);
}
