// Package handler 方向2: CSV导出 + 代理商报表
package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linqi/chain2plus1/internal/middleware"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/model"
)

// ExportProfitsCSV 导出收益明细 CSV
// GET /api/v1/admin/export/profits?user_id=X&start=2026-01-01&end=2026-12-31
func ExportProfitsCSV(c *gin.Context) {
	userID := c.Query("user_id")
	start := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	end := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	db := database.DB.Model(&model.ProfitShare{}).
		Joins("LEFT JOIN users ON users.id = profit_share.to_user_id").
		Where("profit_share.created_at BETWEEN ? AND ?", start+" 00:00:00", end+" 23:59:59")

	if userID != "" {
		db = db.Where("profit_share.to_user_id = ?", userID)
	}

	var profits []struct {
		model.ProfitShare
		ToUsername string `gorm:"column:username"`
	}
	db.Order("profit_share.created_at DESC").Find(&profits)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="profits_%s.csv"`, end[:10]))
	c.Header("Content-Transfer-Encoding", "binary")
	// Write UTF-8 BOM for Excel compatibility
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	wr := csv.NewWriter(c.Writer)
	wr.Write([]string{"ID", "收款用户", "级别", "金额(元)", "类型", "状态", "订单ID", "创建时间"})
	for _, p := range profits {
		wr.Write([]string{
			strconv.Itoa(int(p.ID)),
			p.ToUsername,
			strconv.Itoa(p.Level),
			fmt.Sprintf("%.2f", float64(p.Amount)/100),
			p.Type,
			p.Status,
			strconv.Itoa(int(p.OrderID)),
			p.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	wr.Flush()
}

// ExportOrdersCSV 导出订单 CSV
func ExportOrdersCSV(c *gin.Context) {
	start := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	end := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	var orders []struct {
		model.Order
		Username    string `gorm:"column:username"`
		ProductName string `gorm:"column:product_name"`
	}
	database.DB.Model(&model.Order{}).
		Joins("LEFT JOIN users ON users.id = orders.user_id").
		Joins("LEFT JOIN products ON products.id = orders.product_id").
		Where("orders.created_at BETWEEN ? AND ?", start+" 00:00:00", end+" 23:59:59").
		Order("orders.created_at DESC").
		Find(&orders)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="orders_%s.csv"`, start[:10]))
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	wr := csv.NewWriter(c.Writer)
	wr.Write([]string{"ID", "订单号", "用户", "商品", "金额(元)", "支付方式", "状态", "创建时间"})
	for _, o := range orders {
		wr.Write([]string{
			strconv.Itoa(int(o.ID)),
			o.OrderNo,
			o.Username,
			o.ProductName,
			fmt.Sprintf("%.2f", float64(o.Amount)/100),
			o.PaymentMethod,
			o.Status,
			o.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	wr.Flush()
}

// ExportWithdrawsCSV 导出提现记录 CSV
func ExportWithdrawsCSV(c *gin.Context) {
	start := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	end := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	var withdraws []struct {
		model.Withdraw
		Username string `gorm:"column:username"`
	}
	database.DB.Model(&model.Withdraw{}).
		Joins("LEFT JOIN users ON users.id = withdraws.user_id").
		Where("withdraws.created_at BETWEEN ? AND ?", start+" 00:00:00", end+" 23:59:59").
		Order("withdraws.created_at DESC").
		Find(&withdraws)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="withdraws_%s.csv"`, start[:10]))
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	wr := csv.NewWriter(c.Writer)
	wr.Write([]string{"ID", "用户", "金额(元)", "手续费", "实际到账", "银行", "账号", "状态", "创建时间"})
	for _, w := range withdraws {
		wr.Write([]string{
			strconv.Itoa(int(w.ID)),
			w.Username,
			fmt.Sprintf("%.2f", float64(w.Amount)/100),
			fmt.Sprintf("%.2f", float64(w.Fee)/100),
			fmt.Sprintf("%.2f", float64(w.ActualAmount)/100),
			fmt.Sprintf("%s %s", w.BankName, w.AccountName),
			w.AccountNo,
			w.Status,
			w.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	wr.Flush()
}

// ============================================================
// 代理商多级报表
// ============================================================

// AgentReport 代理商团队报表
// GET /api/v1/admin/agent-report/:user_id
func AgentReport(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	db := database.DB

	// 1. 用户基本信息
	var user model.User
	if err := db.First(&user, uint(userID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 2. 直属团队人数
	var directCount int64
	db.Model(&model.User{}).Where("parent_id = ?", userID).Count(&directCount)

	// 3. 总团队人数 (BFS)
	totalTeam := int(directCount)
	var queue []uint
	db.Model(&model.User{}).Where("parent_id = ?", userID).Pluck("id", &queue)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		var children []uint
		db.Model(&model.User{}).Where("parent_id = ?", current).Pluck("id", &children)
		totalTeam += len(children)
		queue = append(queue, children...)
	}

	// 4. 累计分润总额
	var totalEarned int64
	db.Model(&model.ProfitShare{}).Where("to_user_id = ?", userID).Select("COALESCE(SUM(amount), 0)").Scan(&totalEarned)

	// 5. 本月分润
	var monthEarned int64
	monthStart := time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02")
	db.Model(&model.ProfitShare{}).
		Where("to_user_id = ? AND created_at >= ?", userID, monthStart+" 00:00:00").
		Select("COALESCE(SUM(amount), 0)").Scan(&monthEarned)

	// 6. 下级业绩排行
	type childPerformance struct {
		UserID   uint   `json:"user_id"`
		Username string `json:"username"`
		Level    int    `json:"level"`
		Orders   int64  `json:"orders"`
		Revenue  int64  `json:"revenue"`
	}
	var topChildren []childPerformance
	db.Raw(`
		SELECT u.id as user_id, u.username, u.level,
			COUNT(DISTINCT o.id) as orders,
			COALESCE(SUM(o.amount), 0) as revenue
		FROM users u
		LEFT JOIN orders o ON o.user_id = u.id AND o.status = 'paid'
		WHERE u.parent_id = ?
		GROUP BY u.id, u.username, u.level
		ORDER BY revenue DESC
		LIMIT 10
	`, userID).Scan(&topChildren)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"level":    user.Level,
			"status":   user.Status,
		},
		"team_stats": gin.H{
			"direct_count": directCount,
			"total_count":  totalTeam,
			"direct_limit": 2, // 2+1 chain unlock requirement
		},
		"earnings": gin.H{
			"total":      totalEarned,
			"this_month": monthEarned,
		},
		"top_children": topChildren,
	})
}

// TeamTree 获取完整团队树 (JSON)
// GET /api/v1/admin/team-tree/:user_id?depth=3
func TeamTree(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}
	maxDepth, _ := strconv.Atoi(c.DefaultQuery("depth", "3"))

	var buildTree func(uid uint, depth int) gin.H
	buildTree = func(uid uint, depth int) gin.H {
		if depth > maxDepth {
			return gin.H{"id": uid, "truncated": true}
		}
		var user model.User
		database.DB.Select("id, username, level, status, balance, total_earned").First(&user, uid)

		var children []uint
		database.DB.Model(&model.User{}).Where("parent_id = ?", uid).Pluck("id", &children)

		var childNodes []gin.H
		for _, cid := range children {
			childNodes = append(childNodes, buildTree(cid, depth+1))
		}

		return gin.H{
			"id":       user.ID,
			"username": user.Username,
			"level":    user.Level,
			"status":   user.Status,
			"earned":   user.TotalEarned,
			"children": childNodes,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tree":   buildTree(uint(userID), 0),
		"depth":  maxDepth,
		"user_id": userID,
	})
}

// GetUserIDFromContext 从 context 获取 user_id (中间件辅助)
func GetUserIDFromContext(c *gin.Context) uint {
	return middleware.GetUserID(c)
}
