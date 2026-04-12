package uRand

import (
	"testing"
)

func TestNewRandInt(t *testing.T) {
	tests := []struct {
		name string
		n    int
		min  int
		max  int
	}{
		{"范围100", 100, 0, 99},
		{"范围10", 10, 0, 9},
		{"范围1", 1, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRandInt(tt.n)
			if got < tt.min || got >= tt.max+1 {
				t.Errorf("NewRandInt(%d) = %d, want between %d and %d", tt.n, got, tt.min, tt.max)
			}
		})
	}
}

func TestNewRandIntRange(t *testing.T) {
	tests := []struct {
		name string
		min  int
		max  int
	}{
		{"范围10-20", 10, 20},
		{"范围0-100", 0, 100},
		{"负数范围", -10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRandIntRange(tt.min, tt.max)
			if got < tt.min || got > tt.max {
				t.Errorf("NewRandIntRange(%d, %d) = %d, want between %d and %d", tt.min, tt.max, got, tt.min, tt.max)
			}
		})
	}
}

func TestNewRandString(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{"长度8", 8, 8},
		{"长度16", 16, 16},
		{"长度32", 32, 32},
		{"长度0", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRandString(tt.n)
			if len(got) != tt.expected {
				t.Errorf("NewRandString(%d) length = %d, want %d", tt.n, len(got), tt.expected)
			}
		})
	}
}

func TestNewRandHex(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{"长度16", 16, 16},
		{"长度32", 32, 32},
		{"长度64", 64, 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRandHex(tt.n)
			if len(got) != tt.expected {
				t.Errorf("NewRandHex(%d) length = %d, want %d", tt.n, len(got), tt.expected)
			}
			// 验证只包含十六进制字符
			for _, c := range got {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("NewRandHex(%d) contains invalid char: %c", tt.n, c)
				}
			}
		})
	}
}
