package dto

import (
	"time"
)

// ========== 认证相关 DTO ==========

type RegisterReq struct {
	Username   string `json:"username" binding:"required,min=3,max=32"`
	Password   string `json:"password" binding:"required,min=6,max=32"`
	Phone      string `json:"phone" binding:"omitempty,max=20"`
	Email      string `json:"email" binding:"omitempty,email"`
	InviteCode string `json:"invite_code" binding:"omitempty,max=32"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserInfo struct {
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	Level      int    `json:"level"`
	Balance    int64  `json:"balance"`
	TotalEarned int64 `json:"total_earned"`
	ParentID   *uint  `json:"parent_id,omitempty"`
	InviteCode string `json:"invite_code"`
	CreatedAt  string `json:"created_at"`
}

// ========== 提现相关 DTO ==========

type WithdrawApplyReq struct {
	Amount      int64  `json:"amount" binding:"required,min=10000"` // 最低100元(10000分)
	BankName    string `json:"bank_name" binding:"required"`
	AccountName string `json:"account_name" binding:"required"`
	AccountNo   string `json:"account_no" binding:"required"`
}

type WithdrawApproveReq struct {
	Action string `json:"action" binding:"required,oneof=approve reject"`
	Remark string `json:"remark"`
}

// ========== 排行榜相关 DTO ==========

type LeaderboardItem struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Ranking   int    `json:"ranking"`
	RankValue int64  `json:"rank_value"` // 排名值(分)
}

// ========== 订单相关 DTO ==========

type OrderInfo struct {
	ID            uint    `json:"id"`
	OrderNo       string  `json:"order_no"`
	ProductID     uint    `json:"product_id"`
	ProductName   string  `json:"product_name"`
	ProductImage  string  `json:"product_image"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
	Status        string  `json:"status"`
	PaidAt        *string `json:"paid_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type CreateOrderReq struct {
	ProductID     uint   `json:"product_id" binding:"required"`
	PaymentMethod string `json:"payment_method" binding:"required"`
}

// ========== 分润相关 DTO ==========

type ProfitInfo struct {
	ID          uint    `json:"id"`
	OrderNo     string  `json:"order_no"`
	FromUser    string  `json:"from_user"`
	Level       int     `json:"level"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Description string  `json:"description"`
	CreatedAt   string  `json:"created_at"`
}

// ========== 管理员统计 DTO ==========

type AdminStatsResp struct {
	TotalUsers            int64           `json:"total_users"`
	ActiveUsers           int64           `json:"active_users"`
	TotalOrders           int64           `json:"total_orders"`
	TotalRevenue          float64         `json:"total_revenue"`
	TotalProfit           float64         `json:"total_profit"`
	TodayOrders           int64           `json:"today_orders"`
	TodayRevenue          float64         `json:"today_revenue"`
	PendingWithdraw       int64           `json:"pending_withdraw"`
	PendingWithdrawAmount float64         `json:"pending_withdraw_amount"`
	TopUsers              []LeaderboardItem `json:"top_users"`
}

// ========== Leaderboard/Team response types ==========

type UserWithBalance struct {
	Balance int64 `gorm:"column:balance"`
}

type TimeVal struct {
	Time time.Time
}
