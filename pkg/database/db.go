// Package database 数据库连接与初始化
package database

import (
	"fmt"
	"log"
	"time"

	"github.com/linqi/chain2plus1/internal/config"
	"github.com/linqi/chain2plus1/pkg/logger"
	"github.com/linqi/chain2plus1/pkg/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// Connect 连接数据库
func Connect(cfg *config.DatabaseConfig) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.Charset,
		cfg.Loc,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Info("Database connected successfully: %s:%s/%s", cfg.Host, cfg.Port, cfg.Name)
}

// Migrate 自动迁移数据库表 (Phase 5: 新增第三方扩展模型)
func Migrate() {
	err := DB.AutoMigrate(
		// Phase 1-2: 用户系统
		&model.User{},
		&model.Supplier{},
		&model.Product{},
		&model.Order{},
		&model.ProfitShare{},
		&model.ChainRecord{},
		// Phase 4: 提现+排行榜
		&model.Withdraw{},
		&model.LeaderboardRank{},
		// Phase 5: 第三方支付 + 灵活用工
		&model.ThirdPartyPayment{},
		&model.SupplierInvoice{},
		&model.Freelancer{},
		&model.Task{},
		&model.TimeLog{},
		&model.Settlement{},
		&model.Rating{},
		// Phase C: 审计日志
		&model.AuditLog{},
	)

	if err != nil {
		logger.Error("Failed to migrate database: %v", err)
	}

	logger.Info("Database migration completed")
}
