// Package router 路由定义 (Phase 5: 新增第三方支付 + 灵活用工路由, Phase 6: 新增 WebSocket)
package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/linqi/chain2plus1/internal/config"
	"github.com/linqi/chain2plus1/internal/event"
	"github.com/linqi/chain2plus1/internal/handler"
	"github.com/linqi/chain2plus1/internal/middleware"
	"github.com/linqi/chain2plus1/pkg/database"
)

// Setup 配置所有路由 (Phase 5: 新增支付和灵活用工路由, Phase 6: 新增 WebSocket)
func Setup(cfg *config.Config, wsHandler *event.WSHandler) *gin.Engine {
	gin.SetMode("release")
	engine := gin.New() // Use gin.New() instead of gin.Default() to avoid duplicate logging

	// Global middleware
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())
	engine.Use(middleware.GlobalErrorHandler())
	engine.Use(middleware.SecurityHeaders())
	engine.Use(middleware.CORSMiddleware())
	// Rate limit: 100 requests/10s per IP on public endpoints
	engine.Use(middleware.RateLimit(100, 10*time.Second))

	// 初始化 Handler
	paymentHandler := handler.NewPaymentHandler(cfg)
	freelanceHandler := handler.NewFreelanceHandler()
	dashboardHandler := handler.NewDashboardHandler()
	// Inject Hub reference into freelance handler for WebSocket events
	freelanceHandler.SetHub(wsHandler.Hub())

	// Health check — liveness
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Chain2Plus1 API v2 running",
		})
	})

	// Readiness check — verifies DB is reachable
	engine.GET("/ready", func(c *gin.Context) {
		sqlDB, err := database.DB.DB()
		dbOK := err == nil && sqlDB.Ping() == nil
		if dbOK {
			c.JSON(200, gin.H{
				"status":   "ok",
				"database": "connected",
			})
		} else {
			c.JSON(503, gin.H{
				"status":   "degraded",
				"database": "disconnected",
				"error":    err.Error(),
			})
		}
	})

	// Swagger API documentation
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := engine.Group("/api/v1")
	{
		// 公开路由
		v1.POST("/auth/register", middleware.AuthRateLimit(), handler.Register)
		v1.POST("/auth/login", middleware.AuthRateLimit(), handler.Login)

		// 需要鉴权的路由
		auth := v1.Group("")
		auth.Use(middleware.AuthRequired())
		{
			auth.GET("/user/profile", handler.GetProfile)
			auth.GET("/user/tree", handler.GetUserTree)
			auth.GET("/order/list", handler.ListOrders)
			auth.POST("/order/create", handler.CreateOrder)
			auth.GET("/profit/list", handler.ListProfits)

			// 提现路由
			auth.POST("/withdraw/apply", handler.ApplyWithdraw)
			auth.GET("/withdraw/list", handler.ListWithdraws)
			auth.POST("/recharge", handler.UserRecharge)

			// 排行榜
			auth.GET("/leaderboard/:type", handler.Leaderboard)

			// ============================================================
			// Phase 5: 第三方支付路由
			// ============================================================
			auth.POST("/payment/create", paymentHandler.CreatePayment)
			auth.GET("/payment/status/:payment_no", paymentHandler.QueryPaymentStatus)
			auth.POST("/payment/refund", paymentHandler.ProcessRefund)
			auth.GET("/payment/list", paymentHandler.GetUserPayments)
			auth.GET("/payment/my-payments", paymentHandler.GetUserPayments)

			// ============================================================
			// Phase 5: 灵活用工路由
			// ============================================================
			auth.POST("/freelancer/register", freelanceHandler.RegisterFreelancer)
			auth.GET("/freelancer/:id", freelanceHandler.GetFreelancerProfile)

			// 任务相关
			auth.POST("/task/publish", freelanceHandler.CreateTask)
			auth.POST("/task/create", freelanceHandler.CreateTask)
			auth.GET("/task/list", freelanceHandler.GetTaskList)
			auth.GET("/task/:id", freelanceHandler.GetTaskDetail)
			auth.POST("/task/:id/accept", freelanceHandler.AssignTask)
			auth.POST("/task/assign", freelanceHandler.AssignTask)
			auth.GET("/task/my-tasks", freelanceHandler.GetTaskList)
			auth.GET("/task/published", freelanceHandler.GetTaskList)
			auth.POST("/task/:id/submit", freelanceHandler.SubmitWork)
			auth.POST("/task/:id/review", freelanceHandler.ReviewWork)

			// 工时/结算/评分
			auth.POST("/timelog/create", freelanceHandler.CreateTimeLog)
			auth.POST("/time-log", freelanceHandler.CreateTimeLog)
			auth.GET("/timelog/list", freelanceHandler.ListTimeLogs)
			auth.GET("/time-log/list", freelanceHandler.ListTimeLogs)
			auth.POST("/settlement/create", freelanceHandler.CreateSettlement)
			auth.POST("/settlement", freelanceHandler.CreateSettlement)
			auth.GET("/settlement/list", freelanceHandler.ListSettlements)

			// 评分相关
			auth.POST("/rating/create", freelanceHandler.CreateRating)
			auth.POST("/rating", freelanceHandler.CreateRating)
			auth.GET("/rating/list", freelanceHandler.ListRatings)
			auth.GET("/rating/stats/:freelancer_id", freelanceHandler.GetRatingStats)

			// 管理员扩展: 支付+供应商+商品
			auth.GET("/admin/payments", paymentHandler.GetUserPayments)
		}

		// ============================================================
		// Phase 5: 支付回调路由 (不需要鉴权)
		// ============================================================
		v1.POST("/payment/wechat/notify", paymentHandler.HandleWechatCallback)
		v1.POST("/payment/alipay/notify", paymentHandler.HandleAlipayCallback)

		// ============================================================
		// Phase 5: 管理后台路由 (新增灵活用工审核)
		// ============================================================
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthRequired())
		{
			admin.GET("/stats", handler.AdminStats)
			admin.GET("/withdraw", handler.AdminListWithdraws)
			admin.GET("/withdraw/list", handler.AdminListWithdraws)
			admin.PATCH("/withdraw/:id/approve", handler.ApproveWithdraw)
			admin.GET("/orders", handler.AdminListOrders)
			admin.GET("/users", handler.ListUsers)
			admin.PATCH("/users/:id/status", handler.ToggleUserStatus)
			admin.GET("/suppliers", handler.ListSuppliers)
			admin.GET("/products", handler.ListProducts)
			admin.POST("/recharge", handler.AdminRecharge)

			// 对账（管理员）
			admin.GET("/payment/reconcile", paymentHandler.ReconcilePayments)

			// 灵活用工管理
			admin.GET("/freelancers", freelanceHandler.ListFreelancers)
			admin.PATCH("/freelancer/:id/approve", freelanceHandler.ApproveFreelancer)
			admin.PATCH("/freelancer/:id/reject", freelanceHandler.RejectFreelancer)

			// ============================================================
			// Phase 7: 数据分析看板路由
			// ============================================================
			admin.GET("/dashboard/stats", dashboardHandler.GlobalStats)
			admin.GET("/dashboard/today-stats", dashboardHandler.TodayStats)
			admin.GET("/dashboard/revenue-trend", dashboardHandler.RevenueTrend)
			admin.GET("/dashboard/user-growth", dashboardHandler.UserGrowth)
			admin.GET("/dashboard/order-stats", dashboardHandler.OrderStatistics)
			admin.GET("/dashboard/top-users", dashboardHandler.TopUsers)
			admin.GET("/dashboard/freelance-stats", dashboardHandler.FreelanceStatistics)

			// ============================================================
			// Phase C: 审计日志路由
			// ============================================================
			admin.GET("/audit-logs", handler.ListAuditLogs)

			// ============================================================
			// 方向2: CSV导出 + 代理商报表
			// ============================================================
			admin.GET("/export/profits", handler.ExportProfitsCSV)
			admin.GET("/export/orders", handler.ExportOrdersCSV)
			admin.GET("/export/withdraws", handler.ExportWithdrawsCSV)
			admin.GET("/agent-report/:user_id", handler.AgentReport)
			admin.GET("/team-tree/:user_id", handler.TeamTree)
		}
	}

	// ============================================================
	// Phase 6: WebSocket real-time notifications (public route, auth via query param)
	// ============================================================
	engine.GET("/ws", func(c *gin.Context) {
		wsHandler.HandleConnection(c.Writer, c.Request)
	})

	// WS health check
	engine.GET("/ws/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"connected_clients": wsHandler.Hub().ClientsCount()})
	})

	return engine
}
