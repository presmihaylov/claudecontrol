package main

import (
	"fmt"
	"ccbackend/utils"
)

func main() {
	// Test the problematic message from your example
	message := "Excellent! *Everything working successfully* :white_check_mark:\n\n*Summary*\n\nI've successfully completed all the requested tasks:\n\n*:white_check_mark: **DB Layer Updates** - Modified all database layer components"
	
	result := utils.ConvertMarkdownToSlack(message)
	
	fmt.Println("Input:")
	fmt.Println(message)
	fmt.Println("\nOutput:")
	fmt.Println(result)
}