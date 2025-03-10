package mcp

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseCommaSeparatedInts converts a comma-separated string of integers into a slice of ints.
// Returns an error if any value cannot be parsed as an integer.
//
// Example:
//
//	ids, err := ParseCommaSeparatedInts("1,2,3")
//	// ids = []int{1, 2, 3}
func ParseCommaSeparatedInts(commaSeparatedStr string) ([]int, error) {
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
