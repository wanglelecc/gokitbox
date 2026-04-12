package rpcxclient

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// 鉴权头部常量
const (
	AuthHeaderAppId     = "X-Auth-AppId"
	AuthHeaderTimestamp = "X-Auth-Timestamp"
	AuthHeaderNonce     = "X-Auth-Nonce"
	AuthHeaderSign      = "X-Auth-Sign"
)

// genRpcAuth 生成新的 HMAC-SHA256 鉴权信息（推荐）
// 返回: timestamp, nonce, sign
func genRpcAuth() (string, string, string) {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := genNonce()
	sign := genHmacSha256Sign(appId, appKey, now, nonce)
	return now, nonce, sign
}

// genRpcAuthLegacy 兼容旧的 MD5 鉴权（已废弃，仅用于向后兼容）
// 返回: timestamp, sign
func genRpcAuthLegacy() (string, string) {
	now := strconv.Itoa(int(time.Now().Unix()))
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(appId + "&" + now + appKey))
	cipherStr := md5Ctx.Sum(nil)
	signstr := hex.EncodeToString(cipherStr)
	return now, signstr
}

// genRpcAuthCtx 使用指定的 appId/appKey 生成 HMAC-SHA256 鉴权
// 返回: timestamp, nonce, sign
func genRpcAuthCtx(appId, appKey string) (string, string, string) {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := genNonce()
	sign := genHmacSha256Sign(appId, appKey, now, nonce)
	return now, nonce, sign
}

// genRpcAuthCtxLegacy 使用指定的 appId/appKey 生成 MD5 鉴权（已废弃）
// 返回: timestamp, sign
func genRpcAuthCtxLegacy(appId, appKey string) (string, string) {
	now := strconv.Itoa(int(time.Now().Unix()))
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(appId + "&" + now + appKey))
	cipherStr := md5Ctx.Sum(nil)
	signstr := hex.EncodeToString(cipherStr)
	return now, signstr
}

// genNonce 生成随机 nonce
func genNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read 极少失败，但如果失败则 panic（安全考虑）
		panic(fmt.Errorf("failed to generate nonce: %w", err))
	}
	return hex.EncodeToString(b)
}

// genHmacSha256Sign 生成 HMAC-SHA256 签名
// 签名格式: HMAC-SHA256(appId + "|" + timestamp + "|" + nonce, key)
func genHmacSha256Sign(appId, key, timestamp, nonce string) string {
	message := appId + "|" + timestamp + "|" + nonce
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
