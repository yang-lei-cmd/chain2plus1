package main

import (
	"fmt"
	"log"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/linqi/chain2plus1/pkg/model"
)

func initAdmin() {
	dsn := "root:Linqi@2024@tcp(127.0.0.1:3306)/chain2plus1?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		log.Fatal("DB connect failed:", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Admin@2024"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	adminUser := model.User{
		Username: "admin",
		Password: string(hashedPassword),
		Phone:    "13800000000",
		Email:    "admin@chain2plus1.com",
		Role:     "admin",
		Status:   1,
	}

	if err := db.Create(&adminUser).Error; err != nil {
		// 如果是因为重复用户名导致的错误，静默跳过（管理员已存在）
		if strings.Contains(err.Error(), "Duplicate entry") {
			fmt.Println("Admin user already exists, skipping creation.")
		} else {
			log.Fatal("Failed to create admin:", err)
		}
		return
	}

	fmt.Printf("Admin user created successfully! ID=%d, Username=admin, Password=Admin@2024\n", adminUser.ID)
}
