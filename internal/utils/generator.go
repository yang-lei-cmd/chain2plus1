// Package utils 通用工具函数 - 邀请码生成
package utils

import (
	"fmt"
	"math/rand"
	"time"
)

// GenerateInviteCode 生成唯一邀请码（固定16位：12位时间戳+4位随机）
func GenerateInviteCode() string {
	ts := time.Now().UnixNano() % 1000000000000 // 取后12位
	nonce := rand.Intn(0x1000)
	return fmt.Sprintf("%012X%03X", ts, nonce)
}

// GenerateUserID 生成唯一用户ID
func GenerateUserID() string {
	return fmt.Sprintf("U%d%08x", time.Now().UnixNano(), rand.Int63())
}

// GenerateOrderNo 生成唯一订单号
func GenerateOrderNo() string {
	prefix := time.Now().Format("20060102150405")
	suffix := randomString(8)
	return fmt.Sprintf("ORD%s%s", prefix, suffix)
}

func randomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range result {
		idx := r.Intn(len(chars))
		result[i] = chars[idx]
	}
	return string(result)
}
