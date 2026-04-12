package uMath

import (
	"math"
	"testing"
)

func TestAbs(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"正数", 10, 10},
		{"负数", -10, 10},
		{"零", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Abs(tt.input)
			if got != tt.expected {
				t.Errorf("Abs(%d) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a大", 10, 5, 10},
		{"b大", 5, 10, 10},
		{"相等", 5, 5, 5},
		{"负数", -10, -5, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Max(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("Max(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a小", 5, 10, 5},
		{"b小", 10, 5, 5},
		{"相等", 5, 5, 5},
		{"负数", -10, -5, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Min(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("Min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestCeilDiv(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		divisor  int64
		expected int64
	}{
		{"整除", 9, 3, 3},
		{"有余数", 10, 3, 4},
		{"1页", 5, 10, 1},
		{"除数为0", 10, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CeilDiv(tt.total, tt.divisor)
			if got != tt.expected {
				t.Errorf("CeilDiv(%d, %d) = %d, want %d", tt.total, tt.divisor, got, tt.expected)
			}
		})
	}
}

func TestGeoDistance(t *testing.T) {
	// 测试北京天安门到上海外滩的距离（约 1068km）
	// 天安门: 116.397428, 39.90923
	// 外滩: 121.473701, 31.230416
	got := GeoDistance(116.397428, 39.90923, 121.473701, 31.230416)
	expected := 1068000.0 // 约 1068km，单位米
	diff := math.Abs(got - expected)
	// 允许 1% 误差
	if diff/expected > 0.01 {
		t.Errorf("GeoDistance() = %f, want around %f", got, expected)
	}
}

func TestGeoDistanceStr(t *testing.T) {
	// 短距离（米）
	got := GeoDistanceStr(116.397, 39.909, 116.407, 39.919)
	if !contains(got, "m") {
		t.Errorf("GeoDistanceStr() short distance = %s, should contain 'm'", got)
	}

	// 长距离（公里）
	got = GeoDistanceStr(116.397, 39.909, 121.473, 31.230)
	if !contains(got, "km") {
		t.Errorf("GeoDistanceStr() long distance = %s, should contain 'km'", got)
	}
}

func TestGeoSquareBounds(t *testing.T) {
	bounds := GeoSquareBounds(116.397, 39.909, 1000) // 1km 范围

	// 验证返回四个角
	corners := []string{"left_top", "right_top", "left_bottom", "right_bottom"}
	for _, corner := range corners {
		if _, ok := bounds[corner]; !ok {
			t.Errorf("GeoSquareBounds() missing corner: %s", corner)
		}
	}

	// 验证左上角纬度大于中心点
	if bounds["left_top"].Lat <= 39.909 {
		t.Error("left_top.Lat should be > 39.909")
	}
	// 验证左下角纬度小于中心点
	if bounds["left_bottom"].Lat >= 39.909 {
		t.Error("left_bottom.Lat should be < 39.909")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
