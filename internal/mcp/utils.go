package mcp

import (
	"fmt"
	"strconv"
	"strings"
)

// parseCommaSeparatedInts converts a comma-separated string of integers into a slice of ints.
// Returns an error if any value cannot be parsed as an integer.
//
// Example:
//
//	ids, err := parseCommaSeparatedInts("1,2,3")
//	// ids = []int{1, 2, 3}
func parseCommaSeparatedInts(commaSeparatedStr string) ([]int, error) {
	if commaSeparatedStr == "" {
		return []int{}, nil
	}

	strValues := strings.Split(commaSeparatedStr, ",")
	result := make([]int, 0, len(strValues))

	for _, strVal := range strValues {
		val, err := strconv.Atoi(strVal)
		if err != nil {
			return nil, fmt.Errorf("failed to parse '%s' as integer: %w", strVal, err)
		}
		result = append(result, val)
	}

	return result, nil
}

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
