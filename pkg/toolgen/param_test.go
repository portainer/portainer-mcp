package toolgen

import (
	"reflect"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// Helper function to create a ParameterParser with given arguments
func newTestParser(args map[string]any) *ParameterParser {
	return NewParameterParser(mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: args,
		},
	})
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		param    string
		required bool
		want     string
		wantErr  bool
	}{
		{
			name:     "valid string",
			args:     map[string]any{"name": "test"},
			param:    "name",
			required: true,
			want:     "test",
			wantErr:  false,
		},
		{
			name:     "missing required param",
			args:     map[string]any{},
			param:    "name",
			required: true,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "missing optional param",
			args:     map[string]any{},
			param:    "name",
			required: false,
			want:     "",
			wantErr:  false,
		},
		{
			name:     "wrong type",
			args:     map[string]any{"name": 123},
			param:    "name",
			required: true,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "nil value",
			args:     map[string]any{"name": nil},
			param:    "name",
			required: true,
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(tt.args)
			got, err := p.GetString(tt.param, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNumber(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		param    string
		required bool
		want     float64
		wantErr  bool
	}{
		{
			name:     "valid number",
			args:     map[string]any{"num": float64(42)},
			param:    "num",
			required: true,
			want:     42,
			wantErr:  false,
		},
		{
			name:     "missing required param",
			args:     map[string]any{},
			param:    "num",
			required: true,
			want:     0,
			wantErr:  true,
		},
		{
			name:     "missing optional param",
			args:     map[string]any{},
			param:    "num",
			required: false,
			want:     0,
			wantErr:  false,
		},
		{
			name:     "wrong type",
			args:     map[string]any{"num": "123"},
			param:    "num",
			required: true,
			want:     0,
			wantErr:  true,
		},
		{
			name:     "nil value",
			args:     map[string]any{"num": nil},
			param:    "num",
			required: true,
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(tt.args)
			got, err := p.GetNumber(tt.param, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBoolean(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		param    string
		required bool
		want     bool
		wantErr  bool
	}{
		{
			name:     "valid true",
			args:     map[string]any{"flag": true},
			param:    "flag",
			required: true,
			want:     true,
			wantErr:  false,
		},
		{
			name:     "valid false",
			args:     map[string]any{"flag": false},
			param:    "flag",
			required: true,
			want:     false,
			wantErr:  false,
		},
		{
			name:     "missing required param",
			args:     map[string]any{},
			param:    "flag",
			required: true,
			want:     false,
			wantErr:  true,
		},
		{
			name:     "missing optional param",
			args:     map[string]any{},
			param:    "flag",
			required: false,
			want:     false,
			wantErr:  false,
		},
		{
			name:     "wrong type",
			args:     map[string]any{"flag": "true"},
			param:    "flag",
			required: true,
			want:     false,
			wantErr:  true,
		},
		{
			name:     "nil value",
			args:     map[string]any{"flag": nil},
			param:    "flag",
			required: true,
			want:     false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(tt.args)
			got, err := p.GetBoolean(tt.param, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBoolean() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBoolean() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetArrayOfObjects(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		param    string
		required bool
		want     []any
		wantErr  bool
	}{
		{
			name: "valid array of objects",
			args: map[string]any{"objects": []any{
				map[string]any{"id": 1},
				map[string]any{"id": 2},
			}},
			param:    "objects",
			required: true,
			want: []any{
				map[string]any{"id": 1},
				map[string]any{"id": 2},
			},
			wantErr: false,
		},
		{
			name:     "missing required param",
			args:     map[string]any{},
			param:    "objects",
			required: true,
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "missing optional param",
			args:     map[string]any{},
			param:    "objects",
			required: false,
			want:     []any{},
			wantErr:  false,
		},
		{
			name:     "wrong type",
			args:     map[string]any{"objects": "not an array"},
			param:    "objects",
			required: true,
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "nil value",
			args:     map[string]any{"objects": nil},
			param:    "objects",
			required: true,
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(tt.args)
			got, err := p.GetArrayOfObjects(tt.param, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetArrayOfObjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetArrayOfObjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseArrayOfIntegers(t *testing.T) {
	tests := []struct {
		name    string
		input   []any
		want    []int
		wantErr bool
	}{
		{
			name:    "empty array",
			input:   []any{},
			want:    []int{},
			wantErr: false,
		},
		{
			name:    "single value",
			input:   []any{float64(42)},
			want:    []int{42},
			wantErr: false,
		},
		{
			name:    "multiple values",
			input:   []any{float64(1), float64(2), float64(3), float64(4), float64(5)},
			want:    []int{1, 2, 3, 4, 5},
			wantErr: false,
		},
		{
			name:    "negative values",
			input:   []any{float64(-1), float64(-2), float64(-3)},
			want:    []int{-1, -2, -3},
			wantErr: false,
		},
		{
			name:    "mixed positive and negative values",
			input:   []any{float64(0), float64(1), float64(-2), float64(3), float64(-4)},
			want:    []int{0, 1, -2, 3, -4},
			wantErr: false,
		},
		{
			name:    "invalid string value",
			input:   []any{float64(1), "abc", float64(3)},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid boolean value",
			input:   []any{float64(1), true, float64(3)},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid nil value",
			input:   []any{float64(1), nil, float64(3)},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArrayOfIntegers(tt.input)

			// Check error status
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNumericArray() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expect an error, no need to check the result
			if tt.wantErr {
				return
			}

			// Check result values
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseNumericArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		param    string
		required bool
		want     int
		wantErr  bool
	}{
		{
			name:     "valid integer",
			args:     map[string]any{"num": float64(42)},
			param:    "num",
			required: true,
			want:     42,
			wantErr:  false,
		},
		{
			name:     "valid zero",
			args:     map[string]any{"num": float64(0)},
			param:    "num",
			required: true,
			want:     0,
			wantErr:  false,
		},
		{
			name:     "valid negative",
			args:     map[string]any{"num": float64(-42)},
			param:    "num",
			required: true,
			want:     -42,
			wantErr:  false,
		},
		{
			name:     "missing required param",
			args:     map[string]any{},
			param:    "num",
			required: true,
			want:     0,
			wantErr:  true,
		},
		{
			name:     "missing optional param",
			args:     map[string]any{},
			param:    "num",
			required: false,
			want:     0,
			wantErr:  false,
		},
		{
			name:     "wrong type string",
			args:     map[string]any{"num": "123"},
			param:    "num",
			required: true,
			want:     0,
			wantErr:  true,
		},
		{
			name:     "wrong type boolean",
			args:     map[string]any{"num": true},
			param:    "num",
			required: true,
			want:     0,
			wantErr:  true,
		},
		{
			name:     "nil value",
			args:     map[string]any{"num": nil},
			param:    "num",
			required: true,
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(tt.args)
			got, err := p.GetInt(tt.param, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetArrayOfIntegers(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		param    string
		required bool
		want     []int
		wantErr  bool
	}{
		{
			name: "valid array of integers",
			args: map[string]any{"nums": []any{
				float64(1), float64(2), float64(3),
			}},
			param:    "nums",
			required: true,
			want:     []int{1, 2, 3},
			wantErr:  false,
		},
		{
			name: "valid array with negative numbers",
			args: map[string]any{"nums": []any{
				float64(-1), float64(0), float64(1),
			}},
			param:    "nums",
			required: true,
			want:     []int{-1, 0, 1},
			wantErr:  false,
		},
		{
			name:     "empty array",
			args:     map[string]any{"nums": []any{}},
			param:    "nums",
			required: true,
			want:     []int{},
			wantErr:  false,
		},
		{
			name:     "missing required param",
			args:     map[string]any{},
			param:    "nums",
			required: true,
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "missing optional param",
			args:     map[string]any{},
			param:    "nums",
			required: false,
			want:     []int{},
			wantErr:  false,
		},
		{
			name: "invalid array with string",
			args: map[string]any{"nums": []any{
				float64(1), "2", float64(3),
			}},
			param:    "nums",
			required: true,
			want:     nil,
			wantErr:  true,
		},
		{
			name: "invalid array with boolean",
			args: map[string]any{"nums": []any{
				float64(1), true, float64(3),
			}},
			param:    "nums",
			required: true,
			want:     nil,
			wantErr:  true,
		},
		{
			name: "invalid array with nil",
			args: map[string]any{"nums": []any{
				float64(1), nil, float64(3),
			}},
			param:    "nums",
			required: true,
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "wrong type (string instead of array)",
			args:     map[string]any{"nums": "not an array"},
			param:    "nums",
			required: true,
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "nil value",
			args:     map[string]any{"nums": nil},
			param:    "nums",
			required: true,
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(tt.args)
			got, err := p.GetArrayOfIntegers(tt.param, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetArrayOfIntegers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetArrayOfIntegers() = %v, want %v", got, tt.want)
			}
		})
	}
}
