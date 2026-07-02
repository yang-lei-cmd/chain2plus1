// Package handler Phase 7: 数据分析看板 Handler
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/model"
)

// DashboardHandler 数据分析看板处理器
type DashboardHandler struct{}

// NewDashboardHandler 创建看板处理器
func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// GlobalStats 获取全局统计数据
// GET /api/v1/admin/dashboard/stats
func (h *DashboardHandler) GlobalStats(c *gin.Context) {
	db := database.DB

	// 用户总数
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)

	// 订单总数
	var orderCount int64
	db.Model(&model.Order{}).Count(&orderCount)

	// 订单总金额
	type TotalAmount struct {
		Total int64 `json:"total"`
	}
	var totalAmountResult TotalAmount
	db.Model(&model.Order{}).Select("COALESCE(SUM(amount), 0)").Scan(&totalAmountResult)

	// 分润总数
	var profitCount int64
	db.Model(&model.ProfitShare{}).Count(&profitCount)

	// 支付总数
	var paymentCount int64
	db.Model(&model.ThirdPartyPayment{}).Count(&paymentCount)

	// 自由职业者总数
	var freelancerCount int64
	db.Model(&model.Freelancer{}).Count(&freelancerCount)

	// 提现总数
	var withdrawCount int64
	db.Model(&model.Withdraw{}).Count(&withdrawCount)

	c.JSON(http.StatusOK, gin.H{
		"message":       "获取成功",
		"data": gin.H{
			"user_count":      userCount,
			"order_count":     orderCount,
			"order_total":     totalAmountResult.Total,
			"profit_count":    profitCount,
			"payment_count":   paymentCount,
			"freelancer_count": freelancerCount,
			"withdraw_count":  withdrawCount,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// TodayStats 获取今日统计数据
// GET /api/v1/admin/dashboard/today-stats
func (h *DashboardHandler) TodayStats(c *gin.Context) {
	db := database.DB

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	// 今日新增用户
	var todayUsers int64
	db.Model(&model.User{}).Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).Count(&todayUsers)

	// 今日新增订单
	var todayOrders int64
	db.Model(&model.Order{}).Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).Count(&todayOrders)

	// 今日订单金额
	type TodayAmount struct {
		Total int64 `json:"total"`
	}
	var todayAmountResult TodayAmount
	db.Model(&model.Order{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).
		Scan(&todayAmountResult)

	// 今日分润
	var todayProfits int64
	db.Model(&model.ProfitShare{}).Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).Count(&todayProfits)

	// 今日支付成功
	var todayPayments int64
	db.Model(&model.ThirdPartyPayment{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", todayStart, todayEnd, "success").
		Count(&todayPayments)

	// 今日提现申请
	var todayWithdraws int64
	db.Model(&model.Withdraw{}).Where("created_at BETWEEN ? AND ?", todayStart, todayEnd).Count(&todayWithdraws)

	// 昨日数据对比
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	yesterdayEnd := todayStart

	var yesterdayOrders int64
	db.Model(&model.Order{}).Where("created_at BETWEEN ? AND ?", yesterdayStart, yesterdayEnd).Count(&yesterdayOrders)

	var yesterdayAmountResult TodayAmount
	db.Model(&model.Order{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("created_at BETWEEN ? AND ?", yesterdayStart, yesterdayEnd).
		Scan(&yesterdayAmountResult)

	// 计算环比增长
	orderGrowth := "0.00"
	if yesterdayOrders > 0 {
		growth := float64(todayOrders-yesterdayOrders) / float64(yesterdayOrders) * 100
		orderGrowth = strconv.FormatFloat(growth, 'f', 2, 64)
	}

	amountGrowth := "0.00"
	if yesterdayAmountResult.Total > 0 {
		growth := float64(todayAmountResult.Total-yesterdayAmountResult.Total) / float64(yesterdayAmountResult.Total) * 100
		amountGrowth = strconv.FormatFloat(growth, 'f', 2, 64)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"today": gin.H{
				"new_users":    todayUsers,
				"orders":       todayOrders,
				"order_amount": todayAmountResult.Total,
				"profits":      todayProfits,
				"payments":     todayPayments,
				"withdraws":    todayWithdraws,
			},
			"yesterday_orders":  yesterdayOrders,
			"yesterday_amount":  yesterdayAmountResult.Total,
			"order_growth":      orderGrowth + "%",
			"amount_growth":     amountGrowth + "%",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// RevenueTrend 获取收益趋势数据（最近30天）
// GET /api/v1/admin/dashboard/revenue-trend?days=30
func (h *DashboardHandler) RevenueTrend(c *gin.Context) {
	db := database.DB

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days > 90 {
		days = 90
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days+1)

	// 按日期统计订单金额
	rows, err := db.Raw(`
		SELECT DATE(created_at) as date, COUNT(*) as order_count, COALESCE(SUM(amount), 0) as total_amount
		FROM orders
		WHERE deleted_at IS NULL AND created_at >= ? AND created_at <= ?
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`, startDate, endDate).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	type DailyStats struct {
		Date      string `json:"date"`
		OrderCount int   `json:"order_count"`
		TotalAmount int64 `json:"total_amount"`
	}

	var dailyStats []DailyStats
	for rows.Next() {
		var stat DailyStats
		var dateStr string
		var orderCount int
		var totalAmount int64
		db.ScanRows(rows, &map[string]interface{}{
			"date":         &dateStr,
			"order_count":  &orderCount,
			"total_amount": &totalAmount,
		})
		stat.Date = dateStr
		stat.OrderCount = orderCount
		stat.TotalAmount = totalAmount
		dailyStats = append(dailyStats, stat)
	}

	// 补全缺失的日期
	labels := make([]string, 0, days)
	amounts := make([]int64, 0, days)
	orderCounts := make([]int, 0, days)

 currentDate := startDate
	dateMap := make(map[string]*DailyStats)
	for _, stat := range dailyStats {
		dateMap[stat.Date] = &stat
	}

	for i := 0; i < days; i++ {
		dateStr := currentDate.Format("2006-01-02")
		labels = append(labels, dateStr)
		if stat, ok := dateMap[dateStr]; ok {
			amounts = append(amounts, stat.TotalAmount)
			orderCounts = append(orderCounts, stat.OrderCount)
		} else {
			amounts = append(amounts, 0)
			orderCounts = append(orderCounts, 0)
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"labels":       labels,
			"amounts":      amounts,
			"order_counts": orderCounts,
			"days":         days,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// UserGrowth 获取用户增长数据（最近30天）
// GET /api/v1/admin/dashboard/user-growth?days=30
func (h *DashboardHandler) UserGrowth(c *gin.Context) {
	db := database.DB

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days > 90 {
		days = 90
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days+1)

	var dailyUsers []model.User
	db.Where("created_at >= ? AND created_at <= ?", startDate, endDate).Find(&dailyUsers)

	type CountMap map[string]int
	dailyCount := make(CountMap)
	for _, u := range dailyUsers {
		dateStr := u.CreatedAt.Format("2006-01-02")
		dailyCount[dateStr]++
	}

	labels := make([]string, 0, days)
	counts := make([]int, 0, days)

	currentDate := startDate
	for i := 0; i < days; i++ {
		dateStr := currentDate.Format("2006-01-02")
		labels = append(labels, dateStr)
		counts = append(counts, dailyCount[dateStr])
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// 累计用户数
	var totalUsers int64
	db.Model(&model.User{}).Count(&totalUsers)

	var activeUsers int64
	db.Model(&model.User{}).Where("status = 1").Count(&activeUsers)

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"labels":       labels,
			"counts":       counts,
			"total_users":  totalUsers,
			"active_users": activeUsers,
			"days":         days,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// OrderStatistics 获取订单统计数据
// GET /api/v1/admin/dashboard/order-stats
func (h *DashboardHandler) OrderStatistics(c *gin.Context) {
	db := database.DB

	// 按状态统计
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusCounts []StatusCount
	db.Model(&model.Order{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts)

	// 按支付方式统计
	var paymentMethodCounts []StatusCount
	db.Model(&model.Order{}).
		Select("payment_method, COUNT(*) as count").
		Where("payment_method != ''").
		Group("payment_method").
		Scan(&paymentMethodCounts)

	// 近7天订单趋势
	var recentOrders []model.Order
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	db.Where("created_at >= ?", sevenDaysAgo).Find(&recentOrders)

	type DailyCount struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	dailyOrderCount := make(map[string]int)
	for _, o := range recentOrders {
		dateStr := o.CreatedAt.Format("2006-01-02")
		dailyOrderCount[dateStr]++
	}

	recentLabels := make([]string, 0, 7)
	recentCounts := make([]int, 0, 7)
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		recentLabels = append(recentLabels, dateStr)
		recentCounts = append(recentCounts, dailyOrderCount[dateStr])
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"by_status":        statusCounts,
			"by_payment":       paymentMethodCounts,
			"recent_7_days":    gin.H{"labels": recentLabels, "counts": recentCounts},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// TopUsers 获取收益Top用户排行榜
// GET /api/v1/admin/dashboard/top-users?limit=10
func (h *DashboardHandler) TopUsers(c *gin.Context) {
	db := database.DB

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit > 50 {
		limit = 50
	}

	var users []model.User
	db.Order("total_earned DESC").Limit(limit).Find(&users)

	type TopUser struct {
		ID          uint   `json:"id"`
		Username    string `json:"username"`
		TotalEarned int64  `json:"total_earned"`
		OrderCount  int64  `json:"order_count"`
		InviteCount int64  `json:"invite_count"`
	}

	var topUsers []TopUser
	for _, u := range users {
		var orderCount int64
		db.Model(&model.Order{}).Where("user_id = ?", u.ID).Count(&orderCount)

		var inviteCount int64
		db.Model(&model.User{}).Where("parent_id = ?", u.ID).Count(&inviteCount)

		topUsers = append(topUsers, TopUser{
			ID:          u.ID,
			Username:    u.Username,
			TotalEarned: u.TotalEarned,
			OrderCount:  orderCount,
			InviteCount: inviteCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "获取成功",
		"data":      topUsers,
		"total":     len(topUsers),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// FreelanceStatistics 获取自由职业模块统计
// GET /api/v1/admin/dashboard/freelance-stats
func (h *DashboardHandler) FreelanceStatistics(c *gin.Context) {
	db := database.DB

	// 自由职业者统计
	type FreelancerStatus struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var freelancerStatusCounts []FreelancerStatus
	db.Model(&model.Freelancer{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&freelancerStatusCounts)

	// 任务统计
	var taskStatusCounts []FreelancerStatus
	db.Model(&model.Task{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&taskStatusCounts)

	// 结算统计
	var settlementStats struct {
		TotalSettlements int64   `json:"total_settlements"`
		TotalAmount      int64   `json:"total_amount"`
		PendingAmount    int64   `json:"pending_amount"`
	}
	db.Model(&model.Settlement{}).
		Select("COUNT(*) as total_settlements, COALESCE(SUM(net_amount), 0) as total_amount").
		Scan(&settlementStats)
	db.Model(&model.Settlement{}).
		Where("status = 'pending'").
		Select("COALESCE(SUM(net_amount), 0) as pending_amount").
		Scan(&settlementStats)

	// 评分统计
	var avgRating float64
	db.Model(&model.Rating{}).Select("AVG(score)").Scan(&avgRating)

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"freelancer_status": freelancerStatusCounts,
			"task_status":       taskStatusCounts,
			"settlement":        settlementStats,
			"avg_rating":        avgRating,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
