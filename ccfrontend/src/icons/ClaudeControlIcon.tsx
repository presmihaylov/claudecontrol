import type { SVGProps } from "react";

interface ClaudeControlIconProps extends SVGProps<SVGSVGElement> {
	primaryColor?: "black" | "white" | string;
	secondaryColor?: "black" | "white" | string;
	// Legacy prop for backwards compatibility
	color?: "black" | "white" | string;
}

export function ClaudeControlIcon({ 
	primaryColor, 
	secondaryColor, 
	color = "black", 
	className, 
	...props 
}: ClaudeControlIconProps) {
	// Use new props if provided, otherwise fall back to legacy color prop
	const primary = primaryColor || color;
	const secondary = secondaryColor || "white";
	return (
		<svg
			role="img"
			viewBox="0 0 393 300"
			xmlns="http://www.w3.org/2000/svg"
			className={className}
			{...props}
		>
			<title>Claude Control</title>
			<path d="M60 0H0V300H60V0Z" fill={primary}/>
			<path d="M180 0H0V60H180V0Z" fill={primary}/>
			<path d="M180 240H0V300H180V240Z" fill={primary}/>
			<path d="M180 104V0H120V104H180Z" fill={primary}/>
			<path d="M180 300V185H120V300H180Z" fill={primary}/>
			<path d="M180 115V0H120V115H180Z" fill={primary}/>
			<path d="M393 300V185H333V300H393Z" fill={primary}/>
			<path d="M393 115V0H333V115H393Z" fill={primary}/>
			<path d="M393 104V0H333V104H393Z" fill={primary}/>
			<path d="M393 300V196H333V300H393Z" fill={primary}/>
			<path d="M213 0H273V300H213V0Z" fill={primary}/>
			<path d="M213 0H393V60H213V0Z" fill={primary}/>
			<path d="M213 240H393V300H213V240Z" fill={primary}/>
			<path d="M60 0H0V300H60V0Z" fill={primary}/>
			<path d="M180 0H0V60H180V0Z" fill={primary}/>
			<path d="M180 240H0V300H180V240Z" fill={primary}/>
			<path d="M180 300V185H120V300H180Z" fill={primary}/>
			<path d="M180 115V0H120V115H180Z" fill={primary}/>
			<path d="M180 30V0H150V30H180Z" fill={secondary}/>
			<path d="M180 300V270H150V300H180Z" fill={secondary}/>
			<path d="M30 300V270H0V300H30Z" fill={secondary}/>
			<path d="M30 30V0H0V30H30Z" fill={secondary}/>
			<path d="M393 300V185H333V300H393Z" fill={primary}/>
			<path d="M393 115V0H333V115H393Z" fill={primary}/>
			<path d="M213 0H273V300H213V0Z" fill={primary}/>
			<path d="M213 0H393V60H213V0Z" fill={primary}/>
			<path d="M213 240H393V300H213V240Z" fill={primary}/>
			<path d="M243 30V0H213V30H243Z" fill={secondary}/>
			<path d="M243 300V270H213V300H243Z" fill={secondary}/>
			<path d="M393 30V0H363V30H393Z" fill={secondary}/>
			<path d="M393 300V270H363V300H393Z" fill={secondary}/>
		</svg>
	);
}