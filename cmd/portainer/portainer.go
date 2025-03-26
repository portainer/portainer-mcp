package main

import (
	"flag"
	"log"
	"os"

	"github.com/deviantony/portainer-mcp/internal/mcp"
)

func main() {
	// Parse command line flags
	serverFlag := flag.String("server", "", "The Portainer server URL")
	tokenFlag := flag.String("token", "", "The authentication token for the Portainer server")
	skipTLSVerifyFlag := flag.Bool("skip-tls-verify", true, "Skip TLS certificate verification")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	// Validate required flags
	if *serverFlag == "" || *tokenFlag == "" {
		log.Fatal("Both -server and -token flags are required")
	}

	// Create logger
	logger := log.New(os.Stderr, "[PORTAINER-MCP] ", log.LstdFlags)
	if *verboseFlag {
		logger.Println("Verbose logging enabled")
	}

	// Create server options
	options := []mcp.ServerOption{
		mcp.WithLogger(logger),
		mcp.WithSkipTLSVerify(*skipTLSVerifyFlag),
	}

	// Create and start server
	logger.Printf("Starting Portainer MCP server with server URL: %s", *serverFlag)
	server := mcp.NewPortainerMCPServer(*serverFlag, *tokenFlag, options...)

	// Register all features
	server.AddFeatures()

	// Start server
	err := server.Start()
	if err != nil {
		log.Fatal(err)
	}
}