package utils

// Int64ToIntSlice converts a slice of int64 values to a slice of int values.
// This may result in data loss if the int64 values exceed the range of int.
func Int64ToIntSlice(int64s []int64) []int {
	ints := make([]int, len(int64s))
	for i, int64 := range int64s {
		ints[i] = int(int64)
	}
	return ints
}

// IntToInt64Slice converts a slice of int values to a slice of int64 values.
func IntToInt64Slice(ints []int) []int64 {
	int64s := make([]int64, len(ints))
	for i, int := range ints {
		int64s[i] = int64(int)
	}
	return int64s
}
