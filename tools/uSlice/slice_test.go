package uSlice

import (
	"reflect"
	"testing"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		item     int
		expected bool
	}{
		{"存在", []int{1, 2, 3}, 2, true},
		{"不存在", []int{1, 2, 3}, 4, false},
		{"空切片", []int{}, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Contains(tt.slice, tt.item)
			if got != tt.expected {
				t.Errorf("Contains(%v, %d) = %v, want %v", tt.slice, tt.item, got, tt.expected)
			}
		})
	}
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected int
	}{
		{"存在", []string{"a", "b", "c"}, "b", 1},
		{"不存在", []string{"a", "b", "c"}, "d", -1},
		{"空切片", []string{}, "a", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IndexOf(tt.slice, tt.item)
			if got != tt.expected {
				t.Errorf("IndexOf(%v, %q) = %d, want %d", tt.slice, tt.item, got, tt.expected)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{"有重复", []int{1, 2, 2, 3, 3, 3}, []int{1, 2, 3}},
		{"无重复", []int{1, 2, 3}, []int{1, 2, 3}},
		{"空切片", []int{}, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Unique(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Unique(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCompact(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{"含零", []int{0, 1, 0, 2, 3}, []int{1, 2, 3}},
		{"无零", []int{1, 2, 3}, []int{1, 2, 3}},
		{"全零", []int{0, 0, 0}, []int{}},
		{"空切片", []int{}, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compact(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Compact(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	expected := []int{2, 4}
	got := Filter(input, func(n int) bool { return n%2 == 0 })
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Filter() = %v, want %v", got, expected)
	}
}

func TestMap(t *testing.T) {
	input := []int{1, 2, 3}
	expected := []int{2, 4, 6}
	got := Map(input, func(n int) int { return n * 2 })
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Map() = %v, want %v", got, expected)
	}
}

func TestReduce(t *testing.T) {
	input := []int{1, 2, 3, 4}
	expected := 10
	got := Reduce(input, 0, func(acc, n int) int { return acc + n })
	if got != expected {
		t.Errorf("Reduce() = %d, want %d", got, expected)
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{"正常", []int{1, 2, 3, 4, 5}, []int{5, 4, 3, 2, 1}},
		{"空切片", []int{}, []int{}},
		{"单元素", []int{1}, []int{1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Reverse(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Reverse(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGroupBy(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	got := GroupBy(input, func(n int) string {
		if n%2 == 0 {
			return "even"
		}
		return "odd"
	})
	if len(got["even"]) != 2 || len(got["odd"]) != 3 {
		t.Errorf("GroupBy() = %v, want even:2, odd:3", got)
	}
}

func TestThreeWay(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{2, 3, 4}
	aOnly, both, bOnly := ThreeWay(a, b)
	if !reflect.DeepEqual(aOnly, []int{1}) {
		t.Errorf("ThreeWay() aOnly = %v, want [1]", aOnly)
	}
	if !reflect.DeepEqual(both, []int{2, 3}) {
		t.Errorf("ThreeWay() both = %v, want [2, 3]", both)
	}
	if !reflect.DeepEqual(bOnly, []int{4}) {
		t.Errorf("ThreeWay() bOnly = %v, want [4]", bOnly)
	}
}

func TestChunk(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		size     int
		expected int // 期望的块数
	}{
		{"整除", []int{1, 2, 3, 4}, 2, 2},
		{"有余数", []int{1, 2, 3, 4, 5}, 2, 3},
		{"size为0", []int{1, 2, 3}, 0, 0},
		{"size大于长度", []int{1, 2, 3}, 5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Chunk(tt.input, tt.size)
			if len(got) != tt.expected {
				t.Errorf("Chunk() returned %d chunks, want %d", len(got), tt.expected)
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	input := [][]int{{1, 2}, {3, 4}, {5}}
	expected := []int{1, 2, 3, 4, 5}
	got := Flatten(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Flatten() = %v, want %v", got, expected)
	}
}

func TestDifference(t *testing.T) {
	a := []int{1, 2, 3, 4}
	b := []int{2, 4}
	expected := []int{1, 3}
	got := Difference(a, b)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Difference() = %v, want %v", got, expected)
	}
}

func TestIntersection(t *testing.T) {
	a := []int{1, 2, 3, 4}
	b := []int{2, 4, 6}
	expected := []int{2, 4}
	got := Intersection(a, b)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Intersection() = %v, want %v", got, expected)
	}
}

func TestJoinInts(t *testing.T) {
	input := []int{1, 2, 3}
	expected := "1,2,3"
	got := JoinInts(input, ",")
	if got != expected {
		t.Errorf("JoinInts() = %q, want %q", got, expected)
	}
}

func TestJoinInt64s(t *testing.T) {
	input := []int64{1, 2, 3}
	expected := "1-2-3"
	got := JoinInt64s(input, "-")
	if got != expected {
		t.Errorf("JoinInt64s() = %q, want %q", got, expected)
	}
}

func TestToInt64s(t *testing.T) {
	input := []int{1, 2, 3}
	got := ToInt64s(input)
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("ToInt64s() = %v, want [1, 2, 3]", got)
	}
}

func TestToInts(t *testing.T) {
	input := []int64{1, 2, 3}
	got := ToInts(input)
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("ToInts() = %v, want [1, 2, 3]", got)
	}
}
