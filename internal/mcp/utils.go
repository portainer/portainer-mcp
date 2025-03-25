package mcp

import (
	"fmt"
)

// parseNumericArray converts a slice of any type to a slice of ints.
// Returns an error if any value cannot be parsed as an integer.
//
// Example:
//
//	ids, err := parseNumericArray([]any{1, 2, 3})
//	// ids = []int{1, 2, 3}
func parseNumericArray(array []any) ([]int, error) {
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
