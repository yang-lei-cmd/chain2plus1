// Package seed 初始数据
package seed

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/linqi/chain2plus1/pkg/logger"
	"github.com/linqi/chain2plus1/pkg/model"
)

// SeedBasicData 注入基础测试数据（1个供应商 + 3个商品 + 1个管理员）
func SeedBasicData(db *gorm.DB) {
	// 检查是否已有数据
	var count int64
	db.Model(&model.Supplier{}).Count(&count)
	if count > 0 {
		return // 已有数据，跳过
	}

	// 创建管理员账户
	adminPassword, _ := bcrypt.GenerateFromPassword([]byte("Admin@2024"), bcrypt.DefaultCost)
	adminUser := model.User{
		Username: "admin",
		Password: string(adminPassword),
		Phone:    "13800000000",
		Email:    "admin@chain2plus1.com",
		Role:     "admin",
		Status:   1,
	}
	if err := db.Create(&adminUser).Error; err != nil {
		logger.Error("Failed to create admin user: %v", err)
	} else {
		logger.Info("Admin user created: admin / Admin@2024")
	}

	suppliers := []model.Supplier{
		{
			Name:       "链动科技",
			Code:       "SUP001",
			Contact:    "张三",
			Phone:      "13800000001",
			Address:    "北京市海淀区中关村大街1号",
			BankName:   "招商银行北京分行",
			BankAccount: "6225880123456789",
			Status:     1,
		},
		{
			Name:       "数字商城",
			Code:       "SUP002",
			Contact:    "李四",
			Phone:      "13800000002",
			Address:    "上海市浦东新区陆家嘴环路100号",
			BankName:   "工商银行上海分行",
			BankAccount: "6222020123456789",
			Status:     1,
		},
	}

	for _, s := range suppliers {
		db.Create(&s)
	}

	products := []model.Product{
		{
			SupplierID:  1,
			Name:        "初级会员套餐",
			Description: "含基础权益，价格199元",
			Price:       19900, // 199元 = 19900分
			ImageURL:    "/images/product-basic.jpg",
			Status:      1,
		},
		{
			SupplierID:  1,
			Name:        "高级会员套餐",
			Description: "含高级权益+优先客服，价格499元",
			Price:       49900, // 499元
			ImageURL:    "/images/product-pro.jpg",
			Status:      1,
		},
		{
			SupplierID:  2,
			Name:        "钻石会员套餐",
			Description: "含全部权益+专属顾问，价格999元",
			Price:       99900, // 999元
			ImageURL:    "/images/product-diamond.jpg",
			Status:      1,
		},
	}

	for _, p := range products {
		db.Create(&p)
	}
}
