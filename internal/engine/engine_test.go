// Package engine 链动引擎单元测试
package engine

import (
	"strconv"
	"testing"
	"time"

	"github.com/linqi/chain2plus1/pkg/logger"
	"github.com/linqi/chain2plus1/pkg/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var icSeq int64

func init() {
	logger.InitLogger("error")
}

func ic() string {
	icSeq++
	return "IC" + strconv.FormatInt(time.Now().UnixNano()+icSeq, 36)
}

func setupEngineDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	db.AutoMigrate(&model.User{}, &model.ProfitShare{}, &model.ChainRecord{}, &model.Order{}, &model.Product{}, &model.Supplier{})
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	return db
}

func TestCalculateCommission_3Level(t *testing.T) {
	db := setupEngineDB(t)
	cfg := DefaultChainConfig()

	alice := model.User{Username: "alice", Role: "customer", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&alice)
	bob := model.User{Username: "bob", Role: "customer", Level: 2, Status: 1, ParentID: &alice.ID, InviteCode: ic()}
	db.Create(&bob)
	charlie := model.User{Username: "charlie", Role: "customer", Level: 3, Status: 1, ParentID: &bob.ID, InviteCode: ic()}
	db.Create(&charlie)

	amount := int64(20000)
	details := CalculateCommission(db, charlie.ID, amount, cfg)

	if len(details) != 2 {
		t.Fatalf("expected 2 commission details, got %d", len(details))
	}
	if details[0].Level != 1 || details[0].ToUserID != bob.ID {
		t.Errorf("detail[0]: expected Level 1, ToUserID %d", bob.ID)
	}
	expectedBob := int64(float64(amount) * cfg.CommissionRate[1])
	if details[0].Amount != expectedBob {
		t.Errorf("Bob: expected %d, got %d", expectedBob, details[0].Amount)
	}
	if details[1].Level != 2 || details[1].ToUserID != alice.ID {
		t.Errorf("detail[1]: expected Level 2, ToUserID %d", alice.ID)
	}
	expectedAlice := int64(float64(amount) * cfg.CommissionRate[2])
	if details[1].Amount != expectedAlice {
		t.Errorf("Alice: expected %d, got %d", expectedAlice, details[1].Amount)
	}
}

func TestCalculateCommission_Orphan(t *testing.T) {
	db := setupEngineDB(t)
	root := model.User{Username: "orphan", Role: "customer", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&root)
	details := CalculateCommission(db, root.ID, 5000, DefaultChainConfig())
	if len(details) != 0 {
		t.Errorf("orphan should have 0 commissions, got %d", len(details))
	}
}

func TestUnlockStatus(t *testing.T) {
	db := setupEngineDB(t)
	cfg := DefaultChainConfig()

	alice := model.User{Username: "unlock_a", Role: "customer", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&alice)
	result := CalculateUnlockStatus(db, alice.ID, cfg)
	if result.IsUnlocked {
		t.Error("Alice should start locked")
	}

	child1 := model.User{Username: "child1", Role: "customer", Level: 2, Status: 1, ParentID: &alice.ID, InviteCode: ic()}
	db.Create(&child1)
	ProcessChainLock(db, child1, cfg)
	result = CalculateUnlockStatus(db, alice.ID, cfg)
	if result.IsUnlocked || result.PendingCount != 1 {
		t.Errorf("Alice should still be locked, PendingCount=1, got %v/%d", result.IsUnlocked, result.PendingCount)
	}

	child2 := model.User{Username: "child2", Role: "customer", Level: 2, Status: 1, ParentID: &alice.ID, InviteCode: ic()}
	db.Create(&child2)
	ProcessChainLock(db, child2, cfg)
	result = CalculateUnlockStatus(db, alice.ID, cfg)
	if !result.IsUnlocked {
		t.Fatal("Alice should be unlocked with 2 children")
	}
	if len(result.UnlockedByIDs) != 2 {
		t.Errorf("expected 2 unlockers, got %d", len(result.UnlockedByIDs))
	}
}

func TestDistributeCommission(t *testing.T) {
	db := setupEngineDB(t)
	cfg := DefaultChainConfig()

	alice := model.User{Username: "dist_alice", Role: "customer", Level: 1, Status: 1, InviteCode: ic()}
	db.Create(&alice)
	bob := model.User{Username: "dist_bob", Role: "customer", Level: 2, Status: 1, ParentID: &alice.ID, InviteCode: ic()}
	db.Create(&bob)
	charlie := model.User{Username: "dist_charlie", Role: "customer", Level: 3, Status: 1, ParentID: &bob.ID, InviteCode: ic()}
	db.Create(&charlie)

	order := model.Order{
		UserID: charlie.ID, ProductID: 1,
		OrderNo: "ORD-" + strconv.FormatInt(time.Now().UnixNano()+icSeq, 36),
		Amount: 50000, Status: "paid",
	}
	db.Create(&order)

	details := CalculateCommission(db, charlie.ID, order.Amount, cfg)
	err := DistributeCommission(db, order.ID, charlie.ID, order.Amount, details)
	if err != nil {
		t.Fatalf("DistributeCommission failed: %v", err)
	}

	var updatedBob, updatedAlice model.User
	db.First(&updatedBob, bob.ID)
	db.First(&updatedAlice, alice.ID)
	expectedBob := int64(float64(order.Amount) * cfg.CommissionRate[1])
	expectedAlice := int64(float64(order.Amount) * cfg.CommissionRate[2])
	if updatedBob.Balance != expectedBob {
		t.Errorf("Bob balance: expected %d, got %d", expectedBob, updatedBob.Balance)
	}
	if updatedAlice.Balance != expectedAlice {
		t.Errorf("Alice balance: expected %d, got %d", expectedAlice, updatedAlice.Balance)
	}
	var profitCount int64
	db.Model(&model.ProfitShare{}).Where("order_id = ?", order.ID).Count(&profitCount)
	if profitCount != 2 {
		t.Errorf("expected 2 profit records, got %d", profitCount)
	}
}
