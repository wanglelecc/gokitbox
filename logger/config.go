package logger

import (
	"crypto/rand"
	"math/big"
	"os"
	"strconv"
	"strings"
)

var joinMod = false

type Config struct {
	//日志输出文件
	FileName string
	//日志级别
	Level string
	//日志标签 多日志时使用
	Tag string
	//日志格式
	Format string
	//旧日志保留5个备份
	MaxBackups string
	//日志最大保存MB
	MaxSize string
	//最多保留30个日志 和MaxBackups参数配置1个就可以
	MaxAge string
	// gzip包 默认false
	Compress bool
	//Console输出
	Console bool
	//SuffixEnable环境变量
	SuffixEnv string
	//日志的本地时间
	LocalTime bool
}

func (c *Config) SetConfigMap(conf map[string]string) *Config {
	for k, v := range conf {
		switch k {
		case "fileName":
			c.FileName = v
		case "level":
			c.Level = v
		case "tag":
			c.Tag = v
		case "format":
			c.Format = v
		case "maxBackups":
			c.MaxBackups = v
		case "maxSize":
			c.MaxSize = v
		case "maxAge":
			c.MaxAge = v
		case "compress":
			c.Compress = v != "false"
		case "console":
			c.Console = v == "true"
		case "suffixEnv":
			c.SuffixEnv = v
		case "localTime":
			c.LocalTime = !(v == "false")
		}
	}

	return c
}

// Parse a number with K/M/G suffixes based on thousands (1000) or 2^10 (1024)
func strToNumSuffix(str string, mult int) int {
	num := 1
	if len(str) > 1 {
		switch str[len(str)-1] {
		case 'G', 'g':
			num *= mult
			fallthrough
		case 'M', 'm':
			num *= mult
			fallthrough
		case 'K', 'k':
			num *= mult
			str = str[0 : len(str)-1]
		}
	}
	parsed, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return parsed * num
}

func getRandSuffixEnv(suffixEnv string) string {
	suffix := ""
	if os.Getenv(suffixEnv) == "true" {
		suffix = "." + GetRandomString(10)
	}
	return suffix
}

func GetRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := make([]byte, l)
	max := big.NewInt(int64(len(bytes)))
	for i := 0; i < l; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			result[i] = bytes[0]
			continue
		}
		result[i] = bytes[idx.Int64()]
	}
	return string(result)
}

func fileRandRename(file, suffix string) string {
	if strings.HasSuffix(file, ".log") {
		newname := file[:len(file)-4] + suffix + ".log"
		return newname
	}
	return file
}
