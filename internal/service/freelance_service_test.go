// Package service_tests 灵活用工服务单元测试
package service_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/linqi/chain2plus1/internal/service"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/dto"
	"github.com/linqi/chain2plus1/pkg/logger"
	"github.com/linqi/chain2plus1/pkg/model"

	"github.com/glebarez/sqlite"
)

var testDB *gorm.DB

func setupSuite(tb testing.TB) *gorm.DB {
	tb.Helper()
	logger.InitLogger("error")
	gin.SetMode(gin.TestMode)
	if testDB != nil {
		// Reset tables
		testDB.Migrator().DropTable(
			&model.User{}, &model.Freelancer{}, &model.Task{},
			&model.TimeLog{}, &model.Settlement{}, &model.Rating{},
		)
		testDB.AutoMigrate(
			&model.User{}, &model.Freelancer{}, &model.Task{},
			&model.TimeLog{}, &model.Settlement{}, &model.Rating{},
		)
		database.DB = testDB
		return testDB
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		tb.Fatalf("Failed to open test DB: %v", err)
	}
	db.AutoMigrate(
		&model.User{}, &model.Freelancer{}, &model.Task{},
		&model.TimeLog{}, &model.Settlement{}, &model.Rating{},
	)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	testDB = db
	database.DB = db
	return db
}

var icSeq int64

func ic() string {
	icSeq++
	return "IC" + strconv.FormatInt(time.Now().UnixNano()+icSeq, 36)
}

// ============================================================
// Freelancer Registration Unit Tests
// ============================================================

func TestFreelancerService_Register(t *testing.T) {
	db := setupSuite(t)

	user := model.User{
		Username: "testfree", Password: "hash", Role: "customer",
		Level: 1, Status: 1, InviteCode: ic(),
	}
	db.Create(&user)

	svc := service.NewFreelanceService()
	req := &dto.FreelancerRegisterReq{
		RealName:  "测试用户",
		IDCard:    "110101199001011234",
		Phone:     "13800000100",
		Email:     "test@test.com",
		SkillTags: []string{"golang", "react"},
		Bio:       "全栈开发者",
	}

	freelancer, err := svc.RegisterFreelancer(nil, user.ID, req)
	if err != nil {
		t.Fatalf("RegisterFreelancer failed: %v", err)
	}

	if freelancer.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", freelancer.Status)
	}
	if freelancer.UserID != user.ID {
		t.Errorf("expected UserID %d, got %d", user.ID, freelancer.UserID)
	}
	if freelancer.RealName != req.RealName {
		t.Errorf("expected RealName %s, got %s", req.RealName, freelancer.RealName)
	}

	// Duplicate registration (same user)
	_, err = svc.RegisterFreelancer(nil, user.ID, req)
	if err == nil {
		t.Error("expected error for duplicate registration, got nil")
	}
}

func TestFreelancerService_Approve(t *testing.T) {
	db := setupSuite(t)

	user := model.User{Username: "approveme", Password: "hash", Role: "customer", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&user)
	admin := model.User{Username: "testadmin", Password: "hash", Role: "admin", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&admin)

	freelancer := model.Freelancer{UserID: user.ID, RealName: "待审核", IDCard: "110101199001012345", Status: "pending"}
	db.Create(&freelancer)

	svc := service.NewFreelanceService()
	err := svc.ApproveFreelancer(freelancer.ID, admin.ID)
	if err != nil {
		t.Fatalf("ApproveFreelancer failed: %v", err)
	}

	var updated model.Freelancer
	db.First(&updated, freelancer.ID)
	if updated.Status != "approved" {
		t.Errorf("expected status 'approved', got '%s'", updated.Status)
	}
	if updated.ApprovedBy == nil || *updated.ApprovedBy != admin.ID {
		t.Error("expected ApprovedBy to be set")
	}
}

func TestFreelancerService_CreateAndAssignTask(t *testing.T) {
	db := setupSuite(t)

	publisher := model.User{Username: "pub1", Password: "hash", Role: "customer", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&publisher)
	freeUser := model.User{Username: "free1", Password: "hash", Role: "customer", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&freeUser)
	freelancer := model.Freelancer{UserID: freeUser.ID, RealName: "自由者", IDCard: "110101199001013456", Status: "approved"}
	db.Create(&freelancer)

	svc := service.NewFreelanceService()

	// Create task
	task, err := svc.CreateTask(nil, publisher.ID, &dto.TaskCreateReq{
		Title:         "测试任务",
		Description:   "单元测试任务",
		Category:      "dev",
		SkillTags:     []string{"go"},
		Budget:        50000,
		DurationHours: 24,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if task.Status != "open" {
		t.Errorf("expected status 'open', got '%s'", task.Status)
	}
	if task.PublisherID != publisher.ID {
		t.Errorf("expected PublisherID %d, got %d", publisher.ID, task.PublisherID)
	}
	if task.Budget != 50000 {
		t.Errorf("expected Budget 50000, got %d", task.Budget)
	}

	// Assign task
	err = svc.AssignTask(task.ID, freelancer.ID)
	if err != nil {
		t.Fatalf("AssignTask failed: %v", err)
	}

	var assigned model.Task
	db.First(&assigned, task.ID)
	if assigned.Status != "assigned" {
		t.Errorf("expected status 'assigned', got '%s'", assigned.Status)
	}
	if assigned.AssignedTo == nil || *assigned.AssignedTo != freelancer.ID {
		t.Errorf("expected AssignedTo=%d, got %v", freelancer.ID, assigned.AssignedTo)
	}
}

func TestFreelancerService_SubmitAndReview(t *testing.T) {
	db := setupSuite(t)

	publisher := model.User{Username: "pub2", Password: "hash", Role: "customer", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&publisher)
	freeUser := model.User{Username: "free2", Password: "hash", Role: "customer", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&freeUser)
	freelancer := model.Freelancer{UserID: freeUser.ID, RealName: "赵六", IDCard: "110101199001014567", Status: "approved"}
	db.Create(&freelancer)

	svc := service.NewFreelanceService()
	task := model.Task{
		Title: "提交流程", Description: "测试", Category: "dev",
		Budget: 30000, PublisherID: publisher.ID, Status: "assigned", AssignedTo: &freelancer.ID,
	}
	db.Create(&task)

	// Submit
	err := svc.SubmitWork(task.ID, "已完成开发工作")
	if err != nil {
		t.Fatalf("SubmitWork failed: %v", err)
	}
	var submitted model.Task
	db.First(&submitted, task.ID)
	if submitted.Status != "submitted" {
		t.Errorf("expected 'submitted', got '%s'", submitted.Status)
	}

	// Review (approve)
	err = svc.ReviewWork(task.ID, publisher.ID, true, "质量合格")
	if err != nil {
		t.Fatalf("ReviewWork failed: %v", err)
	}
	db.First(&submitted, task.ID)
	if submitted.Status != "completed" {
		t.Errorf("expected 'completed' (approved), got '%s'", submitted.Status)
	}
	if submitted.ReviewComment != "质量合格" {
		t.Errorf("ReviewComment mismatch: '%s'", submitted.ReviewComment)
	}

	// Review (reject) — task should already be completed, create a new one for reject path
	task2 := model.Task{
		Title: "拒绝流程", Description: "测试", Category: "dev",
		Budget: 10000, PublisherID: publisher.ID, Status: "assigned", AssignedTo: &freelancer.ID,
	}
	db.Create(&task2)
	err = svc.SubmitWork(task2.ID, "工作提交")
	if err != nil {
		t.Fatal(err)
	}
	err = svc.ReviewWork(task2.ID, publisher.ID, false, "需要修改")
	if err != nil {
		t.Fatalf("ReviewWork(reject) failed: %v", err)
	}
	db.First(&task2, task2.ID)
	if task2.Status != "reviewed" {
		t.Errorf("expected 'reviewed' (rejected), got '%s'", task2.Status)
	}
}
