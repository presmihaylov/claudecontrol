"use client";

import { useTheme } from "@/lib/theme-context";
import { Moon, Sun } from "lucide-react";
import { Button } from "./button";

export function ThemeToggle() {
	const { theme, toggleTheme } = useTheme();

	return (
		<Button variant="outline" size="sm" onClick={toggleTheme}>
			{theme === "light" ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4" />}
		</Button>
	);
}
