package toolgen

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
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
        description: A test parameter
    annotations:
      title: Test Tool Title
      readOnlyHint: true
      destructiveHint: false
      idempotentHint: true
      openWorldHint: false`

	err := os.WriteFile(validYamlPath, []byte(validYamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	// Create a newer version YAML file
	newerVersionPath := filepath.Join(tmpDir, "newer.yaml")
	newerVersionContent := `version: "v1.2.0"
tools:
  - name: testTool
    description: A test tool
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter
    annotations:
      title: Test Tool Title
      readOnlyHint: true
      destructiveHint: false
      idempotentHint: true
      openWorldHint: false`

	err = os.WriteFile(newerVersionPath, []byte(newerVersionContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create newer version YAML file: %v", err)
	}

	// Create an older version YAML file (will fail version check)
	olderVersionPath := filepath.Join(tmpDir, "older.yaml")
	olderVersionContent := `version: "v0.9.0"
tools:
  - name: testTool
    description: A test tool
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter
    # Annotations potentially missing, but version check fails first
`

	err = os.WriteFile(olderVersionPath, []byte(olderVersionContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create older version YAML file: %v", err)
	}

	// Create a file with missing version (will fail version check)
	missingVersionPath := filepath.Join(tmpDir, "missing_version.yaml")
	missingVersionContent := `tools:
  - name: testTool
    description: A test tool
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter
    # Annotations potentially missing, but version check fails first
`

	err = os.WriteFile(missingVersionPath, []byte(missingVersionContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create missing version YAML file: %v", err)
	}

	// Create a file with invalid version format (will fail version check)
	invalidVersionPath := filepath.Join(tmpDir, "invalid_version.yaml")
	invalidVersionContent := `version: "1.0"
tools:
  - name: testTool
    description: A test tool
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter
    # Annotations potentially missing, but version check fails first
`

	err = os.WriteFile(invalidVersionPath, []byte(invalidVersionContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid version YAML file: %v", err)
	}

	// Create a file with missing annotations block (should fail annotation check)
	missingAnnotationsPath := filepath.Join(tmpDir, "missing_annotations.yaml")
	missingAnnotationsContent := `version: "v1.0.0"
tools:
  - name: toolWithoutAnnotations
    description: A test tool missing annotations
    parameters:
      - name: param1
        type: string
        required: true
        description: A test parameter
  - name: toolWithAnnotations
    description: A test tool with annotations
    annotations:
      title: Some Title
      readOnlyHint: false
      destructiveHint: false
      idempotentHint: false
      openWorldHint: false
`
	err = os.WriteFile(missingAnnotationsPath, []byte(missingAnnotationsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create missing annotations YAML file: %v", err)
	}

	tests := []struct {
		name           string
		filePath       string
		minimumVersion string
		wantErr        bool
		wantTool       string // name of tool we expect to find
		wantToolCount  int    // expected number of tools loaded
	}{
		{
			name:           "valid yaml file",
			filePath:       validYamlPath,
			minimumVersion: "v1.0.0",
			wantErr:        false,
			wantTool:       "testTool",
			wantToolCount:  1,
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
			wantToolCount:  1,
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
			name:           "missing annotations block",
			filePath:       missingAnnotationsPath,
			minimumVersion: "v1.0.0",
			wantErr:        false,                 // LoadToolsFromYAML itself doesn't error, but skips the invalid tool
			wantTool:       "toolWithAnnotations", // Only the tool with annotations should load
			wantToolCount:  1,                     // Expect only one tool to be loaded successfully
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

			if !tt.wantErr {
				if len(tools) != tt.wantToolCount {
					t.Errorf("LoadToolsFromYAML() loaded %d tools, want %d", len(tools), tt.wantToolCount)
				}
				if tt.wantTool != "" {
					tool, exists := tools[tt.wantTool]
					if !exists {
						t.Errorf("Expected tool '%s' not found in loaded tools: %v", tt.wantTool, tools)
						return
					}
					if tool.Name != tt.wantTool {
						t.Errorf("Tool name mismatch, got %s, want %s", tool.Name, tt.wantTool)
					}
					if tool.Description == "" {
						t.Errorf("Tool %s has no description", tt.wantTool)
					}
					// Basic check to ensure annotations were processed (more detailed checks in TestConvertToolDefinition)
					if tool.Annotations.Title == "" { // Check a field within Annotations
						t.Errorf("Tool %s seems to be missing processed annotations", tt.wantTool)
					}
				}
			}
		})
	}
}

// Helper function to create an invalid YAML file for testing
func createInvalidYAMLFile(t *testing.T) string {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.yaml")
	// Add annotations to avoid failing that check first
	content := `version: "v1.0.0"
tools:
  - name: invalid
    description: [invalid yaml content
    annotations:
      title: Invalid Tool`

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid YAML file: %v", err)
	}
	return path
}

func TestConvertToolDefinition(t *testing.T) {
	// Define a valid annotation struct to reuse
	validAnnotations := Annotations{
		Title:           "Valid Title",
		ReadOnlyHint:    true,
		DestructiveHint: false,
		IdempotentHint:  true,
		OpenWorldHint:   false,
	}

	tests := []struct {
		name          string
		def           ToolDefinition
		wantErr       bool
		wantErrSubstr string              // Optional: check for specific error message content
		want          *mcp.ToolAnnotation // Expected annotation output
	}{
		{
			name: "valid tool definition",
			def: ToolDefinition{
				Name:        "validTool",
				Description: "A valid tool description",
				Annotations: validAnnotations,
			},
			wantErr: false,
			want: &mcp.ToolAnnotation{
				Title:           "Valid Title",
				ReadOnlyHint:    &validAnnotations.ReadOnlyHint,
				DestructiveHint: &validAnnotations.DestructiveHint,
				IdempotentHint:  &validAnnotations.IdempotentHint,
				OpenWorldHint:   &validAnnotations.OpenWorldHint,
			},
		},
		{
			name: "empty name",
			def: ToolDefinition{
				Name:        "",
				Description: "A tool with empty name",
				Annotations: validAnnotations, // Needs annotations even if name is invalid
			},
			wantErr:       true,
			wantErrSubstr: "tool name is required",
		},
		{
			name: "empty description",
			def: ToolDefinition{
				Name:        "noDescTool",
				Description: "",
				Annotations: validAnnotations, // Needs annotations even if desc is invalid
			},
			wantErr:       true,
			wantErrSubstr: "tool description is required",
		},
		{
			name: "missing annotations",
			def: ToolDefinition{
				Name:        "noAnnotationTool",
				Description: "Tool without annotations",
				Annotations: Annotations{}, // Zero value simulates missing block
			},
			wantErr:       true,
			wantErrSubstr: "annotations block is required",
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
				Annotations: validAnnotations,
			},
			wantErr: false,
			want: &mcp.ToolAnnotation{
				Title:           "Valid Title",
				ReadOnlyHint:    &validAnnotations.ReadOnlyHint,
				DestructiveHint: &validAnnotations.DestructiveHint,
				IdempotentHint:  &validAnnotations.IdempotentHint,
				OpenWorldHint:   &validAnnotations.OpenWorldHint,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := convertToolDefinition(tt.def)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrSubstr != "" {
					assert.Contains(t, err.Error(), tt.wantErrSubstr)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.def.Name, tool.Name)
				assert.Equal(t, tt.def.Description, tool.Description)
				assert.Equal(t, *tt.want, tool.Annotations)

			}
		})
	}
}

func TestConvertToolDefinitions(t *testing.T) {
	// Define a valid annotation struct to reuse
	validAnnotations := Annotations{
		Title:           "Valid Title",
		ReadOnlyHint:    true,
		DestructiveHint: false,
		IdempotentHint:  true,
		OpenWorldHint:   false,
	}

	tests := []struct {
		name string
		defs []ToolDefinition
		want int // number of tools expected to be successfully converted
	}{
		{
			name: "empty definitions",
			defs: []ToolDefinition{},
			want: 0,
		},
		{
			name: "single valid tool",
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
					Annotations: validAnnotations,
				},
			},
			want: 1,
		},
		{
			name: "multiple valid tools",
			defs: []ToolDefinition{
				{
					Name:        "tool1",
					Description: "Test tool 1",
					Annotations: validAnnotations,
				},
				{
					Name:        "tool2",
					Description: "Test tool 2",
					Annotations: validAnnotations,
				},
			},
			want: 2,
		},
		{
			name: "invalid tools are skipped",
			defs: []ToolDefinition{
				{
					Name:        "validTool1",
					Description: "Test tool 1",
					Annotations: validAnnotations,
				},
				{
					Name:        "", // Invalid: empty name
					Description: "Tool with empty name",
					Annotations: validAnnotations,
				},
				{
					Name:        "noDescTool", // Invalid: empty description
					Description: "",
					Annotations: validAnnotations,
				},
				{
					Name:        "noAnnotationTool", // Invalid: missing annotations
					Description: "Tool missing annotations",
					Annotations: Annotations{}, // Zero value
				},
				{
					Name:        "validTool2",
					Description: "Test tool 2",
					Annotations: validAnnotations,
				},
			},
			want: 2, // Only 2 valid tools should be returned
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToolDefinitions(tt.defs)
			assert.Len(t, got, tt.want)

			// Verify each tool expected to be converted exists and is valid
			for _, def := range tt.defs {
				// Skip definitions that are expected to cause errors
				if def.Name == "" || def.Description == "" || (def.Annotations == Annotations{}) {
					continue
				}

				tool, exists := got[def.Name]
				assert.True(t, exists, "Tool %s not found in result", def.Name)
				if exists {
					assert.Equal(t, def.Name, tool.Name)
					assert.Equal(t, def.Description, tool.Description)
					assert.NotEmpty(t, tool.Annotations.Title) // Basic check that title is populated
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

// Optional: Add a specific test for convertAnnotation if desired, though it's simple
func TestConvertAnnotation(t *testing.T) {
	input := Annotations{
		Title:           "Test Title",
		ReadOnlyHint:    true,
		DestructiveHint: true,
		IdempotentHint:  false,
		OpenWorldHint:   false,
	}
	want := mcp.ToolAnnotation{
		Title:           "Test Title",
		ReadOnlyHint:    &input.ReadOnlyHint,
		DestructiveHint: &input.DestructiveHint,
		IdempotentHint:  &input.IdempotentHint,
		OpenWorldHint:   &input.OpenWorldHint,
	}

	dummyTool := &mcp.Tool{}
	option := convertAnnotation(input)
	option(dummyTool)

	assert.NotNil(t, option)
	assert.Equal(t, want, dummyTool.Annotations)
}
