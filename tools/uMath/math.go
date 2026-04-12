package uMath

import (
	"fmt"
	"math"
)

// Integer 泛型整数约束
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Number 泛型数值约束（整数 + 浮点数）
type Number interface {
	Integer | ~float32 | ~float64
}

// Ordered 泛型有序类型约束
type Ordered interface {
	Number | ~string
}

// Abs 返回数值的绝对值
//
// 使用示例：
//
//	uMath.Abs(-5)     // 5
//	uMath.Abs(-3.14)  // 3.14
//	uMath.Abs(int64(-100)) // int64(100)
func Abs[T Number](v T) T {
	if v < 0 {
		return -v
	}
	return v
}

// Max 返回两个值中较大的一个
//
// 使用示例：
//
//	uMath.Max(3, 5)          // 5
//	uMath.Max(int64(10), 20) // int64(20)
//	uMath.Max(3.14, 2.71)    // 3.14
func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Min 返回两个值中较小的一个
//
// 使用示例：
//
//	uMath.Min(3, 5)          // 3
//	uMath.Min(int64(10), 20) // int64(10)
//	uMath.Min(3.14, 2.71)    // 2.71
func Min[T Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// CeilDiv 向上取整除法，等价于 math.Ceil(float64(total) / float64(divisor))
// 常用于分页计算总页数
//
// 使用示例：
//
//	uMath.CeilDiv(10, 3) // 4
//	uMath.CeilDiv(9, 3)  // 3（整除时不进位）
//	uMath.CeilDiv(100, 10) // 10（计算10条/页共需几页）
func CeilDiv(total, divisor int64) int64 {
	if divisor == 0 {
		return 0
	}
	return (total + divisor - 1) / divisor
}

// earthRadius 地球平均半径（米）
const earthRadius = 6371000.0

// GeoDistance 计算两点经纬度之间的直线距离，单位：米
// 采用球面三角法（Spherical Law of Cosines），适用于大多数业务场景
//
// 参数：lng1/lat1 为第一点经纬度，lng2/lat2 为第二点经纬度
//
// 使用示例：
//
//	dist := uMath.GeoDistance(116.397428, 39.90923, 121.473701, 31.230416)
//	// dist ≈ 1068km（北京到上海的直线距离）
func GeoDistance(lng1, lat1, lng2, lat2 float64) float64 {
	rad := math.Pi / 180.0
	lat1 = lat1 * rad
	lng1 = lng1 * rad
	lat2 = lat2 * rad
	lng2 = lng2 * rad

	theta := lng2 - lng1
	dist := math.Acos(
		math.Sin(lat1)*math.Sin(lat2) +
			math.Cos(lat1)*math.Cos(lat2)*math.Cos(theta),
	)
	return dist * earthRadius
}

// GeoDistanceStr 计算两点经纬度距离并格式化为字符串
// 距离 < 1000m 时显示 "XXX.XXm"，否则显示 "X.XXkm"
//
// 使用示例：
//
//	uMath.GeoDistanceStr(116.397, 39.909, 116.407, 39.919) // "1234.56m"
//	uMath.GeoDistanceStr(116.397, 39.909, 121.473, 31.230) // "1068.32km"
func GeoDistanceStr(lng1, lat1, lng2, lat2 float64) string {
	dist := GeoDistance(lng1, lat1, lng2, lat2)
	if dist < 1000 {
		return fmt.Sprintf("%.2fm", dist)
	}
	return fmt.Sprintf("%.2fkm", dist/1000)
}

// GeoPoint 经纬度坐标
type GeoPoint struct {
	Lat float64 // 纬度
	Lng float64 // 经度
}

// GeoSquareBounds 以指定经纬度为中心，计算半径为 distanceMeters 米的正方形四个顶点坐标
// 常用于 LBS 附近搜索的矩形范围查询（结合数据库 BETWEEN 查询）
//
// 返回 map 包含 left_top、right_top、left_bottom、right_bottom 四个点
//
// 使用示例：
//
//	bounds := uMath.GeoSquareBounds(116.397428, 39.90923, 1000)
//	// bounds["left_top"]    = {Lat:39.918, Lng:116.385}
//	// bounds["right_bottom"] = {Lat:39.900, Lng:116.409}
func GeoSquareBounds(lng, lat, distanceMeters float64) map[string]GeoPoint {
	dlng := 2 * math.Asin(math.Sin(distanceMeters/(2*earthRadius))/math.Cos(deg2rad(lat)))
	dlng = rad2deg(dlng)
	dlat := rad2deg(distanceMeters / earthRadius)

	return map[string]GeoPoint{
		"left_top":     {Lat: lat + dlat, Lng: lng - dlng},
		"right_top":    {Lat: lat + dlat, Lng: lng + dlng},
		"left_bottom":  {Lat: lat - dlat, Lng: lng - dlng},
		"right_bottom": {Lat: lat - dlat, Lng: lng + dlng},
	}
}

func deg2rad(degrees float64) float64 {
	return degrees * (math.Pi / 180)
}

func rad2deg(radians float64) float64 {
	return radians * (180 / math.Pi)
}
