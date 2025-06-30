package mcp

import (
	"fmt"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
)

// parseAccessMap parses access entries from an array of objects and returns a map of ID to access level
func parseAccessMap(entries []any) (map[int]string, error) {
	accessMap := map[int]string{}

	for _, entry := range entries {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid access entry: %v", entry)
		}

		id, ok := entryMap["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid ID: %v", entryMap["id"])
		}

		access, ok := entryMap["access"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid access: %v", entryMap["access"])
		}

		if !isValidAccessLevel(access) {
			return nil, fmt.Errorf("invalid access level: %s", access)
		}

		accessMap[int(id)] = access
	}

	return accessMap, nil
}

// parseKeyValueMap parses a slice of map[string]any into a map[string]string,
// expecting each map to have "key" and "value" string fields.
func parseKeyValueMap(items []any) (map[string]string, error) {
	resultMap := map[string]string{}

	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid item: %v", item)
		}

		key, ok := itemMap["key"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid key: %v", itemMap["key"])
		}

		value, ok := itemMap["value"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid value: %v", itemMap["value"])
		}

		resultMap[key] = value
	}

	return resultMap, nil
}

func isValidHTTPMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD"}
	return slices.Contains(validMethods, method)
}

// CreateMCPRequest creates a new MCP tool request with the given arguments
func CreateMCPRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}
