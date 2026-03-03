package toolgen

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

// toFloat64 converts various numeric types to float64.
// MCP transports may deliver numbers as float64, int, or json.Number.
// LLMs sometimes send numeric strings (e.g. "90" instead of 90) so
// string values that parse as numbers are also accepted.
func toFloat64(value any, name string) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, fmt.Errorf("%s is not a valid number: %w", name, err)
		}
		return f, nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a number, got string %q", name, v)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("%s must be a number", name)
	}
}

// ParameterParser provides methods to safely extract parameters from request arguments
type ParameterParser struct {
	args map[string]any
}

// NewParameterParser creates a new parameter parser for the given request
func NewParameterParser(request mcp.CallToolRequest) *ParameterParser {
	return &ParameterParser{
		args: request.GetArguments(),
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

// GetNumber extracts a number parameter from the request.
// It handles float64 (standard JSON unmarshalling), int (some MCP transports),
// and json.Number types.
func (p *ParameterParser) GetNumber(name string, required bool) (float64, error) {
	value, ok := p.args[name]
	if !ok || value == nil {
		if required {
			return 0, fmt.Errorf("%s is required", name)
		}
		return 0, nil
	}

	return toFloat64(value, name)
}

// GetInt extracts an integer parameter from the request
func (p *ParameterParser) GetInt(name string, required bool) (int, error) {
	num, err := p.GetNumber(name, required)
	if err != nil {
		return 0, err
	}
	if num != float64(int(num)) {
		return 0, fmt.Errorf("%s must be a whole number, got %v", name, num)
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
		idFloat, err := toFloat64(item, "array element")
		if err != nil {
			return nil, fmt.Errorf("failed to parse '%v' as integer", item)
		}
		if idFloat != float64(int(idFloat)) {
			return nil, fmt.Errorf("value '%v' is not a whole number", idFloat)
		}
		result = append(result, int(idFloat))
	}

	return result, nil
}
