// Package utils 通用工具函数
package utils

import (
	"crypto/rand"
	"math/big"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 对密码进行 bcrypt 哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 校验密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ParseInt64 安全解析 int64
func ParseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// RandomBytes 生成随机字节切片
func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(256))
		if err != nil {
			return nil, err
		}
		b[i] = byte(num.Int64())
	}
	return b, nil
}
