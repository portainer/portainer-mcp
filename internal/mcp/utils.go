package mcp

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// parseNumericArray converts a slice of any type to a slice of ints.
// Returns an error if any value cannot be parsed as an integer.
//
// Example:
//
//	ids, err := parseNumericArray([]any{1, 2, 3})
//	// ids = []int{1, 2, 3}
func parseNumericArray(array []any) ([]int, error) {
	if array == nil {
		return []int{}, nil
	}

	result := make([]int, 0, len(array))

	for _, item := range array {
		idFloat, ok := item.(float64)
		if !ok {
			return nil, NewInvalidParameterError(
				fmt.Sprintf("failed to parse '%v' as integer", item),
				nil,
			)
		}
		result = append(result, int(idFloat))
	}

	return result, nil
}

// parseAccessMapUtil parses access entries from the request parameters and returns a map of ID to access level
func parseAccessMapUtil(entries []any) (map[int]string, error) {
	if entries == nil {
		return map[int]string{}, nil
	}

	accessMap := map[int]string{}

	for _, entry := range entries {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			return nil, NewInvalidParameterError(
				fmt.Sprintf("invalid access entry: %v", entry),
				nil,
			)
		}

		id, ok := entryMap["id"].(float64)
		if !ok {
			return nil, NewInvalidParameterError(
				fmt.Sprintf("invalid ID: %v", entryMap["id"]),
				nil,
			)
		}

		access, ok := entryMap["access"].(string)
		if !ok {
			return nil, NewInvalidParameterError(
				fmt.Sprintf("invalid access: %v", entryMap["access"]),
				nil,
			)
		}

		if !IsValidAccessLevel(access) {
			return nil, NewInvalidParameterError(
				fmt.Sprintf("invalid access level: %s", access),
				nil,
			)
		}

		accessMap[int(id)] = access
	}

	return accessMap, nil
}

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
			return "", NewInvalidParameterError(fmt.Sprintf("%s is required", name), nil)
		}
		return "", nil
	}

	strValue, ok := value.(string)
	if !ok {
		return "", NewInvalidParameterError(fmt.Sprintf("%s must be a string", name), nil)
	}

	return strValue, nil
}

// GetNumber extracts a number parameter from the request
func (p *ParameterParser) GetNumber(name string, required bool) (float64, error) {
	value, ok := p.args[name]
	if !ok || value == nil {
		if required {
			return 0, NewInvalidParameterError(fmt.Sprintf("%s is required", name), nil)
		}
		return 0, nil
	}

	numValue, ok := value.(float64)
	if !ok {
		return 0, NewInvalidParameterError(fmt.Sprintf("%s must be a number", name), nil)
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
			return false, NewInvalidParameterError(fmt.Sprintf("%s is required", name), nil)
		}
		return false, nil
	}

	boolValue, ok := value.(bool)
	if !ok {
		return false, NewInvalidParameterError(fmt.Sprintf("%s must be a boolean", name), nil)
	}

	return boolValue, nil
}

// GetNumericArray extracts an array of numbers parameter from the request
func (p *ParameterParser) GetNumericArray(name string, required bool) ([]int, error) {
	value, ok := p.args[name]
	if !ok || value == nil {
		if required {
			return nil, NewInvalidParameterError(fmt.Sprintf("%s is required", name), nil)
		}
		return []int{}, nil
	}

	arrayValue, ok := value.([]any)
	if !ok {
		return nil, NewInvalidParameterError(fmt.Sprintf("%s must be an array", name), nil)
	}

	return parseNumericArray(arrayValue)
}

// GetAccessMap extracts an access map parameter from the request
func (p *ParameterParser) GetAccessMap(name string, required bool) (map[int]string, error) {
	value, ok := p.args[name]
	if !ok || value == nil {
		if required {
			return nil, NewInvalidParameterError(fmt.Sprintf("%s is required", name), nil)
		}
		return map[int]string{}, nil
	}

	arrayValue, ok := value.([]any)
	if !ok {
		return nil, NewInvalidParameterError(fmt.Sprintf("%s must be an array", name), nil)
	}

	return parseAccessMapUtil(arrayValue)
}