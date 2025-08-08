// Simple test script to verify cursor model functionality
package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

type TestOptions struct {
	CursorModel string `long:"cursor-model" description:"Model to use with Cursor agent (only applies when --agent=cursor)"`
}

func main() {
	var opts TestOptions
	parser := flags.NewParser(&opts, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("CursorModel flag value: '%s'\n", opts.CursorModel)
	fmt.Println("Test completed successfully!")
}