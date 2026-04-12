package uConvert

import (
	"testing"
)

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"int", 42, "42"},
		{"int64", int64(42), "42"},
		{"float64", 3.14, "3.14"},
		{"string", "hello", "hello"},
		{"bool", true, "true"},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToString(tt.input)
			if got != tt.expected {
				t.Errorf("ToString(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{"int", 42, 42},
		{"int64", int64(42), 42},
		{"float64", 42.7, 42},
		{"string", "42", 42},
		{"string invalid", "abc", 0},
		{"bool true", true, 1},
		{"bool false", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToInt(tt.input)
			if got != tt.expected {
				t.Errorf("ToInt(%v) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int64
	}{
		{"int", 42, 42},
		{"int64", int64(42), 42},
		{"float64", 42.7, 42},
		{"string", "42", 42},
		{"string invalid", "abc", 0},
		{"bool true", true, 1},
		{"bool false", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToInt64(tt.input)
			if got != tt.expected {
				t.Errorf("ToInt64(%v) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"int", 42, 42.0},
		{"float64", 3.14, 3.14},
		{"string", "3.14", 3.14},
		{"string invalid", "abc", 0},
		{"bool true", true, 1.0},
		{"bool false", false, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToFloat64(tt.input)
			if got != tt.expected {
				t.Errorf("ToFloat64(%v) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"int 1", 1, true},
		{"int 0", 0, false},
		{"string true", "true", true},
		{"string 1", "1", true},
		{"string false", "false", false},
		{"string 0", "0", false},
		{"string no", "no", false},
		{"int 2", 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToBool(tt.input)
			if got != tt.expected {
				t.Errorf("ToBool(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
