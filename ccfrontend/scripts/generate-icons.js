#!/usr/bin/env node

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const rawIconsDir = path.join(__dirname, "../src/icons/raw");
const iconsDir = path.join(__dirname, "../src/icons");

// Template for React icon components
const componentTemplate = (
	componentName,
	title,
	pathData,
) => `import type { SVGProps } from "react";

interface ${componentName}Props extends SVGProps<SVGSVGElement> {
	color?: "black" | "white" | string;
}

export function ${componentName}({ color = "black", className, ...props }: ${componentName}Props) {
	return (
		<svg
			role="img"
			viewBox="0 0 24 24"
			xmlns="http://www.w3.org/2000/svg"
			className={className}
			{...props}
		>
			<title>${title}</title>
			<path
				fill={color}
				d="${pathData}"
			/>
		</svg>
	);
}`;

// Function to extract path data from SVG
function extractPathFromSVG(svgContent) {
	const pathMatch = svgContent.match(/<path[^>]*d="([^"]*)"[^>]*\/?>(?:<\/path>)?/);
	return pathMatch ? pathMatch[1] : "";
}

// Function to extract title from SVG
function extractTitleFromSVG(svgContent) {
	const titleMatch = svgContent.match(/<title>([^<]*)<\/title>/);
	return titleMatch ? titleMatch[1] : "Icon";
}

// Function to convert filename to component name
function filenameToComponentName(filename) {
	const name = path.basename(filename, ".svg");
	return (
		name
			.split(/[-_]/)
			.map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
			.join("") + "Icon"
	);
}

// Main generation function
function generateIconComponents() {
	try {
		// Read all SVG files from raw directory
		const svgFiles = fs.readdirSync(rawIconsDir).filter((file) => file.endsWith(".svg"));

		console.log(`Found ${svgFiles.length} SVG files to process:`);

		svgFiles.forEach((file) => {
			const svgPath = path.join(rawIconsDir, file);
			const svgContent = fs.readFileSync(svgPath, "utf-8");

			// Extract data
			const pathData = extractPathFromSVG(svgContent);
			const title = extractTitleFromSVG(svgContent);
			const componentName = filenameToComponentName(file);

			if (!pathData) {
				console.warn(`⚠️  Warning: No path data found in ${file}`);
				return;
			}

			// Generate component
			const componentCode = componentTemplate(componentName, title, pathData);

			// Write component file
			const componentFilename = `${componentName}.tsx`;
			const componentPath = path.join(iconsDir, componentFilename);

			fs.writeFileSync(componentPath, componentCode, "utf-8");

			console.log(`✅ Generated ${componentFilename} from ${file}`);
		});

		// Generate index file for easy imports
		const indexContent = svgFiles
			.map((file) => {
				const componentName = filenameToComponentName(file);
				return `export { ${componentName} } from "./${componentName}";`;
			})
			.join("\n");

		fs.writeFileSync(path.join(iconsDir, "index.ts"), indexContent, "utf-8");
		console.log("✅ Generated index.ts");

		console.log("\n🎉 Icon generation complete!");
	} catch (error) {
		console.error("❌ Error generating icons:", error.message);
		process.exit(1);
	}
}

// Run the script
generateIconComponents();
