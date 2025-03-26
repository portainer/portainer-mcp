package mcp

import (
	"fmt"
)

// Schema interface for all schema types
type Schema interface {
	getOptions() []any
}

// StringSchema defines a string parameter schema
type StringSchema struct {
	Description string
	Required    bool
	Default     string
	Enum        []string
}

func (s StringSchema) getOptions() []any {
	var options []any

	if s.Description != "" {
		options = append(options, "description", s.Description)
	}

	if s.Required {
		options = append(options, "required", true)
	}

	if s.Default != "" {
		options = append(options, "default", s.Default)
	}

	if len(s.Enum) > 0 {
		options = append(options, "enum", s.Enum)
	}

	return options
}

// NewStringSchema creates a new string schema
func NewStringSchema(description string, required bool, options ...func(*StringSchema)) StringSchema {
	schema := StringSchema{
		Description: description,
		Required:    required,
	}

	for _, option := range options {
		option(&schema)
	}

	return schema
}

// WithEnum adds an enum to the string schema
func WithEnum(values ...string) func(*StringSchema) {
	return func(s *StringSchema) {
		s.Enum = values
	}
}

// WithDefault adds a default value to the string schema
func WithDefault(value string) func(*StringSchema) {
	return func(s *StringSchema) {
		s.Default = value
	}
}

// NumberSchema defines a number parameter schema
type NumberSchema struct {
	Description string
	Required    bool
	Minimum     *float64
	Maximum     *float64
	Default     *float64
}

func (s NumberSchema) getOptions() []any {
	var options []any

	if s.Description != "" {
		options = append(options, "description", s.Description)
	}

	if s.Required {
		options = append(options, "required", true)
	}

	if s.Minimum != nil {
		options = append(options, "minimum", *s.Minimum)
	}

	if s.Maximum != nil {
		options = append(options, "maximum", *s.Maximum)
	}

	if s.Default != nil {
		options = append(options, "default", *s.Default)
	}

	return options
}

// NewNumberSchema creates a new number schema
func NewNumberSchema(description string, required bool, options ...func(*NumberSchema)) NumberSchema {
	schema := NumberSchema{
		Description: description,
		Required:    required,
	}

	for _, option := range options {
		option(&schema)
	}

	return schema
}

// WithMinimum adds a minimum value to the number schema
func WithMinimum(value float64) func(*NumberSchema) {
	return func(s *NumberSchema) {
		s.Minimum = &value
	}
}

// WithMaximum adds a maximum value to the number schema
func WithMaximum(value float64) func(*NumberSchema) {
	return func(s *NumberSchema) {
		s.Maximum = &value
	}
}

// WithNumberDefault adds a default value to the number schema
func WithNumberDefault(value float64) func(*NumberSchema) {
	return func(s *NumberSchema) {
		s.Default = &value
	}
}

// ArraySchema defines an array parameter schema
type ArraySchema struct {
	Description string
	Required    bool
	Items       map[string]any
}

func (s ArraySchema) getOptions() []any {
	var options []any

	if s.Description != "" {
		options = append(options, "description", s.Description)
	}

	if s.Required {
		options = append(options, "required", true)
	}

	if s.Items != nil {
		options = append(options, "items", s.Items)
	}

	return options
}

// NewArraySchema creates a new array schema
func NewArraySchema(description string, required bool, items map[string]any) ArraySchema {
	return ArraySchema{
		Description: description,
		Required:    required,
		Items:       items,
	}
}

// NewObjectArraySchema creates a new array schema with object items
func NewObjectArraySchema(description string, required bool, properties map[string]map[string]any) ArraySchema {
	return ArraySchema{
		Description: description,
		Required:    required,
		Items: map[string]any{
			"type":       "object",
			"properties": properties,
		},
	}
}

// NewAccessArraySchema creates a new array schema for access objects
func NewAccessArraySchema(description string, entity string) ArraySchema {
	return NewObjectArraySchema(
		description,
		false,
		map[string]map[string]any{
			"id": {
				"type":        "number",
				"description": fmt.Sprintf("The ID of the %s", entity),
			},
			"access": {
				"type":        "string",
				"description": fmt.Sprintf("The access level of the %s. Can be %s", entity, 
					"environment_administrator, helpdesk_user, standard_user, readonly_user or operator_user"),
				"enum": AllAccessLevels,
			},
		},
	)
}

// ObjectSchema defines an object parameter schema
type ObjectSchema struct {
	Description string
	Required    bool
	Properties  map[string]any
}

func (s ObjectSchema) getOptions() []any {
	var options []any

	if s.Description != "" {
		options = append(options, "description", s.Description)
	}

	if s.Required {
		options = append(options, "required", true)
	}

	if s.Properties != nil {
		options = append(options, "properties", s.Properties)
	}

	return options
}

// NewObjectSchema creates a new object schema
func NewObjectSchema(description string, required bool, properties map[string]any) ObjectSchema {
	return ObjectSchema{
		Description: description,
		Required:    required,
		Properties:  properties,
	}
}

// BooleanSchema defines a boolean parameter schema
type BooleanSchema struct {
	Description string
	Required    bool
	Default     *bool
}

func (s BooleanSchema) getOptions() []any {
	var options []any

	if s.Description != "" {
		options = append(options, "description", s.Description)
	}

	if s.Required {
		options = append(options, "required", true)
	}

	if s.Default != nil {
		options = append(options, "default", *s.Default)
	}

	return options
}

// NewBooleanSchema creates a new boolean schema
func NewBooleanSchema(description string, required bool, defaultValue *bool) BooleanSchema {
	return BooleanSchema{
		Description: description,
		Required:    required,
		Default:     defaultValue,
	}
}

// IDSchema creates a standard ID number schema
func IDSchema(entity string) NumberSchema {
	return NewNumberSchema(
		fmt.Sprintf("The ID of the %s to operate on", entity),
		true,
		WithMinimum(1),
	)
}

// NameSchema creates a standard name string schema
func NameSchema(entity string) StringSchema {
	return NewStringSchema(
		fmt.Sprintf("The name of the %s", entity),
		true,
	)
}