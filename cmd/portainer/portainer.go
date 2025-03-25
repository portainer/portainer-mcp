package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/deviantony/portainer-mcp/internal/mcp"
)

func main() {
	serverFlag := flag.String("server", "", "The Portainer server URL")
	tokenFlag := flag.String("token", "", "The authentication token for the Portainer server")
	flag.Parse()

	if *serverFlag == "" || *tokenFlag == "" {
		log.Fatal("Both -server and -token flags are required")
	}

	log.Printf("Starting Portainer MCP server with server URL: %s and token: %s", *serverFlag, *tokenFlag)

	server := mcp.NewPortainerMCPServer(*serverFlag, *tokenFlag)

	server.AddEnvironmentFeatures()
	server.AddEnvironmentGroupFeatures()
	server.AddTagFeatures()
	server.AddStackFeatures()
	server.AddSettingsFeatures()
	server.AddUserFeatures()
	server.AddTeamFeatures()
	server.AddAccessGroupFeatures()

	err := server.Start()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to start server: %w", err))
	}
}
