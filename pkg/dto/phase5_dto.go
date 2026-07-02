// Package dto Phase 5 DTO定义
package dto

import "time"

// ============================================================
// 第三方支付相关 DTO
// ============================================================

// CreatePaymentReq 创建支付请求
type CreatePaymentReq struct {
	OrderID   uint   `json:"order_id" binding:"required"`
	Channel   string `json:"channel" binding:"required,oneof=wechat alipay"`
	Amount    int64  `json:"amount" binding:"required,min=1"`
	Subject   string `json:"subject" binding:"required,max=200"`
	NotifyURL string `json:"notify_url" binding:"required"`
}

// PaymentResp 支付响应
type PaymentResp struct {
	PaymentNo  string    `json:"payment_no"`
	Channel    string    `json:"channel"`
	Amount     int64     `json:"amount"`
	PrepayID   string    `json:"prepay_id,omitempty"`
	ReturnMsg  string    `json:"return_msg,omitempty"` // 二维码/Base64串
	QRCodeURL  string    `json:"qr_code_url,omitempty"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// PaymentStatusReq 支付状态查询请求
type PaymentStatusReq struct {
	PaymentNo string `json:"payment_no" binding:"required"`
}

// PaymentStatusResp 支付状态响应
type PaymentStatusResp struct {
	PaymentNo string `json:"payment_no"`
	Channel   string `json:"channel"`
	Amount    int64  `json:"amount"`
	Status    string `json:"status"` // processing, success, failed, refunded
	PaidAt    *time.Time `json:"paid_at,omitempty"`
}

// RefundRequest 退款申请
type RefundRequest struct {
	PaymentNo string `json:"payment_no" binding:"required"`
	Reason    string `json:"reason" binding:"required,max=255"`
}

// PaymentRecordResp 支付记录响应
type PaymentRecordResp struct {
	ID          uint      `json:"id"`
	PaymentNo   string    `json:"payment_no"`
	Channel     string    `json:"channel"`
	Amount      int64     `json:"amount"`
	Fee         int64     `json:"fee"`
	RealAmount  int64     `json:"real_amount"`
	Status      string    `json:"status"`
	PrepayID    string    `json:"prepay_id,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	RefundedAt  *time.Time `json:"refunded_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ============================================================
// 灵活用工相关 DTO
// ============================================================

// FreelancerRegisterReq 自由职业者注册请求
type FreelancerRegisterReq struct {
	RealName  string   `json:"real_name" binding:"required,max=50"`
	IDCard    string   `json:"id_card" binding:"required,max=18"`
	Phone     string   `json:"phone" binding:"max=20"`
	Email     string   `json:"email" binding:"omitempty,email,max=100"`
	SkillTags []string `json:"skill_tags" binding:"required,min=1"`
	Bio       string   `json:"bio" binding:"max=1000"`
}

// FreelancerProfileResp 自由职业者个人资料响应
type FreelancerProfileResp struct {
	ID            uint      `json:"id"`
	RealName      string    `json:"real_name"`
	SkillTags     []string  `json:"skill_tags"`
	Bio           string    `json:"bio"`
	AvgRating     float64   `json:"avg_rating"`
	TotalJobs     int       `json:"total_jobs"`
	TotalEarnings int64     `json:"total_earnings"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// TaskCreateReq 创建任务请求
type TaskCreateReq struct {
	Title         string    `json:"title" binding:"required,max=200"`
	Description   string    `json:"description" binding:"required"`
	Category      string    `json:"category" binding:"required,oneof=design dev marketing writing"`
	SkillTags     []string  `json:"skill_tags"`
	Budget        int64     `json:"budget" binding:"required,min=100"`
	DurationHours int       `json:"duration_hours" binding:"min=1"`
	Deadline      *time.Time `json:"deadline"`
}

// TaskListReq 任务列表查询请求
type TaskListReq struct {
	Status     string `form:"status"`           // open, assigned, in_progress, completed
	Category   string `form:"category"`
	MinBudget  int64  `form:"min_budget"`
	MaxBudget  int64  `form:"max_budget"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=20"`
}

// TaskDetailResp 任务详情响应
type TaskDetailResp struct {
	ID            uint      `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	SkillTags     []string  `json:"skill_tags"`
	Budget        int64     `json:"budget"`
	DurationHours int       `json:"duration_hours"`
	PublisherID   uint      `json:"publisher_id"`
	PublisherName string    `json:"publisher_name"`
	Status        string    `json:"status"`
	Deadline      *time.Time `json:"deadline"`
	AssignedTo    uint      `json:"assigned_to,omitempty"`
	Submission    string    `json:"submission,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	ReviewComment string    `json:"review_comment,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// TaskAssignReq 任务分配请求
type TaskAssignReq struct {
	TaskID       uint `json:"task_id" binding:"required"`
	FreelancerID uint `json:"freelancer_id" binding:"required"`
}

// TimeLogCreateReq 创建工时记录请求
type TimeLogCreateReq struct {
	TaskID     uint    `json:"task_id" binding:"required"`
	Date       string  `json:"date" binding:"required"`
	Hours      float64 `json:"hours" binding:"required,min=0.5,max=24"`
	Content    string  `json:"content" binding:"required"`
	Screenshot string  `json:"screenshot_url"`
}

// SettlementCreateReq 创建结算记录请求
type SettlementCreateReq struct {
	TaskID       uint   `json:"task_id" binding:"required"`
	BankName     string `json:"bank_name"`
	AccountName  string `json:"account_name" binding:"required"`
	AccountNo    string `json:"account_no" binding:"required"`
	PaymentMethod string `json:"payment_method" binding:"oneof=bank_transfer alipay wechat"`
}

// RatingCreateReq 创建评分请求
type RatingCreateReq struct {
	TaskID       uint   `json:"task_id" binding:"required"`
	FreelancerID uint   `json:"freelancer_id" binding:"required"`
	Score        int    `json:"score" binding:"required,min=1,max=5"`
	Comment      string `json:"comment" binding:"max=500"`
}

// RatingItemResp 评分项响应
type RatingItemResp struct {
	ID           uint      `json:"id"`
	TaskID       uint      `json:"task_id"`
	FreelancerID uint      `json:"freelancer_id"`
	PublisherID  uint      `json:"publisher_id"`
	Score        int       `json:"score"`
	Comment      string    `json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
}

// RatingListReq 评分列表查询请求
type RatingListReq struct {
	FreelancerID uint `form:"freelancer_id"`
	PublisherID  uint `form:"publisher_id"`
	TaskID       uint `form:"task_id"`
	Page         int  `form:"page,default=1"`
	PageSize     int  `form:"page_size,default=20"`
}

// RatingStatsResp 评分统计响应
type RatingStatsResp struct {
	FreelancerID uint    `json:"freelancer_id"`
	AvgRating    float64 `json:"avg_rating"`     // 平均评分
	TotalRatings int     `json:"total_ratings"`  // 总评分数
	FiveStar     int     `json:"five_star"`      // 5星数量
	FourStar     int     `json:"four_star"`      // 4星数量
	ThreeStar    int     `json:"three_star"`     // 3星数量
	TwoStar      int     `json:"two_star"`       // 2星数量
	OneStar      int     `json:"one_star"`       // 1星数量
}
