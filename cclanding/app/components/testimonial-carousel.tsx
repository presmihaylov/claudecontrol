"use client";

import Image from "next/image";

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
	// Duplicate the testimonials array to create seamless infinite scroll
	const duplicatedTestimonials = [...testimonials, ...testimonials];

	return (
		<div className="w-full overflow-hidden">
			<div className="flex animate-scroll">
				{duplicatedTestimonials.map((testimonial, index) => (
					<div
						key={index}
						className="flex-shrink-0 w-[460px] mx-4"
					>
						<a
							href={testimonial.link}
							target="_blank"
							rel="noopener noreferrer"
							className="block hover:opacity-90 transition-opacity"
						>
							<div className="rounded-lg shadow-lg bg-white overflow-hidden">
								<Image
									src={testimonial.image}
									alt={testimonial.alt}
									width={480}
									height={240}
									className="w-full h-auto"
								/>
							</div>
						</a>
					</div>
				))}
			</div>
			
			<style jsx>{`
				@keyframes scroll {
					0% {
						transform: translateX(0);
					}
					100% {
						transform: translateX(-50%);
					}
				}
				
				.animate-scroll {
					animation: scroll 20s linear infinite;
				}
			`}</style>
		</div>
	);
}