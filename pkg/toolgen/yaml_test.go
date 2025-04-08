package toolgen

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestLoadToolsFromYAML(t *testing.T) {
	// Create a minimal test YAML file
	tmpDir := t.TempDir()
	validYamlPath := filepath.Join(tmpDir, "valid.yaml")
	validYamlContent := `version: "v1.0.0"
tools:
  - name: testTool
    description: A test tool
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter`

	err := os.WriteFile(validYamlPath, []byte(validYamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	// Create a newer version YAML file
	newerVersionPath := filepath.Join(tmpDir, "newer.yaml")
	newerVersionContent := `version: "v1.1.0"
tools:
  - name: testTool
    description: A test tool
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter`

	err = os.WriteFile(newerVersionPath, []byte(newerVersionContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create newer version YAML file: %v", err)
	}

	// Create an older version YAML file
	olderVersionPath := filepath.Join(tmpDir, "older.yaml")
	olderVersionContent := `version: "v0.9.0"
tools:
  - name: testTool
    description: A test tool
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter`

	err = os.WriteFile(olderVersionPath, []byte(olderVersionContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create older version YAML file: %v", err)
	}

	// Create a file with missing version
	missingVersionPath := filepath.Join(tmpDir, "missing_version.yaml")
	missingVersionContent := `tools:
  - name: testTool
    description: A test tool
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter`

	err = os.WriteFile(missingVersionPath, []byte(missingVersionContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create missing version YAML file: %v", err)
	}

	// Create a file with invalid version format
	invalidVersionPath := filepath.Join(tmpDir, "invalid_version.yaml")
	invalidVersionContent := `version: "1.0"
tools:
  - name: testTool
    description: A test tool
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter`

	err = os.WriteFile(invalidVersionPath, []byte(invalidVersionContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid version YAML file: %v", err)
	}

	tests := []struct {
		name           string
		filePath       string
		minimumVersion string
		wantErr        bool
		wantTool       string // name of tool we expect to find
	}{
		{
			name:           "valid yaml file",
			filePath:       validYamlPath,
			minimumVersion: "v1.0.0",
			wantErr:        false,
			wantTool:       "testTool",
		},
		{
			name:           "valid yaml file with newer minimum version",
			filePath:       validYamlPath,
			minimumVersion: "v1.1.0",
			wantErr:        true, // Error because file version is below minimum
		},
		{
			name:           "newer version yaml file",
			filePath:       newerVersionPath,
			minimumVersion: "v1.0.0",
			wantErr:        false,
			wantTool:       "testTool",
		},
		{
			name:           "older version yaml file",
			filePath:       olderVersionPath,
			minimumVersion: "v1.0.0",
			wantErr:        true, // Error because file version is below minimum
		},
		{
			name:           "missing version in yaml",
			filePath:       missingVersionPath,
			minimumVersion: "v1.0.0",
			wantErr:        true,
		},
		{
			name:           "invalid version format",
			filePath:       invalidVersionPath,
			minimumVersion: "v1.0.0",
			wantErr:        true, // Error because version format is invalid
		},
		{
			name:           "non-existent file",
			filePath:       "nonexistent.yaml",
			minimumVersion: "v1.0.0",
			wantErr:        true,
		},
		{
			name:           "invalid yaml content",
			filePath:       createInvalidYAMLFile(t),
			minimumVersion: "v1.0.0",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools, err := LoadToolsFromYAML(tt.filePath, tt.minimumVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadToolsFromYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantTool != "" {
				tool, exists := tools[tt.wantTool]
				if !exists {
					t.Errorf("Expected tool %s not found", tt.wantTool)
					return
				}
				if tool.Name != tt.wantTool {
					t.Errorf("Tool name mismatch, got %s, want %s", tool.Name, tt.wantTool)
				}
				if tool.Description == "" {
					t.Errorf("Tool %s has no description", tt.wantTool)
				}
			}
		})
	}
}

// Helper function to create an invalid YAML file for testing
func createInvalidYAMLFile(t *testing.T) string {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.yaml")
	content := `version: "v1.0.0"
tools:
  - name: invalid
    description: [invalid yaml content`

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid YAML file: %v", err)
	}
	return path
}

func TestConvertToolDefinition(t *testing.T) {
	tests := []struct {
		name    string
		def     ToolDefinition
		wantErr bool
	}{
		{
			name: "valid tool definition",
			def: ToolDefinition{
				Name:        "validTool",
				Description: "A valid tool description",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			def: ToolDefinition{
				Name:        "",
				Description: "A tool with empty name",
			},
			wantErr: true,
		},
		{
			name: "empty description",
			def: ToolDefinition{
				Name:        "noDescTool",
				Description: "",
			},
			wantErr: true,
		},
		{
			name: "with parameters",
			def: ToolDefinition{
				Name:        "paramTool",
				Description: "Tool with parameters",
				Parameters: []ParameterDefinition{
					{
						Name:        "param1",
						Type:        "string",
						Required:    true,
						Description: "A test parameter",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := convertToolDefinition(tt.def)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToolDefinition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Test that the tool was created correctly
				tool, err := convertToolDefinition(tt.def)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}

				if tool.Name != tt.def.Name {
					t.Errorf("Tool name mismatch, got %s, want %s", tool.Name, tt.def.Name)
				}

				if tool.Description != tt.def.Description {
					t.Errorf("Tool description mismatch, got %s, want %s", tool.Description, tt.def.Description)
				}
			}
		})
	}
}

func TestConvertToolDefinitions(t *testing.T) {
	tests := []struct {
		name string
		defs []ToolDefinition
		want int // number of tools expected
	}{
		{
			name: "empty definitions",
			defs: []ToolDefinition{},
			want: 0,
		},
		{
			name: "single tool",
			defs: []ToolDefinition{
				{
					Name:        "tool1",
					Description: "Test tool 1",
					Parameters: []ParameterDefinition{
						{
							Name:        "param1",
							Type:        "string",
							Required:    true,
							Description: "Test parameter",
						},
					},
				},
			},
			want: 1,
		},
		{
			name: "multiple tools",
			defs: []ToolDefinition{
				{
					Name:        "tool1",
					Description: "Test tool 1",
				},
				{
					Name:        "tool2",
					Description: "Test tool 2",
				},
			},
			want: 2,
		},
		{
			name: "invalid tools are skipped",
			defs: []ToolDefinition{
				{
					Name:        "tool1",
					Description: "Test tool 1",
				},
				{
					Name:        "", // Invalid: empty name
					Description: "Tool with empty name",
				},
				{
					Name:        "noDescTool", // Invalid: empty description
					Description: "",
				},
				{
					Name:        "tool2",
					Description: "Test tool 2",
				},
			},
			want: 2, // Only 2 valid tools should be returned
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToolDefinitions(tt.defs)
			if len(got) != tt.want {
				t.Errorf("ConvertToolDefinitions() returned %v tools, want %v", len(got), tt.want)
			}

			// Verify each tool has the correct name and description
			for _, def := range tt.defs {
				// Skip invalid tools with empty name or description
				if def.Name == "" || def.Description == "" {
					continue
				}

				tool, exists := got[def.Name]
				if !exists {
					t.Errorf("Tool %s not found in result", def.Name)
					continue
				}

				if tool.Name != def.Name {
					t.Errorf("Tool name mismatch, got %s, want %s", tool.Name, def.Name)
				}

				if tool.Description != def.Description {
					t.Errorf("Tool description mismatch for %s, got %s, want %s",
						def.Name, tool.Description, def.Description)
				}
			}
		})
	}
}

func TestConvertParameter(t *testing.T) {
	tests := []struct {
		name  string
		param ParameterDefinition
		want  reflect.Type // We'll check the type of the returned option
	}{
		{
			name: "string parameter",
			param: ParameterDefinition{
				Name:        "strParam",
				Type:        "string",
				Required:    true,
				Description: "A string parameter",
			},
			want: reflect.TypeOf(mcp.WithString("", mcp.Description(""))),
		},
		{
			name: "number parameter",
			param: ParameterDefinition{
				Name:        "numParam",
				Type:        "number",
				Required:    true,
				Description: "A number parameter",
			},
			want: reflect.TypeOf(mcp.WithNumber("", mcp.Description(""))),
		},
		{
			name: "boolean parameter",
			param: ParameterDefinition{
				Name:        "boolParam",
				Type:        "boolean",
				Required:    true,
				Description: "A boolean parameter",
			},
			want: reflect.TypeOf(mcp.WithBoolean("", mcp.Description(""))),
		},
		{
			name: "array parameter",
			param: ParameterDefinition{
				Name:        "arrayParam",
				Type:        "array",
				Required:    true,
				Description: "An array parameter",
				Items: map[string]any{
					"type": "string",
				},
			},
			want: reflect.TypeOf(mcp.WithArray("", mcp.Description(""))),
		},
		{
			name: "object parameter",
			param: ParameterDefinition{
				Name:        "objParam",
				Type:        "object",
				Required:    true,
				Description: "An object parameter",
				Items: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"key": map[string]any{
							"type": "string",
						},
					},
				},
			},
			want: reflect.TypeOf(mcp.WithObject("", mcp.Description(""))),
		},
		{
			name: "enum parameter",
			param: ParameterDefinition{
				Name:        "enumParam",
				Type:        "string",
				Required:    true,
				Description: "An enum parameter",
				Enum:        []string{"val1", "val2"},
			},
			want: reflect.TypeOf(mcp.WithString("", mcp.Description(""))),
		},
		{
			name: "unknown type parameter",
			param: ParameterDefinition{
				Name:        "unknownParam",
				Type:        "unknown",
				Required:    true,
				Description: "An unknown parameter",
			},
			want: reflect.TypeOf(mcp.WithString("", mcp.Description(""))), // defaults to string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertParameter(tt.param)
			gotType := reflect.TypeOf(got)
			if gotType != tt.want {
				t.Errorf("convertParameter() returned %v, want %v", gotType, tt.want)
			}
		})
	}
}
