// Package middleware JWT 鉴权中间件
package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/linqi/chain2plus1/pkg/jwt"
)

// Context keys used to store/retrieve user info in gin.Context
const (
	UserIDKey   = "user_id"
	UserNameKey = "username"
	UserRoleKey = "role"
)

// CORSMiddleware CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// GetUserID 从 context 中获取 UserID
func GetUserID(c *gin.Context) uint {
	val, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	uid, ok := val.(uint)
	if !ok {
		return 0
	}
	return uid
}

// GetUserName 从 context 中获取 Username
func GetUserName(c *gin.Context) string {
	name, _ := c.Get("username")
	if s, ok := name.(string); ok {
		return s
	}
	return ""
}

// GetUserRole 从 context 中获取 Role
func GetUserRole(c *gin.Context) string {
	role, _ := c.Get("role")
	if s, ok := role.(string); ok {
		return s
	}
	return ""
}

// AuthRequired 需要鉴权的中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Missing authorization header",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid authorization format",
			})
			c.Abort()
			return
		}

		claims, err := jwt.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Store user info in context (convert string user_id to uint)
		userID, _ := strconv.ParseUint(claims.UserID, 10, 32)
		c.Set(UserIDKey, uint(userID))
		c.Set(UserNameKey, claims.Username)
		c.Set(UserRoleKey, claims.Role)
		c.Next()
	}
}

// AdminOnly 仅管理员可访问
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "Admin access required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
