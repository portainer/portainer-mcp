package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/deviantony/portainer-mcp/internal/mcp"
)

const defaultToolsPath = "tools.yaml"

func main() {
	serverFlag := flag.String("server", "", "The Portainer server URL")
	tokenFlag := flag.String("token", "", "The authentication token for the Portainer server")
	toolsFlag := flag.String("tools", "", "The path to the tools YAML file")
	flag.Parse()

	if *serverFlag == "" || *tokenFlag == "" {
		log.Fatal("Both -server and -token flags are required")
	}

	toolsPath := *toolsFlag
	if toolsPath == "" {
		toolsPath = defaultToolsPath
	}

	log.Printf("Starting Portainer MCP server with server URL: %s and token: %s", *serverFlag, *tokenFlag)

	server, err := mcp.NewPortainerMCPServer(*serverFlag, *tokenFlag, toolsPath)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create server: %w", err))
	}

	server.AddEnvironmentFeatures()
	server.AddEnvironmentGroupFeatures()
	server.AddTagFeatures()
	server.AddStackFeatures()
	server.AddSettingsFeatures()
	server.AddUserFeatures()
	server.AddTeamFeatures()
	server.AddAccessGroupFeatures()

	err = server.Start()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to start server: %w", err))
	}
}
