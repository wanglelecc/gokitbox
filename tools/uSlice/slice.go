package uSlice

import (
	"fmt"
	"strconv"
	"strings"
)

// Contains 判断切片中是否包含指定元素
//
// 使用示例：
//
//	uSlice.Contains([]int{1, 2, 3}, 2)          // true
//	uSlice.Contains([]string{"a", "b"}, "c")    // false
func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// IndexOf 返回元素在切片中第一次出现的索引，不存在返回 -1
//
// 使用示例：
//
//	uSlice.IndexOf([]string{"a", "b", "c"}, "b") // 1
//	uSlice.IndexOf([]int{1, 2, 3}, 5)            // -1
func IndexOf[T comparable](slice []T, item T) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

// Unique 去除切片中的重复元素，保持原有顺序
//
// 使用示例：
//
//	uSlice.Unique([]int{1, 2, 2, 3, 1})          // [1, 2, 3]
//	uSlice.Unique([]string{"a", "b", "a", "c"})  // ["a", "b", "c"]
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{}, len(slice))
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Compact 去除切片中的零值元素（"", 0, false, nil 等）
//
// 使用示例：
//
//	uSlice.Compact([]int{0, 1, 0, 2, 3})          // [1, 2, 3]
//	uSlice.Compact([]string{"", "a", "", "b"})    // ["a", "b"]
func Compact[T comparable](slice []T) []T {
	var zero T
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if v != zero {
			result = append(result, v)
		}
	}
	return result
}

// Filter 过滤切片，保留满足条件的元素，返回新切片
//
// 使用示例：
//
//	evens := uSlice.Filter([]int{1, 2, 3, 4, 5}, func(n int) bool { return n%2 == 0 })
//	// evens = [2, 4]
func Filter[T any](slice []T, fn func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

// Map 对切片每个元素执行转换，返回等长新切片
//
// 使用示例：
//
//	doubled := uSlice.Map([]int{1, 2, 3}, func(n int) int { return n * 2 })
//	// doubled = [2, 4, 6]
//
//	names := uSlice.Map(users, func(u User) string { return u.Name })
func Map[T, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

// Reduce 对切片做聚合，从初始值开始依次累积
//
// 使用示例：
//
//	sum := uSlice.Reduce([]int{1, 2, 3, 4}, 0, func(acc, n int) int { return acc + n })
//	// sum = 10
func Reduce[T, U any](slice []T, init U, fn func(U, T) U) U {
	acc := init
	for _, v := range slice {
		acc = fn(acc, v)
	}
	return acc
}

// Reverse 反转切片，返回新切片，不修改原切片
//
// 使用示例：
//
//	uSlice.Reverse([]int{1, 2, 3, 4, 5}) // [5, 4, 3, 2, 1]
//	uSlice.Reverse([]string{"a", "b", "c"}) // ["c", "b", "a"]
func Reverse[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, v := range slice {
		result[len(slice)-1-i] = v
	}
	return result
}

// GroupBy 按指定 key 函数对切片进行分组，返回 map
//
// 使用示例：
//
//	// 按奇偶分组
//	grouped := uSlice.GroupBy([]int{1,2,3,4,5}, func(n int) string {
//	    if n%2 == 0 { return "even" }
//	    return "odd"
//	})
//	// grouped = map["even":[2,4] "odd":[1,3,5]]
func GroupBy[T any, K comparable](slice []T, fn func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, v := range slice {
		k := fn(v)
		result[k] = append(result[k], v)
	}
	return result
}

// ThreeWay 对两个切片做三路分割：仅 a 有的元素、a 和 b 共有的元素、仅 b 有的元素
// 输入会先自动去重
//
// 使用示例：
//
//	aOnly, both, bOnly := uSlice.ThreeWay([]int{1,2,3}, []int{2,3,4})
//	// aOnly = [1], both = [2,3], bOnly = [4]
func ThreeWay[T comparable](a, b []T) (aOnly, both, bOnly []T) {
	a = Unique(a)
	b = Unique(b)

	setB := make(map[T]struct{}, len(b))
	for _, v := range b {
		setB[v] = struct{}{}
	}
	setBoth := make(map[T]struct{})

	for _, v := range a {
		if _, ok := setB[v]; ok {
			both = append(both, v)
			setBoth[v] = struct{}{}
		} else {
			aOnly = append(aOnly, v)
		}
	}

	for _, v := range b {
		if _, ok := setBoth[v]; !ok {
			bOnly = append(bOnly, v)
		}
	}
	return
}

// Chunk 将切片按指定大小分组，最后一组元素数量可能不足 size
//
// 使用示例：
//
//	chunks := uSlice.Chunk([]int{1, 2, 3, 4, 5}, 2)
//	// chunks = [[1,2], [3,4], [5]]
func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	result := make([][]T, 0, (len(slice)+size-1)/size)
	for len(slice) > 0 {
		if len(slice) < size {
			size = len(slice)
		}
		result = append(result, slice[:size:size])
		slice = slice[size:]
	}
	return result
}

// Flatten 将二维切片展平为一维切片
//
// 使用示例：
//
//	flat := uSlice.Flatten([][]int{{1, 2}, {3, 4}, {5}})
//	// flat = [1, 2, 3, 4, 5]
func Flatten[T any](slices [][]T) []T {
	total := 0
	for _, s := range slices {
		total += len(s)
	}
	result := make([]T, 0, total)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// Difference 求差集：返回在 a 中但不在 b 中的元素，保持 a 的顺序
//
// 使用示例：
//
//	diff := uSlice.Difference([]int{1, 2, 3, 4}, []int{2, 4})
//	// diff = [1, 3]
func Difference[T comparable](a, b []T) []T {
	set := make(map[T]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}
	result := make([]T, 0)
	for _, v := range a {
		if _, ok := set[v]; !ok {
			result = append(result, v)
		}
	}
	return result
}

// Intersection 求交集：返回同时在 a 和 b 中的元素，保持 a 的顺序
//
// 使用示例：
//
//	inter := uSlice.Intersection([]int{1, 2, 3, 4}, []int{2, 4, 6})
//	// inter = [2, 4]
func Intersection[T comparable](a, b []T) []T {
	set := make(map[T]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}
	result := make([]T, 0)
	for _, v := range a {
		if _, ok := set[v]; ok {
			result = append(result, v)
		}
	}
	return result
}

// JoinInts 将 int 切片用指定分隔符拼接为字符串
//
// 使用示例：
//
//	uSlice.JoinInts([]int{1, 2, 3}, ",") // "1,2,3"
func JoinInts(slice []int, sep string) string {
	s := make([]string, len(slice))
	for i, v := range slice {
		s[i] = strconv.Itoa(v)
	}
	return strings.Join(s, sep)
}

// JoinInt64s 将 int64 切片用指定分隔符拼接为字符串
//
// 使用示例：
//
//	uSlice.JoinInt64s([]int64{1, 2, 3}, ",") // "1,2,3"
func JoinInt64s(slice []int64, sep string) string {
	s := make([]string, len(slice))
	for i, v := range slice {
		s[i] = strconv.FormatInt(v, 10)
	}
	return strings.Join(s, sep)
}

// JoinAny 将任意类型切片用指定分隔符拼接为字符串（使用 fmt.Sprint 格式化）
//
// 使用示例：
//
//	uSlice.JoinAny([]float64{1.1, 2.2, 3.3}, " | ") // "1.1 | 2.2 | 3.3"
func JoinAny[T any](slice []T, sep string) string {
	s := make([]string, len(slice))
	for i, v := range slice {
		s[i] = fmt.Sprint(v)
	}
	return strings.Join(s, sep)
}

// ToInt64s 将 int 切片转换为 int64 切片
//
// 使用示例：
//
//	uSlice.ToInt64s([]int{1, 2, 3}) // []int64{1, 2, 3}
func ToInt64s(slice []int) []int64 {
	result := make([]int64, len(slice))
	for i, v := range slice {
		result[i] = int64(v)
	}
	return result
}

// ToInts 将 int64 切片转换为 int 切片
//
// 使用示例：
//
//	uSlice.ToInts([]int64{1, 2, 3}) // []int{1, 2, 3}
func ToInts(slice []int64) []int {
	result := make([]int, len(slice))
	for i, v := range slice {
		result[i] = int(v)
	}
	return result
}
