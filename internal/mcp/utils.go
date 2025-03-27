package mcp

import "fmt"

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
