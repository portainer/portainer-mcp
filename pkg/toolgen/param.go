package toolgen

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ParameterParser provides methods to safely extract parameters from request arguments
type ParameterParser struct {
	args map[string]any
}

// NewParameterParser creates a new parameter parser for the given request
func NewParameterParser(request mcp.CallToolRequest) *ParameterParser {
	return &ParameterParser{
		args: request.Params.Arguments,
	}
}

// GetString extracts a string parameter from the request
func (p *ParameterParser) GetString(name string, required bool) (string, error) {
	value, ok := p.args[name]
	if !ok || value == nil {
		if required {
			return "", fmt.Errorf("%s is required", name)
		}
		return "", nil
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", name)
	}

	return strValue, nil
}

// GetNumber extracts a number parameter from the request
func (p *ParameterParser) GetNumber(name string, required bool) (float64, error) {
	value, ok := p.args[name]
	if !ok || value == nil {
		if required {
			return 0, fmt.Errorf("%s is required", name)
		}
		return 0, nil
	}

	numValue, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("%s must be a number", name)
	}

	return numValue, nil
}

// GetInt extracts an integer parameter from the request
func (p *ParameterParser) GetInt(name string, required bool) (int, error) {
	num, err := p.GetNumber(name, required)
	if err != nil {
		return 0, err
	}
	return int(num), nil
}

// GetBoolean extracts a boolean parameter from the request
func (p *ParameterParser) GetBoolean(name string, required bool) (bool, error) {
	value, ok := p.args[name]
	if !ok || value == nil {
		if required {
			return false, fmt.Errorf("%s is required", name)
		}
		return false, nil
	}

	boolValue, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean", name)
	}

	return boolValue, nil
}

// GetArrayOfIntegers extracts an array of numbers parameter from the request
func (p *ParameterParser) GetArrayOfIntegers(name string, required bool) ([]int, error) {
	value, ok := p.args[name]
	if !ok || value == nil {
		if required {
			return nil, fmt.Errorf("%s is required", name)
		}
		return []int{}, nil
	}

	arrayValue, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", name)
	}

	return parseArrayOfIntegers(arrayValue)
}

// GetArrayOfObjects extracts an array of objects parameter from the request
func (p *ParameterParser) GetArrayOfObjects(name string, required bool) ([]any, error) {
	value, ok := p.args[name]
	if !ok || value == nil {
		if required {
			return nil, fmt.Errorf("%s is required", name)
		}
		return []any{}, nil
	}

	arrayValue, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", name)
	}

	return arrayValue, nil
}

// parseArrayOfIntegers converts a slice of any type to a slice of integers.
// Returns an error if any value cannot be parsed as an integer.
//
// Example:
//
//	ids, err := parseArrayOfIntegers([]any{1, 2, 3})
//	// ids = []int{1, 2, 3}
func parseArrayOfIntegers(array []any) ([]int, error) {
	result := make([]int, 0, len(array))

	for _, item := range array {
		idFloat, ok := item.(float64)
		if !ok {
			return nil, fmt.Errorf("failed to parse '%v' as integer", item)
		}
		result = append(result, int(idFloat))
	}

	return result, nil
}
