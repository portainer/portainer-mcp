package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/portainer/portainer-mcp/pkg/toolgen"
	"github.com/rs/zerolog/log"
)

// AnthropicTool defines the structure expected by the Anthropic API
type AnthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
	// Annotations any    `json:"annotations"` // Annotations are currently not supported by the Anthropic API
}

func main() {
	inputYamlPath := flag.String("input", "", "Path to the input tools YAML file (mandatory)")
	outputPath := flag.String("output", "", "Path to the output JSON file (mandatory)")
	flag.Parse()

	if *inputYamlPath == "" {
		log.Fatal().Msg("Input YAML path is mandatory. Please specify using -input flag.")
	}
	if *outputPath == "" {
		log.Fatal().Msg("Output path is mandatory. Please specify using -output flag.")
	}

	tools, err := toolgen.LoadToolsFromYAML(*inputYamlPath, "1.0")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load tools")
	}

	// Convert map[string]mcp.Tool to []AnthropicTool for correct JSON structure
	var anthropicToolList []AnthropicTool
	for _, tool := range tools {
		// Only include fields expected by Anthropic
		anthropicTool := AnthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema, // Assuming mcp.Tool has InputSchema field
			// Annotations: tool.Annotations, // Removed annotations
		}
		anthropicToolList = append(anthropicToolList, anthropicTool)
	}

	jsonData, err := json.MarshalIndent(anthropicToolList, "", "  ")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to marshal tools to JSON")
	}

	err = os.WriteFile(*outputPath, jsonData, 0644)
	if err != nil {
		log.Fatal().Err(err).Str("path", *outputPath).Msg("failed to write JSON to file")
	}

	log.Info().Str("path", *outputPath).Msg("Successfully wrote tools to JSON file")
}
