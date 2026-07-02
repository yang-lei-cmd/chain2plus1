// Package handler Gin 处理器实现（Phase 2 + Phase 3）
package handler

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/linqi/chain2plus1/internal/engine"
	"github.com/linqi/chain2plus1/internal/middleware"
	"github.com/linqi/chain2plus1/internal/utils"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/dto"
	"github.com/linqi/chain2plus1/pkg/jwt"
	"github.com/linqi/chain2plus1/pkg/logger"
	"github.com/linqi/chain2plus1/pkg/model"
)

// ==================== Register 用户注册 ====================

// Register 处理用户注册请求
func Register(c *gin.Context) {
	var req dto.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.DB

	// 检查用户名是否已存在
	var existingUser model.User
	if err := db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 查找上级用户（通过邀请码，根用户无上级）
	var parent model.User
	var parentLevel int = 0
	if req.InviteCode != "" {
		if err := db.Where("invite_code = ?", req.InviteCode).First(&parent).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "邀请码无效"})
			return
		}
		parentLevel = int(parent.Level)
	}

	// 密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Password hash error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	// 生成邀请码
	inviteCode := utils.GenerateInviteCode()

	// 创建新用户
	newUser := model.User{
		Username:   req.Username,
		Password:   string(hashedPassword),
		Phone:      req.Phone,
		Email:      req.Email,
		Role:       "customer",
		InviteCode: inviteCode,
		Level:      parentLevel + 1,
		Status:     1,
	}
	if parentLevel > 0 {
		newUser.ParentID = &parent.ID
	}

	if err := db.Create(&newUser).Error; err != nil {
		logger.Error("Create user error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	// 创建链动记录
	chainRecord := model.ChainRecord{
		UserID:    newUser.ID,
		Action:    "register",
		Status:    "success",
		Data:      `{"parent_level": 0}`,
	}
	if parentLevel > 0 {
		chainRecord.RelatedID = parent.ID
		chainRecord.Action = "bind"
		chainRecord.Data = `{"parent_level": ` + strconv.Itoa(parentLevel) + `}`
	}
	db.Create(&chainRecord)

	if parentLevel > 0 {
		logger.Info("User registered: %s (parent: %s)", req.Username, parent.Username)
	} else {
		logger.Info("Root user registered: %s", req.Username)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "注册成功",
		"user": dto.UserInfo{
			ID:         newUser.ID,
			Username:   newUser.Username,
			Phone:      newUser.Phone,
			Email:      newUser.Email,
			Role:       newUser.Role,
			Level:      newUser.Level,
			InviteCode: newUser.InviteCode,
		},
	})
}

// ==================== Login 用户登录 ====================

// Login 处理用户登录请求
func Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.DB

	// 查找用户
	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 检查用户状态
	if user.Status != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": "账户已被禁用"})
		return
	}

	// 生成 JWT token
	tokenStr, err := jwt.GenerateToken(strconv.FormatUint(uint64(user.ID), 10), user.Username, user.Role)
	if err != nil {
		logger.Error("JWT token generation error: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败"})
		return
	}


	logger.Info("User logged in: %s", req.Username)

	c.JSON(http.StatusOK, gin.H{
		"token": tokenStr,
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"phone":       user.Phone,
			"email":       user.Email,
			"role":        user.Role,
			"level":       user.Level,
			"balance":     float64(user.Balance) / 100,
			"total_earned": float64(user.TotalEarned) / 100,
			"invite_code": user.InviteCode,
			"created_at":  user.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// ==================== GetProfile 获取用户信息 ====================

// GetProfile 获取当前用户信息
func GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.DB

	var user model.User
	if err := db.Preload("Parent").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 查询团队人数
	var teamCount int64
	db.Model(&model.User{}).Where("level > ? AND parent_id IS NOT NULL", user.Level).Count(&teamCount)

	// 查询上级
	var parent *gin.H
	if user.Parent != nil && user.Parent.ID > 0 {
		parent = &gin.H{
			"id":         user.Parent.ID,
			"username":   user.Parent.Username,
			"level":      user.Parent.Level,
			"invite_code": user.Parent.InviteCode,
		}
	}

	// 查询团队成员
	type TeamMember struct {
		ID       uint   
		Username string 
		Level    int    
	}
	var teamMembers []TeamMember
	db.Where("parent_id = ? OR parent_id IN (SELECT id FROM user WHERE parent_id = ?)", user.ID, user.ID).Find(&teamMembers)

	c.JSON(http.StatusOK, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"phone":         user.Phone,
		"email":         user.Email,
		"role":          user.Role,
		"level":         user.Level,
		"balance":       float64(user.Balance) / 100.0,
		"total_earned":  float64(user.TotalEarned) / 100.0,
		"invite_code":   user.InviteCode,
		"parent":        parent,
		"children_count": 0,
		"team_count":    int(teamCount),
		"team_members":  teamMembers,
	})
}

// ==================== GetUserTree 获取关系链 ====================

// GetUserTree 获取当前用户的完整关系链（含下线）
func GetUserTree(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.DB

	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 查询直属下线（一级）
	var children []model.User
	db.Where("parent_id = ?", user.ID).Find(&children)

	childInfos := make([]gin.H, 0, len(children))
	for _, ch := range children {
		childInfos = append(childInfos, gin.H{
			"id":         ch.ID,
			"username":   ch.Username,
			"level":      ch.Level,
			"invite_code": ch.InviteCode,
		})
	}

	// 查询上级
	var parentInfo *gin.H
	if user.ParentID != nil && *user.ParentID > 0 {
		var parent model.User
		if db.First(&parent, *user.ParentID).Error == nil {
			parentInfo = &gin.H{
				"id":       parent.ID,
				"username": parent.Username,
				"level":    parent.Level,
			}
		}
	}

	// 统计总团队人数
	var totalDownline int64
	db.Model(&model.User{}).Where("parent_id IS NOT NULL").Count(&totalDownline)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"level":        user.Level,
			"invite_code":  user.InviteCode,
		},
		"parent":         parentInfo,
		"children":       childInfos,
		"total_downline": int(totalDownline),
	})
}

// ==================== 订单相关 ====================

// ListOrders 获取订单列表
func ListOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.DB

	var orders []model.Order
	db.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&orders)

	result := make([]dto.OrderInfo, 0, len(orders))
	for _, o := range orders {
		var product model.Product
		db.First(&product, o.ProductID)
		result = append(result, dto.OrderInfo{
			ID:            o.ID,
			OrderNo:       o.OrderNo,
			ProductID:     o.ProductID,
			ProductName:   product.Name,
			ProductImage:  product.ImageURL,
			Amount:       float64(o.Amount) / 100.0,
			PaymentMethod: o.PaymentMethod,
			Status:       o.Status,
			CreatedAt:    o.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"orders": result})
}

// CreateOrder 创建订单并触发分润（Phase 3）
func CreateOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.DB

	var req dto.CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查询商品
	var product model.Product
	if err := db.First(&product, req.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品不存在"})
		return
	}

	if product.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商品已下架"})
		return
	}

	// 生成员订单号
		orderNo := utils.GenerateOrderNo()

	// 创建订单
	paidAt := time.Now()
	order := model.Order{
		UserID:         userID,
		ProductID:      req.ProductID,
		OrderNo:        orderNo,
		Amount:         product.Price,
		PaymentMethod:  req.PaymentMethod,
		Status:         "paid", // 模拟支付成功
		PaidAt:         &paidAt,
		PaymentNo:      "PAY" + orderNo,
	}
	if err := db.Create(&order).Error; err != nil {
		logger.Error("Create order error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建订单失败"})
		return
	}

	// 获取当前用户信息（下单用户）
	var buyer model.User
	db.First(&buyer, userID)

	// 计算分润
	cfg := engine.DefaultChainConfig()
	commissions := engine.CalculateCommission(db, buyer.ID, product.Price, cfg)
	logger.Info("DEBUG: CalculateCommission returned %d records for buyer_id=%d amount=%d", len(commissions), buyer.ID, product.Price)
	for i, c := range commissions {
		logger.Info("DEBUG commission[%d]: to_user=%d level=%d amount=%d", i, c.ToUserID, c.Level, c.Amount)
	}

	// 发放分润
	if len(commissions) > 0 {
		if err := engine.DistributeCommission(db, order.ID, buyer.ID, product.Price, commissions); err != nil {
			logger.Error("Distribute commission error: %v", err)
			// 订单已创建，分润失败不影响
		}
	}

	// 检查上级解锁
	engine.ProcessChainLock(db, buyer, cfg)

	// 构建响应 — 查询订单及关联商品
	var orderWithProduct model.Order
	db.Preload("Product").First(&orderWithProduct, order.ID)

	profitRecords := make([]map[string]interface{}, 0)
	if len(commissions) > 0 {
		// 查询刚刚创建的分润记录
		var profits []model.ProfitShare
		db.Where("order_id = ?", order.ID).Find(&profits)
		for _, p := range profits {
			profitRecords = append(profitRecords, map[string]interface{}{
				"id":          p.ID,
				"level":       p.Level,
				"amount":      float64(p.Amount) / 100.0,
				"type":        p.Type,
				"description": p.Description,
			})
		}
	}

	logger.Info("Order created: %s, amount: %.2f, commissions: %d",
		orderNo, float64(order.Amount)/100.0, len(commissions))

	c.JSON(http.StatusCreated, gin.H{
		"message":     "订单创建成功",
		"order": map[string]interface{}{
			"id":              order.ID,
			"order_no":        order.OrderNo,
			"product_name":    orderWithProduct.Product.Name,
			"amount":          float64(order.Amount) / 100.0,
			"payment_method":  order.PaymentMethod,
			"status":          order.Status,
			"paid_at":         order.PaidAt,
		},
		"commissions": profitRecords,
	})
}

// ListProfits 获取分润记录（Phase 3）
func ListProfits(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.DB

	var profits []model.ProfitShare
	db.Where("to_user_id = ?", userID).
		Preload("Order").
		Order("created_at DESC").
		Limit(50).
		Find(&profits)

	result := make([]dto.ProfitInfo, 0, len(profits))
	for _, p := range profits {
		fromUser := ""
		if p.Order.UserID > 0 {
			db.Model(&model.User{}).Select("username").First(&fromUser, p.Order.UserID)
		}
		result = append(result, dto.ProfitInfo{
			ID:          p.ID,
			OrderNo:     p.Order.OrderNo,
			FromUser:    fromUser,
			Level:       p.Level,
			Amount:      float64(p.Amount) / 100.0,
			Type:        p.Type,
			Status:      p.Status,
			Description: p.Description,
			CreatedAt:   p.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"profits": result})
}

// ==================== 提现相关 (Phase 4) ====================

// ApplyWithdraw 申请提现
func ApplyWithdraw(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUserName(c)
	db := database.DB

	var req dto.WithdrawApplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查询用户信息
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 检查余额是否足够
	if user.Balance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        "余额不足",
			"current_balance": float64(user.Balance) / 100.0,
		})
		return
	}

	// 计算手续费（1%）
	fee := int64(math.Round(float64(req.Amount) * 0.01))
	actualAmount := req.Amount - fee

	// 创建提现记录
	withdraw := model.Withdraw{
		UserID:       user.ID,
		Amount:       req.Amount,
		Fee:          fee,
		ActualAmount: actualAmount,
		BankName:     req.BankName,
		AccountName:  req.AccountName,
		AccountNo:    req.AccountNo,
		Status:       "pending",
	}

	if err := db.Create(&withdraw).Error; err != nil {
		logger.Error("Failed to create withdraw request for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提现申请失败"})
		return
	}

	logger.Info("Withdraw applied: user=%s(%d) amount=%.2f fee=%.2f actual=%.2f",
		username, userID, float64(req.Amount)/100, float64(fee)/100, float64(actualAmount)/100)

	c.JSON(http.StatusOK, gin.H{
		"message":      "提现申请已提交",
		"id":           withdraw.ID,
		"amount":       float64(withdraw.Amount) / 100,
		"fee":          float64(withdraw.Fee) / 100,
		"actual_amount": float64(withdraw.ActualAmount) / 100,
		"status":       withdraw.Status,
		"created_at":   withdraw.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// ListWithdraws 查看提现记录
func ListWithdraws(c *gin.Context) {
	userID := middleware.GetUserID(c)
	db := database.DB

	var withdraws []model.Withdraw
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&withdraws).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	type RecordResp struct {
		ID           uint       `json:"id"`
		Amount       float64    `json:"amount"`
		Fee          float64    `json:"fee"`
		ActualAmount float64    `json:"actual_amount"`
		BankName     string     `json:"bank_name"`
		AccountName  string     `json:"account_name"`
		Status       string     `json:"status"`
		Remark       string     `json:"remark"`
		CreatedAt    string     `json:"created_at"`
		ApprovedAt   *string    `json:"approved_at"`
	}

	resp := make([]RecordResp, len(withdraws))
	for i, w := range withdraws {
		createdAt := w.CreatedAt.Format("2006-01-02 15:04:05")
		record := RecordResp{
			ID:           w.ID,
			Amount:       float64(w.Amount) / 100,
			Fee:          float64(w.Fee) / 100,
			ActualAmount: float64(w.ActualAmount) / 100,
			BankName:     w.BankName,
			AccountName:  w.AccountName,
			Status:       w.Status,
			Remark:       w.Remark,
			CreatedAt:    createdAt,
			ApprovedAt:   nil,
		}
		if w.ApprovedAt != nil {
			s := w.ApprovedAt.Format("2006-01-02 15:04:05")
			record.ApprovedAt = &s
		}
		resp[i] = record
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(withdraws),
		"items": resp,
	})
}

// ApproveWithdraw 管理员审核提现
func ApproveWithdraw(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	// 检查管理员权限
	var currentUser model.User
	if err := database.DB.First(&currentUser, adminID).Error; err != nil || currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	var req dto.WithdrawApproveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	withdrawID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的提现ID"})
		return
	}

	db := database.DB
	var withdraw model.Withdraw
	if err := db.First(&withdraw, uint(withdrawID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "提现记录不存在"})
		return
	}

	if withdraw.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该提现已审核"})
		return
	}

	switch req.Action {
	case "approve":
		withdraw.Status = "approved"
		withdraw.ApprovedBy = &adminID
		now := time.Now()
		withdraw.ApprovedAt = &now
		withdraw.Remark = req.Remark

		// 扣减用户余额
		if err := db.Model(&model.User{}).Where("id = ?", withdraw.UserID).UpdateColumn("balance", gorm.Expr("balance - ?", withdraw.Amount)).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "余额扣减失败"})
			return
		}

		logger.Info("Withdraw approved: id=%d user_id=%d amount=%.2f approved_by=%d",
			withdraw.ID, withdraw.UserID, float64(withdraw.Amount)/100, adminID)

	case "reject":
		if req.Remark == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "拒绝时必须填写原因"})
			return
		}
		withdraw.Status = "rejected"
		withdraw.Remark = req.Remark

		logger.Info("Withdraw rejected: id=%d user_id=%d reason=%s",
			withdraw.ID, withdraw.UserID, req.Remark)
	}

	if err := db.Save(&withdraw).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "审核失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "审核完成",
		"withdraw": withdraw,
	})
}

// AdminListWithdraws 管理员查看所有提现申请
func AdminListWithdraws(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	// 检查管理员权限
	var currentUser model.User
	if err := database.DB.First(&currentUser, adminID).Error; err != nil || currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	db := database.DB
	var withdraws []model.Withdraw
	db.Preload("User").Order("created_at DESC").Limit(50).Find(&withdraws)

	type Item struct {
		ID           uint       `json:"id"`
		UserID       uint       `json:"user_id"`
		Username     string     `json:"username"`
		Amount       float64    `json:"amount"`
		Fee          float64    `json:"fee"`
		ActualAmount float64    `json:"actual_amount"`
		BankName     string     `json:"bank_name"`
		AccountName  string     `json:"account_name"`
		Status       string     `json:"status"`
		Remark       string     `json:"remark"`
		CreatedAt    string     `json:"created_at"`
		ApprovedAt   *string    `json:"approved_at"`
	}

	items := make([]Item, len(withdraws))
	for i, w := range withdraws {
		var username string
		if w.User.ID > 0 {
			username = w.User.Username
		}
		createdAt := w.CreatedAt.Format("2006-01-02 15:04:05")
		item := Item{
			ID:           w.ID,
			UserID:       w.UserID,
			Username:     username,
			Amount:       float64(w.Amount) / 100,
			Fee:          float64(w.Fee) / 100,
			ActualAmount: float64(w.ActualAmount) / 100,
			BankName:     w.BankName,
			AccountName:  w.AccountName,
			Status:       w.Status,
			Remark:       w.Remark,
			CreatedAt:    createdAt,
		}
		if w.ApprovedAt != nil {
			s := w.ApprovedAt.Format("2006-01-02 15:04:05")
			item.ApprovedAt = &s
		}
		items[i] = item
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(withdraws),
		"items": items,
	})
}

// AdminStats 管理员统计数据
func AdminStats(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	db := database.DB

	// 检查管理员权限
	var currentUser model.User
	if err := db.First(&currentUser, adminID).Error; err != nil || currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	// 统计用户数
	var totalUsers, activeUsers int64
	db.Model(&model.User{}).Count(&totalUsers)
	db.Model(&model.User{}).Where("status = 1").Count(&activeUsers)

	// 统计订单数和总收入
	var totalOrders int64
	var totalRevenue int64
	db.Model(&model.Order{}).Count(&totalOrders)
	db.Model(&model.Order{}).Select("SUM(amount)").Scan(&totalRevenue)

	// 统计分润总额
	var totalProfit int64
	db.Model(&model.ProfitShare{}).Select("COALESCE(SUM(amount),0)").Scan(&totalProfit)

	// 今日订单和收入
	today := time.Now().Format("2006-01-02")
	var todayOrders, todayRevenue int64
	db.Model(&model.Order{}).Where("DATE(created_at) = ?", today).Count(&todayOrders)
	db.Model(&model.Order{}).Where("DATE(created_at) = ?", today).Select("COALESCE(SUM(amount),0)").Scan(&todayRevenue)

	// 待审核提现
	var pendingWithdraw int64
	var pendingWithdrawAmount int64
	db.Model(&model.Withdraw{}).Where("status = ?", "pending").Count(&pendingWithdraw)
	db.Model(&model.Withdraw{}).Where("status = ?", "pending").Select("COALESCE(SUM(amount),0)").Scan(&pendingWithdrawAmount)

	// Top 5 收益用户
	type TopUser struct {
		ID          uint    `json:"id"`
		Username    string  `json:"username"`
		TotalEarned int64   `json:"total_earned"`
	}
	var topUsers []TopUser
	db.Table("user").Select("id, username, total_earned").Order("total_earned DESC").Limit(5).Scan(&topUsers)

	topList := make([]dto.LeaderboardItem, len(topUsers))
	for i, u := range topUsers {
		topList[i] = dto.LeaderboardItem{
			UserID:    u.ID,
			Username:  u.Username,
			Ranking:   i + 1,
			RankValue: u.TotalEarned,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_users":        totalUsers,
		"active_users":       activeUsers,
		"total_orders":       totalOrders,
		"total_revenue":      float64(totalRevenue) / 100,
		"total_profit":       float64(totalProfit) / 100,
		"today_orders":       todayOrders,
		"today_revenue":      float64(todayRevenue) / 100,
		"pending_withdraw":   pendingWithdraw,
		"pending_withdraw_amount": float64(pendingWithdrawAmount) / 100,
		"top_users":          topList,
	})
}

// Leaderboard 排行榜
func Leaderboard(c *gin.Context) {
	db := database.DB
	leaderboardType := c.Param("type")

	var title string
	var orderBy string

	switch leaderboardType {
	case "total_earned":
		title = "收益排行榜"
		orderBy = "total_earned DESC"
	case "team_size":
		title = "团队排行榜"
		// 递归计算每个用户的团队总人数（包括直接+间接下级）
		var users []model.User
		db.Find(&users)

		type UserInfo struct {
			ID        uint
			Username  string
			ParentID  *uint
		}

		userMap := make(map[uint]*UserInfo)
		for _, u := range users {
			info := &UserInfo{ID: u.ID, Username: u.Username}
			if u.ParentID != nil {
				info.ParentID = u.ParentID
			}
			userMap[u.ID] = info
		}

		teamSize := make(map[uint]int)
		for _, info := range userMap {
			size := 0
			queue := []uint{info.ID}
			visited := make(map[uint]bool)
			for len(queue) > 0 {
				current := queue[0]
				queue = queue[1:]
				if visited[current] {
					continue
				}
				visited[current] = true
				for _, u := range userMap {
					if u.ParentID != nil && *u.ParentID == current && !visited[u.ID] {
						size++
						queue = append(queue, u.ID)
					}
				}
			}
			teamSize[info.ID] = size
		}

		type TeamResult struct {
			UserID    uint
			Username  string
			TeamCount int
		}

		var results []TeamResult
		for _, info := range userMap {
			results = append(results, TeamResult{
				UserID:    info.ID,
				Username:  info.Username,
				TeamCount: teamSize[info.ID],
			})
		}

		// 排序
		for i := 0; i < len(results); i++ {
			for j := i + 1; j < len(results); j++ {
				if results[j].TeamCount > results[i].TeamCount {
					results[i], results[j] = results[j], results[i]
				}
			}
		}

		if len(results) > 50 {
			results = results[:50]
		}

		items := make([]dto.LeaderboardItem, len(results))
		for i, r := range results {
			items[i] = dto.LeaderboardItem{
				UserID:    r.UserID,
				Username:  r.Username,
				Ranking:   i + 1,
				RankValue: int64(r.TeamCount),
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"type":        "team_size",
			"title":       title,
			"items":       items,
			"total":       len(items),
			"update_time": time.Now().Format("2006-01-02 15:04:05"),
		})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的排行榜类型"})
		return
	}

	// 按 total_earned 或 recharge 排行
	var users []model.User
	db.Select("id, username, total_earned").Order(orderBy).Limit(50).Find(&users)

	items := make([]dto.LeaderboardItem, len(users))
	for i, u := range users {
		items[i] = dto.LeaderboardItem{
			UserID:    u.ID,
			Username:  u.Username,
			Ranking:   i + 1,
			RankValue: int64(u.TotalEarned / 100),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"type":         leaderboardType,
		"title":        title,
		"items":        items,
		"total":        len(items),
		"update_time":  time.Now().Format("2006-01-02 15:04:05"),
	})
}


// ==================== 管理员功能 (Phase 4) ====================

// ListUsers 管理员查看所有用户
func ListUsers(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	// 检查管理员权限
	var currentUser model.User
	if err := database.DB.First(&currentUser, adminID).Error; err != nil || currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	db := database.DB
	var users []model.User
	db.Select("id, username, phone, email, role, level, balance, total_earned, status, created_at").Order("created_at DESC").Limit(100).Find(&users)

	type UserItem struct {
		ID          uint      `json:"id"`
		Username    string    `json:"username"`
		Phone       string    `json:"phone"`
		Email       string    `json:"email"`
		Role        string    `json:"role"`
		Level       int       `json:"level"`
		Balance     float64   `json:"balance"`
		TotalEarned float64   `json:"total_earned"`
		Status      int       `json:"status"`
		CreatedAt   string    `json:"created_at"`
	}

	items := make([]UserItem, len(users))
	for i, u := range users {
		items[i] = UserItem{
			ID:          u.ID,
			Username:    u.Username,
			Phone:       u.Phone,
			Email:       u.Email,
			Role:        u.Role,
			Level:       u.Level,
			Balance:     float64(u.Balance) / 100,
			TotalEarned: float64(u.TotalEarned) / 100,
			Status:      u.Status,
			CreatedAt:   u.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, gin.H{"total": len(items), "users": items})
}

// ToggleUserStatus 管理员切换用户状态（禁用/启用）
func ToggleUserStatus(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	userID := c.Param("id")

	// 检查管理员权限
	var currentUser model.User
	if err := database.DB.First(&currentUser, adminID).Error; err != nil || currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 不能操作管理员自己
	if user.ID == uint(adminID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能操作自己的账户"})
		return
	}

	// 切换状态
	newStatus := 1
	if user.Status == 1 {
		newStatus = 0
	}

	if err := database.DB.Model(&user).Update("status", newStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
		return
	}

	logger.Info("User status toggled: id=%d status=%d by admin=%d", user.ID, newStatus, adminID)
	c.JSON(http.StatusOK, gin.H{"message": "操作成功", "user_id": user.ID, "status": newStatus})
}

// ListSuppliers 管理员查看所有供应商
func ListSuppliers(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	// 检查管理员权限
	var currentUser model.User
	if err := database.DB.First(&currentUser, adminID).Error; err != nil || currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	var suppliers []model.Supplier
	database.DB.Find(&suppliers)

	type SupplierItem struct {
		ID        uint      `json:"id"`
		Name      string    `json:"name"`
		Code      string    `json:"code"`
		Contact   string    `json:"contact"`
		Phone     string    `json:"phone"`
		Status    int       `json:"status"`
		CreatedAt string    `json:"created_at"`
	}

	items := make([]SupplierItem, len(suppliers))
	for i, s := range suppliers {
		items[i] = SupplierItem{
			ID:        s.ID,
			Name:      s.Name,
			Code:      s.Code,
			Contact:   s.Contact,
			Phone:     s.Phone,
			Status:    s.Status,
			CreatedAt: s.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, gin.H{"total": len(items), "suppliers": items})
}

// ListProducts 管理员查看所有商品
func ListProducts(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	// 检查管理员权限
	var currentUser model.User
	if err := database.DB.First(&currentUser, adminID).Error; err != nil || currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	var products []model.Product
	database.DB.Preload("Supplier").Find(&products)

	type ProductItem struct {
		ID           uint      `json:"id"`
		SupplierName string    `json:"supplier_name"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		Price        float64   `json:"price"`
		ImageURL     string    `json:"image_url"`
		Status       int       `json:"status"`
		CreatedAt    string    `json:"created_at"`
	}

	items := make([]ProductItem, len(products))
	for i, p := range products {
		var supplierName string
		if p.Supplier.ID > 0 {
			supplierName = p.Supplier.Name
		}
		items[i] = ProductItem{
			ID:           p.ID,
			SupplierName: supplierName,
			Name:         p.Name,
			Description:  p.Description,
			Price:        float64(p.Price) / 100,
			ImageURL:     p.ImageURL,
			Status:       p.Status,
			CreatedAt:    p.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, gin.H{"total": len(items), "products": items})
}

// AdminListOrders 管理员查看所有订单
func AdminListOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	type OrderSummary struct {
		ID        uint    `json:"id"`
		OrderNo   string  `json:"order_no"`
		Username  string  `json:"username"`
		ProductName string `json:"product_name"`
		Amount    float64 `json:"amount"`
		Status    string  `json:"status"`
		CreatedAt string  `json:"created_at"`
	}

	var orders []model.Order
	var total int64

	db := database.DB

	db.Model(&model.Order{}).Count(&total)
	db.Preload("Product").Preload("User").Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders)

	items := make([]OrderSummary, len(orders))
	for i, o := range orders {
		productName := ""
		if o.Product.ID > 0 {
			productName = o.Product.Name
		}
		
		items[i] = OrderSummary{
			ID:          o.ID,
			OrderNo:     o.OrderNo,
			Username:    o.User.Username,
			ProductName: productName,
			Amount:      float64(o.Amount) / 100,
			Status:      o.Status,
			CreatedAt:   o.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"page":    page,
		"page_size": pageSize,
		"orders":  items,
	})
}

// AdminRecharge 管理员给用户充值
func AdminRecharge(c *gin.Context) {
	type RechargeReq struct {
		UserID   uint   `json:"user_id" binding:"required"`
		Amount   int64  `json:"amount" binding:"required,min=1"`
		Remark   string `json:"remark"`
	}

	var req RechargeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.DB

	var user model.User
	if err := db.First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 更新余额
	err := db.Model(&user).Update("balance", gorm.Expr("balance + ?", req.Amount)).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Recharge successful",
		"user_id":  user.ID,
		"username": user.Username,
		"amount":   float64(req.Amount) / 100,
		"balance":  float64(user.Balance) / 100,
		"remark":   req.Remark,
	})
}

// UserRecharge 用户端充值（模拟充值成功）
func UserRecharge(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUserName(c)

	type RechargeReq struct {
		Amount      int64  `json:"amount" binding:"required,min=100"` // 最低1元
		PaymentMode string `json:"payment_mode" binding:"oneof=wechat alipay balance"`
	}

	var req RechargeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.DB

	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 更新余额
	updateErr := db.Model(&user).Update("balance", gorm.Expr("balance + ?", req.Amount)).Error
	if updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "充值失败"})
		return
	}

	// 重新加载用户数据以获取最新余额
	db.First(&user, userID)

	logger.Info("User recharge: user=%s(%d) amount=%.2f payment_mode=%s",
		username, userID, float64(req.Amount)/100, req.PaymentMode)

	c.JSON(http.StatusOK, gin.H{
		"message":       "充值成功",
		"user_id":       user.ID,
		"username":      user.Username,
		"amount":        float64(req.Amount) / 100,
		"balance":       float64(user.Balance) / 100,
		"payment_mode":  req.PaymentMode,
	})
}
