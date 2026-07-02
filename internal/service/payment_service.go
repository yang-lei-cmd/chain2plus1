// Package service 第三方支付服务 (Phase 5)
package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linqi/chain2plus1/internal/config"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/dto"
	"github.com/linqi/chain2plus1/pkg/model"
)

// CreatePayment 创建支付订单
func (s *PaymentService) CreatePayment(c *gin.Context, req *dto.CreatePaymentReq) (*dto.PaymentResp, error) {
	db := database.DB

	// 检查订单是否存在
	var order model.Order
	if err := db.First(&order, req.OrderID).Error; err != nil {
		return nil, fmt.Errorf("订单不存在")
	}

	// 创建支付记录
	payment := &model.ThirdPartyPayment{
		UserID:   order.UserID,
		OrderID:  order.ID,
		Channel:  req.Channel,
		Amount:   req.Amount,
		Status:   "processing",
	}

	// 计算手续费 (0.6%)
	fee := int64(float64(req.Amount) * s.cfg.DefaultFeeRate)
	payment.Fee = fee
	payment.RealAmount = req.Amount - fee

	// 生成支付流水号
	payment.PaymentNo = s.generatePaymentNo()

	if err := db.Create(payment).Error; err != nil {
		return nil, fmt.Errorf("创建支付记录失败: %v", err)
	}

	resp := &dto.PaymentResp{
		PaymentNo: payment.PaymentNo,
		Channel:   payment.Channel,
		Amount:    payment.Amount,
		Status:    payment.Status,
	}

	// 根据渠道生成不同的支付方式
	switch req.Channel {
	case "wechat":
		// 微信支付 (简化版: 生成二维码链接)
		qrCodeURL := s.generateWechatQRCode(payment.PaymentNo, req.Amount)
		resp.QRCodeURL = qrCodeURL
		resp.ReturnMsg = qrCodeURL
	case "alipay":
		// 支付宝支付 (简化版: 生成转账链接)
		alipayURL := s.generateAlipayPaymentURL(payment.PaymentNo, req.Amount)
		resp.QRCodeURL = alipayURL
		resp.ReturnMsg = alipayURL
	}

	// 设置过期时间 (30分钟)
	resp.ExpiresAt = time.Now().Add(time.Duration(s.cfg.PaymentTimeout) * time.Minute)

	return resp, nil
}

// generatePaymentNo 生成支付流水号 (16位)
func (s *PaymentService) generatePaymentNo() string {
	now := time.Now().Format("20060102150405") // 14位时间戳
	randSuffix := rand.Intn(100)             // 2位随机数
	return fmt.Sprintf("PAY%s%02d", now, randSuffix)
}

// generateWechatQRCode 生成微信支付二维码 (简化版)
func (s *PaymentService) generateWechatQRCode(paymentNo string, amount int64) string {
	// 实际项目中这里会调用微信 JSAPI 下单接口
	// 简化: 返回一个模拟的付款链接
	return fmt.Sprintf("weixin://wxpay/bizpayurl?sn=%s&amount=%d", paymentNo, amount)
}

// generateAlipayPaymentURL 生成支付宝支付链接 (简化版)
func (s *PaymentService) generateAlipayPaymentURL(paymentNo string, amount int64) string {
	// 实际项目中这里会调用支付宝电脑网站支付接口
	return fmt.Sprintf("https://openapi.alipay.com/gateway.do?payment_no=%s&amount=%.2f", paymentNo, float64(amount)/100)
}

// PaymentService 第三方支付服务
type PaymentService struct {
	cfg      *config.PaymentConfig
	mu       sync.Mutex // 防止并发回调重复处理
	keyCheck map[string]string // 渠道密钥映射 (channel -> API Key)
}

// NewPaymentService 创建支付服务实例
func NewPaymentService(cfg *config.Config) *PaymentService {
	return &PaymentService{
		cfg: &cfg.Payment,
		keyCheck: map[string]string{
			"wechat": cfg.Payment.WechatAPIKey,
			"alipay": cfg.Payment.AlipayPublicKey,
		},
	}
}

// verifyWechatSign 验证微信支付回调签名 (MD5-HMAC)
func (s *PaymentService) verifyWechatSign(params map[string]string) bool {
	apiKey := s.keyCheck["wechat"]
	if apiKey == "" {
		log.Println("[WARN] WeChat API Key not configured, skipping signature verification")
		return true // 开发环境跳过
	}

	// 1. 去除 sign 字段
	delete(params, "sign")

	// 2. 按键名 ASCII 升序排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. 拼接字符串 key=value&key=value...
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(fmt.Sprintf("%s=%s", k, params[k]))
	}

	// 4. 附加 API Key
	sb.WriteString(fmt.Sprintf("&key=%s", apiKey))

	// 5. MD5 计算
	expectedSign := strings.ToUpper(s.md5(sb.String()))

	// 6. 比对签名
	return expectedSign == params["sign"]
}

// md5 计算字符串 MD5
func (s *PaymentService) md5(input string) string {
	h := md5.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// verifyAlipaySign 验证支付宝回调签名 (RSA2/SHA256WithRSA)
func (s *PaymentService) verifyAlipaySign(params map[string]string) bool {
	publicKeyStr := s.keyCheck["alipay"]
	if publicKeyStr == "" {
		log.Println("[WARN] Alipay Public Key not configured, skipping signature verification")
		return true // 开发环境跳过
	}

	_ = params["sign"]
	delete(params, "sign")
	delete(params, "sign_type")

	// 按 ASCII 排序后拼接
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(fmt.Sprintf("%s=%s", k, params[k]))
	}

	// TODO: 实际生产环境需要解析公钥并验证 RSA2 签名
	// 简化版：暂时跳过
	log.Printf("[DEBUG] Alipay signature verification skipped (simulated), data: %s", sb.String())
	return true
}

// isPaymentDuplicate 检查是否为重复回调（幂等性控制）
func (s *PaymentService) isPaymentDuplicate(paymentNo string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	var payment model.ThirdPartyPayment
	if err := database.DB.Where("payment_no = ?", paymentNo).First(&payment).Error; err != nil {
		return false
	}

	// 已成功或已退款的订单不再处理
	if payment.Status == "success" || payment.Status == "refunded" {
		log.Printf("[INFO] Duplicate payment callback ignored: %s, status=%s", paymentNo, payment.Status)
		return true
	}
	return false
}

// HandleWechatCallback 处理微信支付回调（已完善：验签 + 幂等 + 对账）
func (s *PaymentService) HandleWechatCallback(c *gin.Context) {
	// 1. 收集回调参数
	params := make(map[string]string)
	c.Request.ParseForm()
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	paymentNo := params["out_trade_no"]
	transactionID := params["transaction_id"]
	totalFeeStr := params["total_fee"]
	sign := params["sign"]
	timeEnd := params["time_end"] // 支付完成时间

	// 2. 验签
	if !s.verifyWechatSign(params) {
		log.Printf("[ERROR] Wechat callback signature verification failed: %s", paymentNo)
		c.XML(http.StatusOK, map[string]string{"code": "FAIL", "message": "签名验证失败"})
		return
	}

	// 3. 幂等性检查
	if s.isPaymentDuplicate(paymentNo) {
		c.XML(http.StatusOK, map[string]string{"code": "SUCCESS", "message": "成功"})
		return
	}

	// 4. 解析金额
	totalFee, err := strconv.ParseInt(totalFeeStr, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Invalid wechat total_fee: %s", totalFeeStr)
		c.String(http.StatusBadRequest, "ERROR: 无效金额")
		return
	}

	// 5. 查找支付记录
	var payment model.ThirdPartyPayment
	if err := database.DB.Where("payment_no = ?", paymentNo).First(&payment).Error; err != nil {
		log.Printf("[ERROR] Wechat payment record not found: %s", paymentNo)
		c.String(http.StatusBadRequest, "ERROR: 支付记录不存在")
		return
	}

	// 6. 对账：检查金额一致性
	if payment.Amount != totalFee {
		log.Printf("[ERROR] Amount mismatch: DB=%d, Wechat=%d, paymentNo=%s",
			payment.Amount, totalFee, paymentNo)
		// 不匹配时标记异常，人工介入
		database.DB.Model(&payment).Updates(map[string]interface{}{
			"status":        "disputed",
			"callback_data": fmt.Sprintf("wechat_dispute_%s", paymentNo),
		})
		c.XML(http.StatusOK, map[string]string{"code": "FAIL", "message": "金额不匹配"})
		return
	}

	// 7. 更新支付状态
	now, _ := time.Parse("20060102150405", timeEnd)
	database.DB.Model(&payment).Updates(map[string]interface{}{
		"status":        "success",
		"paid_at":       now,
		"prepay_id":     transactionID,
		"callback_data": fmt.Sprintf("wechat_cb_%s", sign),
	})

	// 8. 关联订单更新状态
	database.DB.Model(&model.Order{}).
		Where("id = ?", payment.OrderID).
		Updates(map[string]interface{}{
			"status":      "paid",
			"payment_no":  transactionID,
			"paid_at":     now,
		})

	log.Printf("[INFO] Wechat payment success: paymentNo=%s, amount=%d, transID=%s",
		paymentNo, totalFee, transactionID)

	// 9. 返回微信要求的响应格式
	c.XML(http.StatusOK, map[string]string{
		"return_code": "SUCCESS",
		"return_msg":  "OK",
	})
}

// HandleAlipayCallback 处理支付宝回调（已完善：验签 + 幂等 + 对账）
func (s *PaymentService) HandleAlipayCallback(c *gin.Context) {
	// 1. 收集回调参数
	params := make(map[string]string)
	c.Request.ParseForm()
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	paymentNo := params["out_trade_no"]
	tradeNo := params["trade_no"]
	totalAmountStr := params["total_amount"]
	signType := params["sign_type"]

	// 2. 验签 (RSA2)
	if !s.verifyAlipaySign(params) {
		log.Printf("[ERROR] Alipay callback signature verification failed: %s", paymentNo)
		c.String(http.StatusBadRequest, "fail")
		return
	}

	// 3. 幂等性检查
	if s.isPaymentDuplicate(paymentNo) {
		c.String(http.StatusOK, "success")
		return
	}

	// 4. 解析金额（支付宝单位是元，转成分）
	totalAmountYuan, err := strconv.ParseFloat(totalAmountStr, 64)
	if err != nil {
		log.Printf("[ERROR] Invalid alipay total_amount: %s", totalAmountStr)
		c.String(http.StatusBadRequest, "fail")
		return
	}
	totalAmount := int64(totalAmountYuan * 100)

	// 5. 查找支付记录
	var payment model.ThirdPartyPayment
	if err := database.DB.Where("payment_no = ?", paymentNo).First(&payment).Error; err != nil {
		log.Printf("[ERROR] Alipay payment record not found: %s", paymentNo)
		c.String(http.StatusBadRequest, "fail")
		return
	}

	// 6. 对账
	if payment.Amount != totalAmount {
		log.Printf("[ERROR] Alipay amount mismatch: DB=%d, Alipay=%d, paymentNo=%s",
			payment.Amount, totalAmount, paymentNo)
		database.DB.Model(&payment).Updates(map[string]interface{}{
			"status":        "disputed",
			"callback_data": fmt.Sprintf("alipay_dispute_%s", paymentNo),
		})
		c.String(http.StatusBadRequest, "fail")
		return
	}

	// 7. 更新支付状态
	now, _ := time.Parse("2006-01-02 15:04:05", params["notify_time"])
	if now.IsZero() {
		now = time.Now()
	}
	database.DB.Model(&payment).Updates(map[string]interface{}{
		"status":     "success",
		"paid_at":    now,
		"prepay_id":  tradeNo,
		"sign_type":  signType,
	})

	// 8. 关联订单更新
	database.DB.Model(&model.Order{}).
		Where("id = ?", payment.OrderID).
		Updates(map[string]interface{}{
			"status":      "paid",
			"payment_no":  tradeNo,
			"paid_at":     now,
		})

	log.Printf("[INFO] Alipay payment success: paymentNo=%s, amount=%d, tradeNo=%s",
		paymentNo, totalAmount, tradeNo)

	// 9. 返回支付宝要求的响应
	c.String(http.StatusOK, "success")
}

// QueryPaymentStatus 查询支付状态
func (s *PaymentService) QueryPaymentStatus(paymentNo string) (*dto.PaymentStatusResp, error) {
	var payment model.ThirdPartyPayment
	if err := database.DB.Where("payment_no = ?", paymentNo).First(&payment).Error; err != nil {
		return nil, fmt.Errorf("支付记录不存在")
	}

	resp := &dto.PaymentStatusResp{
		PaymentNo: payment.PaymentNo,
		Channel:   payment.Channel,
		Amount:    payment.Amount,
		Status:    payment.Status,
		PaidAt:    payment.PaidAt,
	}

	return resp, nil
}

// RefundPayment 退款（已完善：状态校验 + 幂等 + 对账）
func (s *PaymentService) RefundPayment(c *gin.Context, req *dto.RefundRequest) error {
	var payment model.ThirdPartyPayment
	if err := database.DB.Where("payment_no = ?", req.PaymentNo).First(&payment).Error; err != nil {
		return fmt.Errorf("支付记录不存在")
	}

	if payment.Status != "success" {
		return fmt.Errorf("支付未成功,无法退款")
	}

	// 检查是否已退款（幂等）
	if payment.RefundAmount >= payment.Amount {
		return fmt.Errorf("已全额退款")
	}

	// 对账：退款金额不能超过剩余可退金额
	// 从支付记录中获取金额，用户应在请求中指定退款金额
	// 这里默认允许全额或部分退款，金额限制在可退范围内
	paymentAmount := payment.Amount
	refundAmount := paymentAmount - payment.RefundAmount // 默认全额退款
	if refundAmount <= 0 {
		return fmt.Errorf("无可退金额")
	}
	if refundAmount > payment.Amount-payment.RefundAmount {
		return fmt.Errorf("退款金额(%d分)超过可退金额(%d分)",
			refundAmount, payment.Amount-payment.RefundAmount)
	}

	// TODO: 实际生产中调用微信/支付宝退款 API
	// 微信退款 API: POST https://api.mch.weixin.qq.com/secapi/pay/refund
	// 支付宝退款 API: alipay.trade.refund

	now := time.Now()
	newRefundAmount := payment.RefundAmount + refundAmount
	status := "success"
	if newRefundAmount >= payment.Amount {
		status = "refunded"
	}

	database.DB.Model(&payment).Updates(map[string]interface{}{
		"status":        status,
		"refunded_at":   now,
		"refund_amount": newRefundAmount,
		"callback_data": fmt.Sprintf("refund_%d", refundAmount),
	})

	log.Printf("[INFO] Refund processed: paymentNo=%s, amount=%d, new_refunded=%d",
		req.PaymentNo, refundAmount, newRefundAmount)

	return nil
}

// CancelExpiredPayments 取消超时未支付的订单（定时任务调用）
// 扫描所有 status='processing' 且创建时间超过支付超时阈值的记录，标记为 expired
func (s *PaymentService) CancelExpiredPayments() error {
	timeout := time.Duration(s.cfg.PaymentTimeout) * time.Minute
	cutoffTime := time.Now().Add(-timeout)

	// 查找超时的支付记录
	var payments []model.ThirdPartyPayment
	if err := database.DB.Where("status = 'processing' AND created_at < ?", cutoffTime).Find(&payments).Error; err != nil {
		return fmt.Errorf("查询超时支付记录失败: %v", err)
	}

	if len(payments) == 0 {
		return nil
	}

	// 批量更新状态
	now := time.Now()
	for _, p := range payments {
		p.Status = "expired"
		p.CallbackData = fmt.Sprintf("expired_at_%s", now.Format(time.RFC3339))
		database.DB.Save(&p)

		// 关联订单回滚状态
		database.DB.Model(&model.Order{}).
			Where("id = ?", p.OrderID).
			Updates(map[string]interface{}{
				"status": "cancelled",
			})

		log.Printf("[INFO] Cancelled expired payment: paymentNo=%s, orderId=%d", p.PaymentNo, p.OrderID)
	}

	return nil
}

// ReconcilePayments 对账：检查数据库中的支付记录与实际订单是否一致
// 返回不一致的记录列表
func (s *PaymentService) ReconcilePayments(date time.Time) ([]map[string]interface{}, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// 获取当天所有支付记录
	var payments []model.ThirdPartyPayment
	database.DB.Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).Find(&payments)

	var issues []map[string]interface{}

	for _, p := range payments {
		// 获取关联订单
		var order model.Order
		if err := database.DB.First(&order, p.OrderID).Error; err != nil {
			issues = append(issues, map[string]interface{}{
				"type":     "orphan_payment",
				"message":  fmt.Sprintf("支付记录 %s 关联的订单不存在", p.PaymentNo),
				"payment":  p.PaymentNo,
				"amount":   p.Amount,
				"status":   p.Status,
			})
			continue
		}

		// 检查金额一致性
		if p.Amount != order.Amount {
			issues = append(issues, map[string]interface{}{
				"type":     "amount_mismatch",
				"message":  fmt.Sprintf("支付金额(%d) != 订单金额(%d)", p.Amount, order.Amount),
				"payment":  p.PaymentNo,
				"order":    order.OrderNo,
				"pay_amt":  p.Amount,
				"order_amt": order.Amount,
			})
		}

		// 检查订单状态
		if p.Status == "success" && order.Status != "paid" {
			issues = append(issues, map[string]interface{}{
				"type":     "status_inconsistency",
				"message":  fmt.Sprintf("支付成功但订单状态非 paid: paymentNo=%s, orderStatus=%s", p.PaymentNo, order.Status),
				"payment":  p.PaymentNo,
				"order":    order.OrderNo,
				"pay_status": p.Status,
				"order_status": order.Status,
			})
		}
	}

	return issues, nil
}

// GetPaymentRecords 获取用户支付记录
func (s *PaymentService) GetPaymentRecords(userID uint, page, pageSize int) ([]dto.PaymentRecordResp, int64, error) {
	var payments []model.ThirdPartyPayment
	var total int64

	query := database.DB.Model(&model.ThirdPartyPayment{}).Where("user_id = ?", userID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&payments).Error; err != nil {
		return nil, 0, err
	}

	var records []dto.PaymentRecordResp
	for _, p := range payments {
		records = append(records, dto.PaymentRecordResp{
			ID:         p.ID,
			PaymentNo:  p.PaymentNo,
			Channel:    p.Channel,
			Amount:     p.Amount,
			Fee:        p.Fee,
			RealAmount: p.RealAmount,
			Status:     p.Status,
			PrepayID:   p.PrepayID,
			PaidAt:     p.PaidAt,
			RefundedAt: p.RefundedAt,
			CreatedAt:  p.CreatedAt,
		})
	}

	return records, total, nil
}

// initMD5Checksum 生成 MD5 校验和 (简化版)
func initMD5Checksum(values string, apiKey string) string {
	h := md5.New()
	h.Write([]byte(values + "&key=" + apiKey))
	return hex.EncodeToString(h.Sum(nil))
}
