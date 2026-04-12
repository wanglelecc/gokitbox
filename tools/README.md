# Tools 工具库

Gokitbox Tools 是一个实用的 Go 语言工具函数集合，提供字符串处理、文件操作、加密解密、数据转换等常用功能。

## 安装

```shell
go get github.com/wanglelecc/gokitbox/tools
```

## 模块概览

| 模块 | 路径 | 功能说明 |
|------|------|----------|
| uAddress | `tools/uAddress` | 网络地址相关工具（获取内网 IP 等） |
| uConvert | `tools/uConvert` | 数据类型转换 |
| uCrypto | `tools/uCrypto` | 加密解密（AES、RSA、Base64、MD5 等） |
| uDate | `tools/uDate` | 日期时间处理 |
| uHash | `tools/uHash` | 哈希算法（MD5、SHA1、SHA256、HMAC 等） |
| uHTTP | `tools/uHTTP` | HTTP 相关工具 |
| uMath | `tools/uMath` | 数学计算工具 |
| uNo | `tools/uNo` | 数字格式化 |
| uOs | `tools/uOs` | 文件/目录操作 |
| uRand | `tools/uRand` | 随机数/随机字符串生成 |
| uSlice | `tools/uSlice` | 切片/数组操作 |
| uSnowflake | `tools/uSnowflake` | 分布式雪花 ID 生成器 |
| uString | `tools/uString` | 字符串处理 |
| uVerify | `tools/uVerify` | 数据校验（身份证、手机号、邮箱等） |

## 使用示例

### uAddress - 网络地址工具

```go
import "github.com/wanglelecc/gokitbox/tools/uAddress"

// 获取本机所有内网 IPv4 地址
ips, err := uAddress.IntranetIP()
// ips = ["192.168.1.100", "10.0.0.5"]

// 判断是否为内网地址
ok := uAddress.IsIntranet("192.168.1.1")  // true
ok = uAddress.IsIntranet("8.8.8.8")       // false
```

### uOs - 文件操作

```go
import "github.com/wanglelecc/gokitbox/tools/uOs"

// 判断目录是否存在
ok, err := uOs.DirExists("/home/dev")

// 判断文件是否存在
ok, err := uOs.FileExists("/tmp/app.log")

// 自动创建目录
err := uOs.MkdirIfNotExist("/tmp/app/logs/2024")

// 写入文件（自动创建父目录）
err := uOs.WriteFile("/tmp/logs/app.log", []byte("hello\n"))

// 追加内容到文件
err := uOs.AppendFile("/tmp/logs/app.log", []byte("new line\n"))

// 复制文件
err := uOs.CopyFile("/tmp/src.txt", "/backup/dst.txt")

// 安全删除（文件不存在不报错）
err := uOs.SafeRemove("/tmp/app.pid")
```

### uRand - 随机数生成

```go
import "github.com/wanglelecc/gokitbox/tools/uRand"

// 生成 [0, 100) 范围内的随机整数
n := uRand.NewRandInt(100)

// 生成 [10, 20] 范围内的随机整数
n := uRand.NewRandIntRange(10, 20)

// 生成 8 位随机字母数字字符串
s := uRand.NewRandString(8)

// 生成 32 位随机十六进制字符串
s := uRand.NewRandHex(32)
```

### uString - 字符串处理

```go
import "github.com/wanglelecc/gokitbox/tools/uString"

// 判空
ok := uString.IsEmpty("")        // true
ok := uString.IsEmpty("  \t")    // true
ok := uString.IsNotEmpty("hello") // true

// 判断是否为纯数字
ok := uString.IsNumeric("123456") // true

// 左填充
s := uString.PadLeft("42", "0", 6)    // "000042"

// 按 rune 截取（支持中文）
s := uString.SubStr("你好世界", 0, 2)     // "你好"

// 截断并添加省略符
s := uString.Truncate("hello world", 5, "...") // "hello..."

// 驼峰转下划线
s := uString.ToSnakeCase("UserName")   // "user_name"

// 下划线转驼峰
s := uString.ToCamelCase("user_name")  // "userName"

// 下划线转大驼峰
s := uString.ToPascalCase("user_name") // "UserName"

// HTML 转义（防 XSS）
s := uString.HTMLEscape("<script>alert(1)</script>")

// 反转字符串
s := uString.Reverse("hello")    // "olleh"
s := uString.Reverse("你好世界")   // "界世好你"
```

### uSnowflake - 雪花 ID

```go
import (
    "context"
    "github.com/wanglelecc/gokitbox/tools/uSnowflake"
)

// 初始化雪花算法（需要 Redis 支持）
// project: 项目名, service: 服务名
ctx := context.Background()
uSnowflake.InitSnowflake(ctx, "my_project", "order_service")

// 生成 int64 类型 ID
id := uSnowflake.NewIdInt64()

// 生成字符串类型 ID
idStr := uSnowflake.NewIdString()
```

> 注意：使用雪花 ID 前必须先调用 InitSnowflake 初始化，且需要 Redis 支持。

### uHash - 哈希算法

```go
import "github.com/wanglelecc/gokitbox/tools/uHash"

// MD5
hash := uHash.MD5("hello")                    // 32位小写
hash = uHash.MD5Upper("hello")                // 32位大写

// SHA1
hash = uHash.SHA1("hello")

// SHA256
hash = uHash.SHA256("hello")

// HMAC-SHA256
hash = uHash.HmacSHA256("data", "secret_key")

// 文件 MD5
hash, err := uHash.FileMD5("/path/to/file")
```

### uCrypto - 加密解密

```go
import "github.com/wanglelecc/gokitbox/tools/uCrypto"

// Base64 编码/解码
encoded := uCrypto.Base64Encode([]byte("hello"))
decoded, err := uCrypto.Base64Decode(encoded)

// URL Safe Base64
encoded := uCrypto.Base64URLEncode([]byte("hello"))

// AES 加密/解密（CBC 模式）
key := []byte("1234567890123456")  // 16/24/32 字节
iv := []byte("1234567890123456")   // 16 字节
encrypted, err := uCrypto.AESEncrypt([]byte("hello"), key, iv)
decrypted, err := uCrypto.AESDecrypt(encrypted, key, iv)

// RSA 加密/解密
encrypted, err := uCrypto.RSAEncrypt([]byte("hello"), publicKey)
decrypted, err := uCrypto.RSADecrypt(encrypted, privateKey)

// RSA 签名/验签
signature, err := uCrypto.RSASign(data, privateKey, crypto.SHA256)
ok, err := uCrypto.RSAVerify(data, signature, publicKey, crypto.SHA256)
```

### uVerify - 数据校验

```go
import "github.com/wanglelecc/gokitbox/tools/uVerify"

// 手机号校验（中国大陆）
ok := uVerify.IsMobile("13800138000")  // true

// 邮箱校验
ok := uVerify.IsEmail("test@example.com")  // true

// 身份证号校验（18位）
ok := uVerify.IsIDCard("11010119900101xxxx")  // true

// URL 校验
ok := uVerify.IsURL("https://example.com")  // true

// IP 地址校验
ok := uVerify.IsIP("192.168.1.1")     // true
ok := uVerify.IsIPv4("192.168.1.1")   // true
ok := uVerify.IsIPv6("::1")           // true
```

### uDate - 日期时间

```go
import "github.com/wanglelecc/gokitbox/tools/uDate"

// 获取当前时间戳（秒）
ts := uDate.Now()

// 获取当前时间戳（毫秒）
ms := uDate.NowMilli()

// 时间戳转字符串
timeStr := uDate.Format(1609459200, "2006-01-02 15:04:05")

// 字符串转时间戳
ts, err := uDate.Parse("2021-01-01 00:00:00", "2006-01-02 15:04:05")

// 获取当天开始/结束时间戳
start, end := uDate.TodayRange()

// 获取当月开始/结束时间戳
start, end := uDate.MonthRange()

// 时间加减
future := uDate.AddDays(1609459200, 7)   // 7天后
past := uDate.AddDays(1609459200, -7)    // 7天前
```

### uSlice - 切片操作

```go
import "github.com/wanglelecc/gokitbox/tools/uSlice"

// 去重
unique := uSlice.UniqueInt([]int{1, 2, 2, 3, 3, 3})  // [1 2 3]
uniqueStr := uSlice.UniqueString([]string{"a", "b", "b"})  // ["a" "b"]

// 包含判断
ok := uSlice.ContainsInt([]int{1, 2, 3}, 2)  // true
ok = uSlice.ContainsString([]string{"a", "b"}, "a")  // true

// 交集
inter := uSlice.IntersectionInt([]int{1, 2, 3}, []int{2, 3, 4})  // [2 3]

// 并集
union := uSlice.UnionInt([]int{1, 2}, []int{2, 3})  // [1 2 3]

// 差集
diff := uSlice.DifferenceInt([]int{1, 2, 3}, []int{2, 3, 4})  // [1]

// 反转
reversed := uSlice.ReverseInt([]int{1, 2, 3})  // [3 2 1]
```

## 更多文档

各子模块详细文档请参考源码中的函数注释，每个函数都包含详细的使用示例。
