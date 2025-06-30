package toolgen

import (
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// ToolsConfig represents the entire YAML configuration
type ToolsConfig struct {
	Version string           `yaml:"version"`
	Tools   []ToolDefinition `yaml:"tools"`
}

// ToolDefinition represents a single tool in the YAML config
type ToolDefinition struct {
	Name        string                `yaml:"name"`
	Description string                `yaml:"description"`
	Parameters  []ParameterDefinition `yaml:"parameters"`
	Annotations Annotations           `yaml:"annotations"`
}

// ParameterDefinition represents a tool parameter in the YAML config
type ParameterDefinition struct {
	Name        string         `yaml:"name"`
	Type        string         `yaml:"type"`
	Required    bool           `yaml:"required"`
	Enum        []string       `yaml:"enum,omitempty"`
	Description string         `yaml:"description"`
	Items       map[string]any `yaml:"items,omitempty"`
}

// Annotations represents a tool annotations in the YAML config
type Annotations struct {
	Title           string `yaml:"title"`
	ReadOnlyHint    bool   `yaml:"readOnlyHint"`
	DestructiveHint bool   `yaml:"destructiveHint"`
	IdempotentHint  bool   `yaml:"idempotentHint"`
	OpenWorldHint   bool   `yaml:"openWorldHint"`
}

// LoadToolsFromYAML loads tool definitions from a YAML file
// It returns the tools and the version of the tools.yaml file
func LoadToolsFromYAML(filePath string, minimumVersion string) (map[string]mcp.Tool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config ToolsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.Version == "" {
		return nil, fmt.Errorf("missing version in tools.yaml")
	}

	if !semver.IsValid(config.Version) {
		return nil, fmt.Errorf("invalid version in tools.yaml: %s", config.Version)
	}

	if semver.Compare(config.Version, minimumVersion) < 0 {
		return nil, fmt.Errorf("tools.yaml version %s is below the minimum required version %s", config.Version, minimumVersion)
	}

	return convertToolDefinitions(config.Tools), nil
}

// convertToolDefinitions converts YAML tool definitions to mcp.Tool objects
func convertToolDefinitions(defs []ToolDefinition) map[string]mcp.Tool {
	tools := make(map[string]mcp.Tool, len(defs))

	for _, def := range defs {
		tool, err := convertToolDefinition(def)
		if err != nil {
			log.Printf("skipping invalid tool definition %s: %s", def.Name, err)
			continue
		}

		tools[def.Name] = tool
	}

	return tools
}

// convertToolDefinition converts a single YAML tool definition to an mcp.Tool
func convertToolDefinition(def ToolDefinition) (mcp.Tool, error) {
	if def.Name == "" {
		return mcp.Tool{}, fmt.Errorf("tool name is required")
	}

	if def.Description == "" {
		return mcp.Tool{}, fmt.Errorf("tool description is required for tool '%s'", def.Name)
	}

	var zeroAnnotations Annotations
	if def.Annotations == zeroAnnotations {
		return mcp.Tool{}, fmt.Errorf("annotations block is required for tool '%s'", def.Name)
	}

	options := []mcp.ToolOption{
		mcp.WithDescription(def.Description),
	}

	for _, param := range def.Parameters {
		options = append(options, convertParameter(param))
	}

	options = append(options, convertAnnotation(def.Annotations))

	return mcp.NewTool(def.Name, options...), nil
}

// convertAnnotation converts a YAML annotation definition to an mcp option
func convertAnnotation(annotation Annotations) mcp.ToolOption {
	return mcp.WithToolAnnotation(mcp.ToolAnnotation{
		Title:           annotation.Title,
		ReadOnlyHint:    &annotation.ReadOnlyHint,
		DestructiveHint: &annotation.DestructiveHint,
		IdempotentHint:  &annotation.IdempotentHint,
		OpenWorldHint:   &annotation.OpenWorldHint,
	})
}

// convertParameter converts a YAML parameter definition to an mcp option
func convertParameter(param ParameterDefinition) mcp.ToolOption {
	var options []mcp.PropertyOption

	options = append(options, mcp.Description(param.Description))

	if param.Required {
		options = append(options, mcp.Required())
	}

	if param.Enum != nil {
		options = append(options, mcp.Enum(param.Enum...))
	}

	if len(param.Items) > 0 {
		options = append(options, mcp.Items(param.Items))
	}

	switch param.Type {
	case "string":
		return mcp.WithString(param.Name, options...)
	case "number":
		return mcp.WithNumber(param.Name, options...)
	case "boolean":
		return mcp.WithBoolean(param.Name, options...)
	case "array":
		return mcp.WithArray(param.Name, options...)
	case "object":
		return mcp.WithObject(param.Name, options...)
	default:
		// Default to string if type is unknown
		return mcp.WithString(param.Name, options...)
	}
}
