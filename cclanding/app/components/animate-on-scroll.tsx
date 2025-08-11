"use client";

import { motion, useInView } from "framer-motion";
import { ReactNode, useRef } from "react";

type AnimateOnScrollProps = {
	children: ReactNode;
	className?: string;
	delay?: number;
	duration?: number;
	once?: boolean;
};

export function AnimateOnScroll({
	children,
	className = "",
	delay = 0,
	duration = 0.5,
	once = true,
}: AnimateOnScrollProps) {
	const ref = useRef<HTMLDivElement>(null);
	const isInView = useInView(ref, { once, amount: 0.3 });

	const variants = {
		hidden: { opacity: 0, y: 20 },
		visible: { opacity: 1, y: 0 },
	};

	return (
		<motion.div
			ref={ref}
			initial="hidden"
			animate={isInView ? "visible" : "hidden"}
			variants={variants}
			transition={{
				duration,
				delay,
				ease: "easeOut",
			}}
			className={className}
		>
			{children}
		</motion.div>
	);
}