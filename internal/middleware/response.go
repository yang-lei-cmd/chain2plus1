// Package middleware 统一响应格式 + 全局错误处理
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/linqi/chain2plus1/pkg/logger"
)

// Response 统一 JSON 响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Success 成功响应 (200)
func Success(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: message,
		Data:    data,
	})
}

// Created 创建成功响应 (201)
func Created(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    201,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应 (通用)
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Error:   message,
	})
}

// BadRequest 400
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, 400, message)
}

// Unauthorized 401
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, 401, message)
}

// Forbidden 403
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, 403, message)
}

// NotFound 404
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, 404, message)
}

// Conflict 409
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, 409, message)
}

// InternalError 500
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, 500, message)
}

// TooManyRequests 429
func TooManyRequests(c *gin.Context, message string) {
	Error(c, http.StatusTooManyRequests, 429, message)
}

// GlobalErrorHandler 全局错误恢复中间件
// 捕获未被 handler 处理的 panic，返回统一错误格式
func GlobalErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Unhandled panic: %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
					Code:    500,
					Message: "Internal server error",
					Error:   "internal_error",
				})
			}
		}()
		c.Next()
	}
}
