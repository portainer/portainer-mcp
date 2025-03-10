package mcp

import (
	"reflect"
	"testing"
)

func TestParseCommaSeparatedInts(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    []int{},
			wantErr: false,
		},
		{
			name:    "single value",
			input:   "42",
			want:    []int{42},
			wantErr: false,
		},
		{
			name:    "multiple values",
			input:   "1,2,3,4,5",
			want:    []int{1, 2, 3, 4, 5},
			wantErr: false,
		},
		{
			name:    "negative values",
			input:   "-1,-2,-3",
			want:    []int{-1, -2, -3},
			wantErr: false,
		},
		{
			name:    "mixed values",
			input:   "0,1,-2,3,-4",
			want:    []int{0, 1, -2, 3, -4},
			wantErr: false,
		},
		{
			name:    "invalid value",
			input:   "1,2,abc,3",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid value with spaces",
			input:   "1, 2, 3",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "floating point value",
			input:   "1,2.5,3",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCommaSeparatedInts(tt.input)

			// Check error status
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommaSeparatedInts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expect an error, no need to check the result
			if tt.wantErr {
				return
			}

			// Check result values
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCommaSeparatedInts() = %v, want %v", got, tt.want)
			}
		})
	}
}
