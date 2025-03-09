package utils

func Int64ToIntSlice(int64s []int64) []int {
	ints := make([]int, len(int64s))
	for i, int64 := range int64s {
		ints[i] = int(int64)
	}
	return ints
}

func IntToInt64Slice(ints []int) []int64 {
	int64s := make([]int64, len(ints))
	for i, int := range ints {
		int64s[i] = int64(int)
	}
	return int64s
}
