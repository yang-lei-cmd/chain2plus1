// Package handler Phase 5: 第三方支付 Handler
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linqi/chain2plus1/internal/config"
	"github.com/linqi/chain2plus1/internal/service"
	"github.com/linqi/chain2plus1/pkg/dto"
)

// PaymentHandler 支付处理器
type PaymentHandler struct {
	paymentService *service.PaymentService
}

// NewPaymentHandler 创建支付处理器
func NewPaymentHandler(cfg *config.Config) *PaymentHandler {
	return &PaymentHandler{
		paymentService: service.NewPaymentService(cfg),
	}
}

// CreatePayment 创建支付
// POST /api/v1/payment/create
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req dto.CreatePaymentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paymentResp, err := h.paymentService.CreatePayment(c, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "支付创建成功",
		"payment":  paymentResp,
	})
}

// HandleWechatCallback 微信支付回调
// POST /api/v1/payment/wechat/notify
func (h *PaymentHandler) HandleWechatCallback(c *gin.Context) {
	h.paymentService.HandleWechatCallback(c)
}

// HandleAlipayCallback 支付宝回调
// POST /api/v1/payment/alipay/notify
func (h *PaymentHandler) HandleAlipayCallback(c *gin.Context) {
	h.paymentService.HandleAlipayCallback(c)
}

// QueryPaymentStatus 查询支付状态
// GET /api/v1/payment/status/:payment_no
func (h *PaymentHandler) QueryPaymentStatus(c *gin.Context) {
	paymentNo := c.Param("payment_no")

	resp, err := h.paymentService.QueryPaymentStatus(paymentNo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment": resp,
	})
}

// ProcessRefund 申请退款
// POST /api/v1/payment/refund
func (h *PaymentHandler) ProcessRefund(c *gin.Context) {
	var req dto.RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.paymentService.RefundPayment(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "退款申请已提交",
	})
}

// GetUserPayments 获取用户支付记录
// GET /api/v1/payment/my-payments
func (h *PaymentHandler) GetUserPayments(c *gin.Context) {
	userID := c.GetUint("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	payments, total, err := h.paymentService.GetPaymentRecords(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"payments": payments,
		"pagination": gin.H{
			"total":  total,
			"page":   page,
			"pages":  (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// ReconcilePayments 对账（管理后台）
// GET /admin/payment/reconcile?date=2026-07-02
func (h *PaymentHandler) ReconcilePayments(c *gin.Context) {
	dateStr := c.DefaultQuery("date", "")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日期格式，请使用 YYYY-MM-DD"})
		return
	}

	issues, err := h.paymentService.ReconcilePayments(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"date":    dateStr,
		"total":   len(issues),
		"issues":  issues,
		"message": "对账完成",
	})
}
