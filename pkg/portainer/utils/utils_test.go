package utils

import (
	"reflect"
	"testing"
)

func TestInt64ToIntSlice(t *testing.T) {
	tests := []struct {
		name   string
		int64s []int64
		want   []int
	}{
		{
			name:   "empty slice",
			int64s: []int64{},
			want:   []int{},
		},
		{
			name:   "single element",
			int64s: []int64{42},
			want:   []int{42},
		},
		{
			name:   "multiple elements",
			int64s: []int64{1, 2, 3, 4, 5},
			want:   []int{1, 2, 3, 4, 5},
		},
		{
			name:   "large numbers",
			int64s: []int64{1000000000, 2000000000},
			want:   []int{1000000000, 2000000000},
		},
		{
			name:   "negative numbers",
			int64s: []int64{-1, -10, -100},
			want:   []int{-1, -10, -100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Int64ToIntSlice(tt.int64s)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Int64ToIntSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntToInt64Slice(t *testing.T) {
	tests := []struct {
		name string
		ints []int
		want []int64
	}{
		{
			name: "empty slice",
			ints: []int{},
			want: []int64{},
		},
		{
			name: "single element",
			ints: []int{42},
			want: []int64{42},
		},
		{
			name: "multiple elements",
			ints: []int{1, 2, 3, 4, 5},
			want: []int64{1, 2, 3, 4, 5},
		},
		{
			name: "large numbers",
			ints: []int{1000000000, 2000000000},
			want: []int64{1000000000, 2000000000},
		},
		{
			name: "negative numbers",
			ints: []int{-1, -10, -100},
			want: []int64{-1, -10, -100},
		},
		{
			name: "max int32 value",
			ints: []int{2147483647},
			want: []int64{2147483647},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntToInt64Slice(tt.ints)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IntToInt64Slice() = %v, want %v", got, tt.want)
			}
		})
	}
}
