package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/obsidian-mcp-server/obsidian-mcp-server/internal/mcp"
)

const (
	defaultBaseURL = "http://127.0.0.1:27123"
)

func main() {
	var (
		apiToken = flag.String("token", "", "Obsidian API token (can also be set via OBSIDIAN_API_TOKEN env var)")
		baseURL  = flag.String("url", defaultBaseURL, "Obsidian server base URL")
		version  = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		fmt.Println("obsidian-mcp-server v0.1.0")
		return
	}

	// Get API token from environment if not provided via flag
	if *apiToken == "" {
		*apiToken = os.Getenv("OBSIDIAN_API_TOKEN")
	}

	if *apiToken == "" {
		fmt.Fprintf(os.Stderr, "Error: API token is required. Use -token flag or set OBSIDIAN_API_TOKEN environment variable.\n")
		fmt.Fprintf(os.Stderr, "Usage: %s -token <your-api-token>\n", os.Args[0])
		os.Exit(1)
	}

	// Create and start the MCP server
	server := mcp.NewMCPServer(*apiToken, *baseURL)

	fmt.Fprintf(os.Stderr, "Starting Obsidian MCP Server...\n")
	fmt.Fprintf(os.Stderr, "Base URL: %s\n", *baseURL)
	fmt.Fprintf(os.Stderr, "Listening on stdin/stdout for MCP requests\n")

	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
