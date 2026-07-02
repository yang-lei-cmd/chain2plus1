// Package handler Phase C: 审计日志处理器
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/linqi/chain2plus1/internal/middleware"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/model"
)

// ListAuditLogs 获取审计日志列表 (管理员)
// GET /api/v1/admin/audit-logs
func ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	action := c.Query("action")

	db := database.DB.Model(&model.AuditLog{})
	if action != "" {
		db = db.Where("action = ?", action)
	}

	var total int64
	db.Count(&total)

	var logs []model.AuditLog
	db.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"audit_logs": logs,
		"pagination": gin.H{
			"total": total,
			"page":  page,
			"pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// WriteAuditLog 写入审计日志
func WriteAuditLog(c *gin.Context, action, target, detail string) {
	adminID := middleware.GetUserID(c)
	username := middleware.GetUserName(c)
	ip := c.ClientIP()

	detailBytes, _ := json.Marshal(map[string]string{"detail": detail})
	log := model.AuditLog{
		UserID:   adminID,
		Username: username,
		Action:   action,
		Target:   target,
		Detail:   string(detailBytes),
		IP:       ip,
	}
	database.DB.Create(&log)
}
