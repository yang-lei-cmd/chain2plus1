// Package engine 链动引擎核心实现
package engine

import (
	"encoding/json"
	"fmt"

	"github.com/linqi/chain2plus1/pkg/logger"
	"github.com/linqi/chain2plus1/pkg/model"

	"gorm.io/gorm"
)

// ChainConfig 链动配置
type ChainConfig struct {
	UnlockRequirement int               `json:"unlock_requirement"` // 激活所需下线数 (2+1 => 2)
	CommissionRate    map[int]float64   `json:"commission_rate"`    // 各级分润比例
	MaxLevel          int               `json:"max_level"`          // 最大分润层级
}

// DefaultChainConfig 默认链动配置
func DefaultChainConfig() ChainConfig {
	return ChainConfig{
		UnlockRequirement: 2, // 每人需激活2个下线才能解锁
		CommissionRate: map[int]float64{
			1: 0.10, // 一级 10%
			2: 0.08, // 二级 8%
			3: 0.05, // 三级 5%
			4: 0.03, // 四级 3%
			5: 0.02, // 五级 2%
		},
		MaxLevel: 5,
	}
}

// UnlockResult 解锁结果
type UnlockResult struct {
	UserID         uint     `json:"user_id"`
	IsUnlocked     bool     `json:"is_unlocked"`
	UnlockedByIDs  []uint   `json:"unlocked_by_ids"` // 帮忙解锁的下线ID
	PendingCount   int      `json:"pending_count"`   // 还差几个
}

// CalculateUnlockStatus 计算用户解锁状态
func CalculateUnlockStatus(db *gorm.DB, userID uint, cfg ChainConfig) UnlockResult {
	result := UnlockResult{
		UserID:      userID,
		IsUnlocked:  false,
		PendingCount: cfg.UnlockRequirement,
	}

	// 查询该用户的所有直接下线（status=active 的）
	var children []model.User
	db.Where("parent_id = ? AND status = 1", userID).Find(&children)

	unlockedCount := 0
	unlockedByIDs := make([]uint, 0)

	for _, child := range children {
		// 只有已激活的用户才算解锁贡献
		if child.Level > 0 && child.InviteCode != "" {
			unlockedCount++
			unlockedByIDs = append(unlockedByIDs, child.ID)
		}
	}

	result.UnlockedByIDs = unlockedByIDs
	result.PendingCount = cfg.UnlockRequirement - unlockedCount

	if unlockedCount >= cfg.UnlockRequirement {
		result.IsUnlocked = true
	}

	return result
}

// ForceUnlock 强制解锁用户
func ForceUnlock(db *gorm.DB, userID uint) error {
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// 标记为已解锁（通过更新状态）
	user.Status = 2 // 2: unlocked
	return db.Save(&user).Error
}

// ProcessChainLock 处理链动解锁逻辑 — 用户注册后检查上级是否需要解锁
func ProcessChainLock(db *gorm.DB, newUser model.User, cfg ChainConfig) {
	if newUser.ParentID == nil {
		return // 根用户无上级
	}

	parentID := *newUser.ParentID
	parent := CalculateUnlockStatus(db, parentID, cfg)

	if parent.IsUnlocked {
		// 上级解锁成功
		if err := ForceUnlock(db, parentID); err != nil {
			logger.Error("Failed to unlock parent %d: %v", parentID, err)
			return
		}

		// 记录解锁事件
		record := model.ChainRecord{
			UserID:    newUser.ID,
			Action:    "unlock",
			RelatedID: parentID,
			OrderID:   0,
			Status:    "success",
			Data:      toJSON(map[string]interface{}{
				"parent_id":     parentID,
				"unlocked_by":   parent.UnlockedByIDs,
				"unlock_reason": "registered " + fmt.Sprint(len(parent.UnlockedByIDs)) + " children",
			}),
		}
		db.Create(&record)
		logger.Info("User %d unlocked by user %d's children", parentID, newUser.ID)
	}
}

// ==================== 分润计算引擎 ====================

// CommissionDetail 分润明细
type CommissionDetail struct {
	ToUserID   uint    `json:"to_user_id"`
	ToUsername string  `json:"to_username"`
	Level      int     `json:"level"`
	Rate       float64 `json:"rate"`
	Amount     int64   `json:"amount"` // 分
}

// CalculateCommission 计算分润 — 从下单用户沿关系链向上计算
func CalculateCommission(db *gorm.DB, orderUserID uint, amount int64, cfg ChainConfig) []CommissionDetail {
	details := make([]CommissionDetail, 0)

	// 从下单用户开始，沿父级向上遍历
	currentID := orderUserID
	level := 1

	for level <= cfg.MaxLevel {
		var parent model.User
		if err := db.Select("id", "username", "parent_id", "status").First(&parent, currentID).Error; err != nil {
			break // 没有上级了
		}

		if parent.ParentID == nil {
			break // 到达根用户
		}

		rate, exists := cfg.CommissionRate[level]
		if !exists {
			break
		}

		commission := int64(float64(amount) * rate)
		if commission <= 0 {
			break
		}

		details = append(details, CommissionDetail{
			ToUserID:   *parent.ParentID,
			ToUsername: parent.Username,
			Level:      level,
			Rate:       rate,
			Amount:     commission,
		})

		currentID = *parent.ParentID
		level++
	}

	return details
}

// DistributeCommission 发放分润并更新用户余额
func DistributeCommission(db *gorm.DB, orderID uint, fromUserID uint, amount int64, details []CommissionDetail) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, detail := range details {
			if detail.Amount <= 0 {
				continue
			}

			// 创建分润记录
			profit := model.ProfitShare{
				OrderID:     orderID,
				FromUserID:  fromUserID,
				ToUserID:    detail.ToUserID,
				Level:       detail.Level,
				Amount:      detail.Amount,
				Type:        "direct",
				Status:      "settled",
				Description: fmt.Sprintf("Level %d commission: %.1f%%", detail.Level, detail.Rate*100),
			}
			if err := tx.Create(&profit).Error; err != nil {
				return fmt.Errorf("failed to create profit share: %w", err)
			}

			// 更新收款用户余额（仅更新两个金额字段，不覆盖其他字段）
			if err := tx.Model(&model.User{}).Where("id = ?", detail.ToUserID).UpdateColumns(map[string]interface{}{
				"balance":      gorm.Expr("balance + ?", detail.Amount),
				"total_earned": gorm.Expr("total_earned + ?", detail.Amount),
			}).Error; err != nil {
				return fmt.Errorf("failed to update user %d balance: %w", detail.ToUserID, err)
			}

			logger.Info("Commission: %.2f to user %d (%s) from order %d",
				float64(detail.Amount)/100.0, detail.ToUserID, detail.ToUsername, orderID)
		}

		return nil
	})
}

// Helper: JSON marshal
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
