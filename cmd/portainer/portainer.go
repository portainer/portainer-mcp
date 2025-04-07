package main

import (
	"flag"

	"github.com/portainer/portainer-mcp/internal/mcp"
	"github.com/rs/zerolog/log"
)

const defaultToolsPath = "tools.yaml"

func main() {
	serverFlag := flag.String("server", "", "The Portainer server URL")
	tokenFlag := flag.String("token", "", "The authentication token for the Portainer server")
	toolsFlag := flag.String("tools", "", "The path to the tools YAML file")
	flag.Parse()

	if *serverFlag == "" || *tokenFlag == "" {
		log.Fatal().Msg("Both -server and -token flags are required")
	}

	toolsPath := *toolsFlag
	if toolsPath == "" {
		toolsPath = defaultToolsPath
	}

	log.Info().
		Str("server", *serverFlag).
		Str("token", *tokenFlag).
		Msg("Starting Portainer MCP server")

	server, err := mcp.NewPortainerMCPServer(*serverFlag, *tokenFlag, toolsPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
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
		log.Fatal().Err(err).Msg("failed to start server")
	}
}
