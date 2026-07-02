// Package model 数据模型定义 (Phase 5 扩展: 第三方支付 + 灵活用工)
package model

import (
	"time"

	"gorm.io/gorm"
)

// ============================================================
// 基础模型
// ============================================================

// BaseModel 基础模型
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// ============================================================
// 第三方扩展模型 (Phase 5)
// ============================================================

// ThirdPartyPayment 第三方支付记录表
type ThirdPartyPayment struct {
	BaseModel
	UserID        uint   `gorm:"index" json:"user_id"`
	User          User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	OrderID       uint   `gorm:"index" json:"order_id"`
	Order         Order  `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	PaymentNo     string `gorm:"type:varchar(64);uniqueIndex" json:"payment_no"` // 支付平台流水号
	Channel       string `gorm:"type:varchar(20)" json:"channel"`               // wechat, alipay
	Amount        int64  `gorm:"not null" json:"amount"`                        // 支付金额(分)
	Fee           int64  `gorm:"default:0" json:"fee"`                          // 手续费(分)
	RealAmount    int64  `gorm:"default:0" json:"real_amount"`                  // 实际到账(分)
	Status        string `gorm:"type:varchar(20);default:processing" json:"status"` // processing, success, failed, refunded
	PrepayID      string `gorm:"type:varchar(128)" json:"prepay_id"`            // 预支付ID(微信/jsapi下单)
	ReturnMsg     string `gorm:"type:text" json:"return_msg"`                   // 返回消息(二维码等)
	PaidAt        *time.Time `json:"paid_at"`                                   // 支付成功时间
	RefundedAt    *time.Time `json:"refunded_at"`                                 // 退款时间
	RefundAmount  int64  `gorm:"default:0" json:"refund_amount"`                // 退款金额(分)
	CallbackData  string `gorm:"type:text" json:"callback_data"`                // 回调原始数据
}

// SupplierInvoice 供应商发票表
type SupplierInvoice struct {
	BaseModel
	ThirdPartyPaymentID uint   `gorm:"index" json:"third_party_payment_id"`
	Payment             ThirdPartyPayment `gorm:"foreignKey:ThirdPartyPaymentID" json:"payment,omitempty"`
	Amount      int64  `gorm:"not null" json:"amount"`        // 发票金额(分)
	Type        string `gorm:"type:varchar(20)" json:"type"`  // VAT普通发票, VAT专用发票
	Status      string `gorm:"type:varchar(20);default:pending" json:"status"` // pending, issued, rejected
	IssueNo     string `gorm:"type:varchar(64)" json:"issue_no"` // 发票编号
	IssuedAt    *time.Time `json:"issued_at"`                // 开票时间
}

// Freelancer 自由职业者表
type Freelancer struct {
	BaseModel
	UserID       uint   `gorm:"index" json:"user_id"`
	User         User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	RealName     string `gorm:"type:varchar(50)" json:"real_name"`   // 真实姓名
	IDCard       string `gorm:"type:varchar(18);uniqueIndex" json:"id_card"` // 身份证号
	Phone        string `gorm:"type:varchar(20)" json:"phone"`
	Email        string `gorm:"type:varchar(100)" json:"email"`
	SkillTags    string `gorm:"type:text" json:"skill_tags"`         // 技能标签(JSON数组)
	Bio          string `gorm:"type:text" json:"bio"`                // 个人简介
	AvgRating    float64 `gorm:"default:0" json:"avg_rating"`        // 平均评分
	TotalJobs    int    `gorm:"default:0" json:"total_jobs"`         // 完成任务数
	TotalEarnings int64 `gorm:"default:0" json:"total_earnings"`     // 累计收益(分)
	Status       string `gorm:"type:varchar(20);default:pending" json:"status"` // pending, approved, rejected, disabled
	ApprovedBy   *uint  `json:"approved_by"`                         // 审核人
	ApprovedAt   *time.Time `json:"approved_at"`                     // 审核时间
	RejectReason string `gorm:"type:varchar(255)" json:"reject_reason"`
}

// Task 灵活用工任务表
type Task struct {
	BaseModel
	Title         string `gorm:"type:varchar(200);not null" json:"title"`
	Description   string `gorm:"type:text" json:"description"`
	Category      string `gorm:"type:varchar(50)" json:"category"` // design, dev, marketing, writing
	SkillTags     string `gorm:"type:text" json:"skill_tags"`       // 所需技能标签
	Budget        int64  `gorm:"not null" json:"budget"`            // 预算(分)
	DurationHours int    `gorm:"default:24" json:"duration_hours"`  // 预计工时(小时)
	PublisherID   uint   `gorm:"index" json:"publisher_id"`         // 发布者(企业用户ID)
	Publisher     User   `gorm:"foreignKey:PublisherID" json:"publisher,omitempty"`
	Status        string `gorm:"type:varchar(20);default:open" json:"status"` // open, assigned, in_progress, submitted, reviewed, completed, cancelled
	Deadline      *time.Time `json:"deadline"`                      // 截止时间
	AssignedTo    *uint  `json:"assigned_to"`                   // 分配的自由职业者ID
	AssignedFreelancer *Freelancer `gorm:"foreignKey:AssignedTo" json:"freelancer,omitempty"`
	Submission    string `gorm:"type:text" json:"submission"`     // 提交成果描述
	ReviewerID    *uint  `json:"reviewer_id"`                    // 审核人ID
	ReviewedAt    *time.Time `json:"reviewed_at"`               // 审核时间
	ReviewComment string `gorm:"type:text" json:"review_comment"` // 审核意见
	PaidAt        *time.Time `json:"paid_at"`                      // 结算时间
}

// TimeLog 工时记录表
type TimeLog struct {
	BaseModel
	TaskID    uint   `gorm:"index" json:"task_id"`
	Task      Task   `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	FreelancerID uint `gorm:"index" json:"freelancer_id"`
	Freelancer Freelancer `gorm:"foreignKey:FreelancerID" json:"freelancer,omitempty"`
	Date      string `gorm:"type:date" json:"date"`          // 工作日期
	Hours     float64 `gorm:"not null" json:"hours"`          // 工作时长
	Content   string `gorm:"type:text" json:"content"`        // 工作内容描述
	Screenshot string `gorm:"type:text" json:"screenshot_url"` // 截图URL
	Status    string `gorm:"type:varchar(20);default:pending" json:"status"` // pending, approved, rejected
	ApprovedBy *uint `json:"approved_by"`                    // 审核人
	ApprovedAt *time.Time `json:"approved_at"`               // 审核时间
	RejectReason string `gorm:"type:varchar(255)" json:"reject_reason"`
}

// Settlement 薪资结算表
type Settlement struct {
	BaseModel
	TaskID         uint   `gorm:"index" json:"task_id"`
	Task           Task   `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	FreelancerID   uint   `gorm:"index" json:"freelancer_id"`
	Freelancer     Freelancer `gorm:"foreignKey:FreelancerID" json:"freelancer,omitempty"`
	Amount         int64  `gorm:"not null" json:"amount"`         // 结算金额(分)
	TaxAmount      int64  `gorm:"default:0" json:"tax_amount"`    // 代扣税费
	PlatformFee    int64  `gorm:"default:0" json:"platform_fee"`  // 平台服务费
	NetAmount      int64  `gorm:"not null" json:"net_amount"`     // 净收益(分)
	PaymentMethod  string `gorm:"type:varchar(20)" json:"payment_method"` // bank_transfer, alipay, wechat
	BankName       string `gorm:"type:varchar(100)" json:"bank_name"`
	AccountName    string `gorm:"type:varchar(100)" json:"account_name"`
	AccountNo      string `gorm:"type:varchar(50)" json:"account_no"`
	Status         string `gorm:"type:varchar(20);default:pending" json:"status"` // pending, approved, paid, failed
	Remark         string `gorm:"type:varchar(255)" json:"remark"`
	PaidAt         *time.Time `json:"paid_at"`
	ApprovedBy     *uint  `json:"approved_by"`
	ApprovedAt     *time.Time `json:"approved_at"`
}

// Rating 评分表
type Rating struct {
	BaseModel
	TaskID        uint  `gorm:"index" json:"task_id"`
	Task          Task  `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	FreelancerID  uint  `gorm:"index" json:"freelancer_id"`
	Freelancer    Freelancer `gorm:"foreignKey:FreelancerID" json:"freelancer,omitempty"`
	PublisherID   uint  `json:"publisher_id"`
	Score         int   `gorm:"not null" json:"score"` // 1-5
	Comment       string `gorm:"type:text" json:"comment"`
}

// ============================================================
// 原有模型 (Phase 1-4)
// ============================================================

// User 用户表
type User struct {
	BaseModel
	Username    string `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Password    string `gorm:"type:varchar(255);not null" json:"-"`
	Phone       string `gorm:"type:varchar(20)" json:"phone"`
	Email       string `gorm:"type:varchar(100)" json:"email"`
	Role        string `gorm:"type:varchar(20);default:customer" json:"role"` // admin, supplier, customer, freelancer
	InviteCode  string `gorm:"type:varchar(32);uniqueIndex" json:"invite_code"`
	ParentID    *uint    `gorm:"index" json:"parent_id"`
	Parent      *User    `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Level       int    `gorm:"default:1" json:"level"` // 用户等级 1-5
	Status      int    `gorm:"default:1" json:"status"` // 1:active 0:disable
	Balance     int64  `gorm:"default:0" json:"balance"` // 余额(分)
	TotalEarned int64  `gorm:"default:0" json:"total_earned"` // 累计收益
}

// Supplier 供应商表
type Supplier struct {
	BaseModel
	Name       string `gorm:"type:varchar(100);not null" json:"name"`
	Code       string `gorm:"type:varchar(50);uniqueIndex" json:"code"`
	OwnerID    uint   `gorm:"index" json:"owner_id"` // 负责人(用户ID)
	Contact    string `gorm:"type:varchar(50)" json:"contact"`
	Phone      string `gorm:"type:varchar(20)" json:"phone"`
	Address    string `gorm:"type:text" json:"address"`
	BankName   string `gorm:"type:varchar(100)" json:"bank_name"`
	BankAccount string `gorm:"type:varchar(50)" json:"bank_account"`
	Status     int    `gorm:"default:1" json:"status"` // 1:active 0:disable
}

// Product 商品表
type Product struct {
	BaseModel
	SupplierID  uint   `gorm:"index" json:"supplier_id"`
	Supplier    Supplier `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
	Name        string `gorm:"type:varchar(200);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Price       int64  `gorm:"not null" json:"price"` // 价格(分)
	ImageURL    string `gorm:"type:varchar(500)" json:"image_url"`
	Status      int    `gorm:"default:1" json:"status"` // 1:on_sale 0:off_sale
}

// Order 订单表
type Order struct {
	BaseModel
	UserID      uint   `gorm:"index" json:"user_id"`
	User        User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ProductID   uint   `gorm:"index" json:"product_id"`
	Product     Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	OrderNo     string `gorm:"type:varchar(32);uniqueIndex" json:"order_no"`
	Amount      int64  `gorm:"not null" json:"amount"` // 订单金额(分)
	PaymentMethod string `gorm:"type:varchar(20)" json:"payment_method"` // wechat, alipay
	Status      string `gorm:"type:varchar(20);default:pending" json:"status"` // pending, paid, completed, cancelled
	PaidAt      *time.Time `json:"paid_at"`
	PaymentNo   string `gorm:"type:varchar(64)" json:"payment_no"` // 支付平台订单号
}

// ProfitShare 分润记录表
type ProfitShare struct {
	BaseModel
	OrderID      uint   `gorm:"index" json:"order_id"`
	Order        Order  `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	FromUserID   uint   `gorm:"index" json:"from_user_id"` // 付款用户
	ToUserID     uint   `gorm:"index" json:"to_user_id"`   // 收款用户
	Level        int    `gorm:"not null" json:"level"`     // 分润层级 1-5
	Amount       int64  `gorm:"not null" json:"amount"`    // 分润金额(分)
	Type         string `gorm:"type:varchar(20)" json:"type"` // direct, team
	Status       string `gorm:"type:varchar(20);default:pending" json:"status"` // pending, settled, withdrawn
	SettledAt    *time.Time `json:"settled_at"`
	Description  string `gorm:"type:varchar(255)" json:"description"`
}

// ChainRecord 链动记录表
type ChainRecord struct {
	BaseModel
	UserID    uint `gorm:"index" json:"user_id"`
	Action    string `gorm:"type:varchar(20)" json:"action"` // bind, unlock, earn
	RelatedID uint `gorm:"index" json:"related_id"` // 关联用户ID
	OrderID   uint `gorm:"index" json:"order_id"`
	Status    string `gorm:"type:varchar(20)" json:"status"`
	Data      string `gorm:"type:json" json:"data"` // 额外数据(JSON)
}

// Withdraw 提现申请表
type Withdraw struct {
	BaseModel
	UserID      uint   `gorm:"index" json:"user_id"`
	User        User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Amount      int64  `gorm:"not null" json:"amount"` // 提现金额(分)
	Fee         int64  `gorm:"default:0" json:"fee"`   // 手续费(分)
	ActualAmount int64 `gorm:"default:0" json:"actual_amount"` // 实际到账(分)
	BankName    string `gorm:"type:varchar(100)" json:"bank_name"` // 收款银行
	AccountName string `gorm:"type:varchar(100)" json:"account_name"` // 收款账户
	AccountNo   string `gorm:"type:varchar(50)" json:"account_no"` // 账号
	Status      string `gorm:"type:varchar(20);default:pending" json:"status"` // pending, approved, rejected, paid
	Remark      string `gorm:"type:varchar(255)" json:"remark"` // 备注/拒付原因
	ApprovedBy  *uint  `json:"approved_by"` // 审核人ID
	ApprovedAt  *time.Time `json:"approved_at"`
}

// LeaderboardRank 排行榜排名
type LeaderboardRank struct {
	BaseModel
	UserID    uint   `gorm:"uniqueIndex:idx_userleaderboard_type" json:"user_id"`
	User      User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Type      string `gorm:"type:varchar(20);index" json:"type"` // total_earned, team_size, recharge
	RankValue int64  `gorm:"default:0" json:"rank_value"` // 排名值
	Ranking   int    `gorm:"default:0" json:"ranking"`    // 当前排名
}

