package main

import (
	"flag"

	"github.com/portainer/portainer-mcp/internal/mcp"
	"github.com/portainer/portainer-mcp/internal/tooldef"
	"github.com/rs/zerolog/log"
)

const defaultToolsPath = "tools.yaml"
const version = "0.1.0"

func main() {
	serverFlag := flag.String("server", "", "The Portainer server URL")
	tokenFlag := flag.String("token", "", "The authentication token for the Portainer server")
	toolsFlag := flag.String("tools", "", "The path to the tools YAML file")
	readOnlyFlag := flag.Bool("read-only", false, "Run in read-only mode")
	flag.Parse()

	if *serverFlag == "" || *tokenFlag == "" {
		log.Fatal().Msg("Both -server and -token flags are required")
	}

	toolsPath := *toolsFlag
	if toolsPath == "" {
		toolsPath = defaultToolsPath
	}

	// We first check if the tools.yaml file exists
	// We'll create it from the embedded version if it doesn't exist
	exists, err := tooldef.CreateToolsFileIfNotExists(toolsPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create tools.yaml file")
	}

	if exists {
		log.Info().Msg("using existing tools.yaml file")
	} else {
		log.Info().Msg("created tools.yaml file")
	}

	log.Info().
		Str("server", *serverFlag).
		Str("token", *tokenFlag).
		Str("tools", toolsPath).
		Bool("read-only", *readOnlyFlag).
		Str("version", version).
		Msg("Starting Portainer MCP server")

	server, err := mcp.NewPortainerMCPServer(*serverFlag, *tokenFlag, toolsPath, mcp.WithReadOnly(*readOnlyFlag))
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
