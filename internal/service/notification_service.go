// Package service Phase C: 通知服务（邮件/短信 Mock）
package service

import (
	"fmt"
	"log"
)

// NotificationType 通知类型
type NotificationType string

const (
	NotifRegister  NotificationType = "register"
	NotifWithdraw  NotificationType = "withdraw"
	NotifApproved  NotificationType = "approved"
	NotifRejected  NotificationType = "rejected"
	NotifRecharge  NotificationType = "recharge"
)

// NotificationService 通知服务
type NotificationService struct {
	// 实际集成时需要配置:
	// SMTP config for email
	// SMS API key for SMS
}

// NewNotificationService 创建通知服务
func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// SendEmail 发送邮件 (Mock)
func (s *NotificationService) SendEmail(to, subject, body string) error {
	log.Printf("[NOTIFICATION] Email to=%s subject=%s body=%s", to, subject, body)
	return nil
}

// SendSMS 发送短信 (Mock)
func (s *NotificationService) SendSMS(phone, content string) error {
	log.Printf("[NOTIFICATION] SMS to=%s content=%s", phone, content)
	return nil
}

// NotifyRegister 注册成功通知
func (s *NotificationService) NotifyRegister(email, phone, username string) {
	msg := fmt.Sprintf("欢迎 %s 注册链动2+1分销系统！", username)
	if email != "" {
		s.SendEmail(email, "注册成功", msg)
	}
	if phone != "" {
		s.SendSMS(phone, msg)
	}
}

// NotifyWithdraw 提现通知
func (s *NotificationService) NotifyWithdraw(email, phone string, amount int64, status string) {
	action := "已提交"
	if status == "approved" {
		action = "已通过"
	} else if status == "rejected" {
		action = "已被拒绝"
	}
	msg := fmt.Sprintf("您的提现申请 ¥%.2f %s", float64(amount)/100, action)
	if email != "" {
		s.SendEmail(email, "提现通知", msg)
	}
	if phone != "" {
		s.SendSMS(phone, msg)
	}
}

// NotifyRecharge 充值通知
func (s *NotificationService) NotifyRecharge(email, phone string, amount int64) {
	msg := fmt.Sprintf("您的账户已成功充值 ¥%.2f", float64(amount)/100)
	if email != "" {
		s.SendEmail(email, "充值成功", msg)
	}
	if phone != "" {
		s.SendSMS(phone, msg)
	}
}
