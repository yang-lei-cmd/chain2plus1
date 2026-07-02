// Package jwt JWT 令牌工具
package jwt

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// Claims JWT 载荷
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"` // admin, supplier, customer
	jwt.RegisteredClaims
}

var secret []byte
var expireDuration time.Duration

// InitJWT 初始化 JWT 配置
func InitJWT(s string, expireDays int) {
	secret = []byte(s)
	expireDuration = time.Duration(expireDays) * 24 * time.Hour
	if expireDuration == 0 {
		expireDuration = 7 * 24 * time.Hour
	}
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID, username, role string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "chain2plus1",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
