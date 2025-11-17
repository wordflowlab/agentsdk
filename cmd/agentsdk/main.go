package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			log.Fatalf("agentsdk serve failed: %v", err)
		}
	case "mcp-serve":
		if err := runMCPServe(os.Args[2:]); err != nil {
			log.Fatalf("agentsdk mcp-serve failed: %v", err)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  agentsdk serve [flags]")
	fmt.Println("  agentsdk mcp-serve [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  serve      Start an HTTP server (Gin-based architecture)")
	fmt.Println("  mcp-serve  Start an MCP HTTP server")
	fmt.Println()
	fmt.Println("Use 'agentsdk <subcommand> -h' for subcommand-specific flags.")
}
