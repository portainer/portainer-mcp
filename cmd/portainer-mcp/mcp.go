package main

import (
	"flag"
	"os"
	"strings"

	"github.com/portainer/portainer-mcp/internal/mcp"
	"github.com/portainer/portainer-mcp/internal/tooldef"
	"github.com/rs/zerolog/log"
)

const defaultToolsPath = "tools.yaml"

var (
	Version   string
	BuildDate string
	Commit    string
)

func main() {
	log.Info().
		Str("version", Version).
		Str("build-date", BuildDate).
		Str("commit", Commit).
		Msg("Portainer MCP server")

	serverFlag := flag.String("server", "", "The Portainer server URL")
	tokenFlag := flag.String("token", "", "The authentication token for the Portainer server")
	tokenFileFlag := flag.String("token-file", "", "Path to a file containing the authentication token")
	toolsFlag := flag.String("tools", "", "The path to the tools YAML file")
	readOnlyFlag := flag.Bool("read-only", false, "Run in read-only mode")
	disableVersionCheckFlag := flag.Bool("disable-version-check", false, "Disable Portainer server version check")
	skipTLSFlag := flag.Bool("skip-tls-verify", false, "Skip TLS certificate verification when connecting to Portainer (not recommended for production)")

	flag.Parse()

	if *serverFlag == "" {
		log.Fatal().Msg("-server flag is required")
	}

	token := *tokenFlag
	if token == "" && *tokenFileFlag != "" {
		data, err := os.ReadFile(*tokenFileFlag)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to read token file")
		}
		token = strings.TrimSpace(string(data))
	}
	if token == "" {
		token = os.Getenv("PORTAINER_TOKEN")
	}
	if token == "" {
		log.Fatal().Msg("Token is required: use -token flag, -token-file flag, or PORTAINER_TOKEN environment variable")
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
		Str("portainer-host", *serverFlag).
		Str("tools-path", toolsPath).
		Bool("read-only", *readOnlyFlag).
		Bool("disable-version-check", *disableVersionCheckFlag).
		Bool("skip-tls-verify", *skipTLSFlag).
		Msg("starting MCP server")

	server, err := mcp.NewPortainerMCPServer(*serverFlag, token, toolsPath, mcp.WithReadOnly(*readOnlyFlag), mcp.WithDisableVersionCheck(*disableVersionCheckFlag), mcp.WithSkipTLSVerify(*skipTLSFlag))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
	}

	server.AddEnvironmentFeatures()
	server.AddEnvironmentGroupFeatures()
	server.AddTagFeatures()
	server.AddStackFeatures()
	server.AddLocalStackFeatures()
	server.AddSettingsFeatures()
	server.AddUserFeatures()
	server.AddTeamFeatures()
	server.AddAccessGroupFeatures()
	server.AddDockerProxyFeatures()
	server.AddKubernetesProxyFeatures()

	err = server.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}
}
