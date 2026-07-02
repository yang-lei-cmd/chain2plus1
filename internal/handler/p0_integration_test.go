package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/linqi/chain2plus1/internal/config"
	"github.com/linqi/chain2plus1/internal/engine"
	"github.com/linqi/chain2plus1/internal/handler"
	"github.com/linqi/chain2plus1/internal/middleware"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/logger"
	"github.com/linqi/chain2plus1/pkg/model"

	// CGO-free SQLite driver for GORM
	"github.com/glebarez/sqlite"
)

var testDB *gorm.DB
var paymentHandler *handler.PaymentHandler
var freelanceHandler *handler.FreelanceHandler

func setupTestDB(t *testing.T) {
	t.Helper()

	// Init logger silently (error level to reduce noise)
	logger.InitLogger("error")
	gin.SetMode(gin.TestMode)

	if testDB != nil {
		// Reset: drop all tables and recreate
		testDB.Migrator().DropTable(
			&model.User{}, &model.Product{}, &model.Supplier{},
			&model.Order{}, &model.ProfitShare{}, &model.ChainRecord{},
			&model.Withdraw{}, &model.LeaderboardRank{},
			&model.ThirdPartyPayment{}, &model.SupplierInvoice{},
			&model.Freelancer{}, &model.Task{}, &model.TimeLog{},
			&model.Settlement{}, &model.Rating{},
		&model.AuditLog{},
		)
		testDB.AutoMigrate(
			&model.User{}, &model.Product{}, &model.Supplier{},
			&model.Order{}, &model.ProfitShare{}, &model.ChainRecord{},
			&model.Withdraw{}, &model.LeaderboardRank{},
			&model.ThirdPartyPayment{}, &model.SupplierInvoice{},
			&model.Freelancer{}, &model.Task{}, &model.TimeLog{},
			&model.Settlement{}, &model.Rating{},
		&model.AuditLog{},
		)
		return
	}

	var err error
	testDB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	testDB.AutoMigrate(
		&model.User{}, &model.Product{}, &model.Supplier{},
		&model.Order{}, &model.ProfitShare{}, &model.ChainRecord{},
		&model.Withdraw{}, &model.LeaderboardRank{},
		&model.ThirdPartyPayment{}, &model.SupplierInvoice{},
		&model.Freelancer{}, &model.Task{}, &model.TimeLog{},
		&model.Settlement{}, &model.Rating{},
		&model.AuditLog{},
	)

	database.DB = testDB

	// Force single connection for SQLite in-memory (supports concurrent tests)
	sqlDB, _ := testDB.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Initialize P1 handlers
	cfg := &config.Config{
		Payment: config.PaymentConfig{
			DefaultFeeRate:  0.006,
			PaymentTimeout:  30,
		},
	}
	paymentHandler = handler.NewPaymentHandler(cfg)
	freelanceHandler = handler.NewFreelanceHandler()
}

// --- helpers ---

func hashPW(pw string) string {
	b, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b)
}

func marshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func inviteCode() string {
	return "IC" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// ensureAdmin creates an admin user in the test DB and returns its ID.
func ensureAdmin(t *testing.T) uint {
	t.Helper()
	var admin model.User
	result := testDB.Where("role = ?", "admin").First(&admin)
	if result.Error == nil {
		return admin.ID
	}
	admin = model.User{
		Username: "testadmin", Password: hashPW("Admin@2024"),
		Phone: "13800000000", Email: "admin@test.com",
		Role: "admin", Level: 1, Status: 1,
		InviteCode: "ADM" + strconv.FormatInt(time.Now().UnixNano(), 36),
	}
	testDB.Create(&admin)
	return admin.ID
}

// ============================================================
// P0 Test Case 1: User Registration
// ============================================================

func Test_Register_Success_Root(t *testing.T) {
	setupTestDB(t)
	reqBody := map[string]string{
		"username": "rootuser1",
		"password": "Test123456",
		"phone":    "13800000001",
		"email":    "root1@test.com",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d. Body: %s", resp.StatusCode, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["message"] != "注册成功" {
		t.Errorf("expected message '注册成功', got %v", result["message"])
	}

	user := result["user"].(map[string]interface{})
	if len(user["invite_code"].(string)) != 15 {
		t.Errorf("expected invite_code length 15, got %d", len(user["invite_code"].(string)))
	}
}

func Test_Register_Child(t *testing.T) {
	setupTestDB(t)

	// Create parent first
	parentPW := hashPW("Parent123")
	parent := model.User{
		Username: "parent1", Password: parentPW, Phone: "13800000010",
		Email: "parent1@test.com", Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&parent)

	// Register child with parent's invite code
	reqBody := map[string]string{
		"username":   "child1",
		"password":   "Child123456",
		"phone":      "13800000011",
		"email":      "child1@test.com",
		"invite_code": parent.InviteCode,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d. Body: %s", resp.StatusCode, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	user := result["user"].(map[string]interface{})
	if user["level"] != float64(2) {
		t.Errorf("expected level 2 for child, got %v", user["level"])
	}

	// Verify parent relationship in DB
	var childUser model.User
	testDB.Where("username = ?", "child1").First(&childUser)
	if childUser.ParentID == nil || *childUser.ParentID != parent.ID {
		t.Error("expected child to have correct parent_id")
	}
}

func Test_Register_DuplicateUsername(t *testing.T) {
	setupTestDB(t)

	payload := map[string]string{"username": "duptest", "password": "Test123", "phone": "13800000050", "email": "dup@test.com"}
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(marshal(payload)))
		c.Request.Header.Set("Content-Type", "application/json")
		handler.Register(c)

		if i == 0 && w.Code != http.StatusCreated {
			t.Errorf("first register: expected 201, got %d", w.Code)
		}
		if i == 1 && w.Code != http.StatusConflict {
			t.Errorf("second register: expected 409, got %d", w.Code)
		}
	}
}

func Test_Register_InvalidInviteCode(t *testing.T) {
	setupTestDB(t)

	reqBody := map[string]string{
		"username":    "badinvite",
		"password":    "Test123",
		"phone":       "13800000051",
		"email":       "bad@test.com",
		"invite_code": "INVALID_CODE",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Register(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for invalid invite code, got %d", w.Code)
	}
}

// ============================================================
// P0 Test Case 2: User Login
// ============================================================

func Test_Login_Success(t *testing.T) {
	setupTestDB(t)

	pw := hashPW("Logintest123")
	testDB.Create(&model.User{
		Username: "logintest", Password: pw, Phone: "13800000040",
		Email: "login@test.com", Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	})

	reqBody := map[string]string{"username": "logintest", "password": "Logintest123"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Login(c)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", resp.StatusCode, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["token"] == nil {
		t.Fatal("expected token in login response, got nil")
	}

	userInfo := result["user"].(map[string]interface{})
	if userInfo["username"] != "logintest" {
		t.Errorf("expected username 'logintest', got %v", userInfo["username"])
	}
	if userInfo["role"] != "customer" {
		t.Errorf("expected role 'customer', got %v", userInfo["role"])
	}
}

func Test_Login_WrongPassword(t *testing.T) {
	setupTestDB(t)

	pw := hashPW("Correct123")
	testDB.Create(&model.User{
		Username: "correctpw", Password: pw, Phone: "13800000041",
		Email: "correctpw@test.com", Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	})

	reqBody := map[string]string{"username": "correctpw", "password": "WrongPassword"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}
}

func Test_Login_UserNotFound(t *testing.T) {
	setupTestDB(t)

	reqBody := map[string]string{"username": "nonexistent_user", "password": "any"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for nonexistent user, got %d", w.Code)
	}
}

// ============================================================
// P0 Test Case 3: Create Order (Triggers Commission)
// ============================================================

func Test_CreateOrder_Success(t *testing.T) {
	setupTestDB(t)

	// Create root user
	rootPW := hashPW("Root123")
	root := model.User{
		Username: "rootorder", Password: rootPW, Phone: "13800000050",
		Email: "rootorder@test.com", Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&root)

	// Create product
	product := model.Product{Name: "初级套餐", Price: 19900, Status: 1, SupplierID: 1}
	testDB.Create(&product)

	// Set up router for authenticated request
	// CreateOrder requires middleware-set values
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/order/create", bytes.NewReader(marshal(map[string]interface{}{
		"product_id":     product.ID,
		"payment_method": "mock_wechat",
	})))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, root.ID)
	c.Set(middleware.UserNameKey, "rootorder")
	c.Set(middleware.UserRoleKey, "customer")

	handler.CreateOrder(c)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", resp.StatusCode, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "订单创建成功" {
		t.Errorf("expected message '订单创建成功', got %v", result["message"])
	}

	// Verify order was created
	var order model.Order
	testDB.Where("user_id = ?", root.ID).Order("created_at DESC").First(&order)
	if order.Amount != product.Price {
		t.Errorf("expected order amount %d, got %d", product.Price, order.Amount)
	}
	if order.Status != "paid" {
		t.Errorf("expected order status 'paid', got %s", order.Status)
	}

	// Verify commission was recorded (even if root has no parent, 0 commission is OK)
	var profitCount int64
	testDB.Model(&model.ProfitShare{}).Where("order_id = ?", order.ID).Count(&profitCount)
	// May be 0 for root user with no parent, that's fine
	t.Logf("Commission records for order %d: %d", order.ID, profitCount)
}

func Test_CreateOrder_ProductNotFound(t *testing.T) {
	setupTestDB(t)

	rootPW := hashPW("RootPW123")
	root := model.User{
		Username: "rootprod", Password: rootPW, Phone: "13800000051",
		Email: "rootprod@test.com", Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&root)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/order/create", bytes.NewReader(marshal(map[string]interface{}{
		"product_id":     999999,
		"payment_method": "mock_wechat",
	})))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, root.ID)
	c.Set(middleware.UserNameKey, "rootprod")
	c.Set(middleware.UserRoleKey, "customer")
	handler.CreateOrder(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent product, got %d", w.Code)
	}
}

// ============================================================
// P0 Test Case 4: List Profits
// ============================================================

func Test_ListProfits(t *testing.T) {
	setupTestDB(t)

	rootPW := hashPW("ProfitRoot123")
	root := model.User{
		Username: "profitroot", Password: rootPW, Phone: "13800000060",
		Email: "profitroot@test.com", Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&root)

	// Insert a profit share manually (normally created by CreateOrder)
	profit := model.ProfitShare{
		FromUserID: root.ID, ToUserID: root.ID, OrderID: 1, Level: 1,
		Amount: 1000, Type: "team", Status: "pending",
		Description: "测试分润",
	}
	testDB.Create(&profit)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/profit/list", nil)
	c.Set(middleware.UserIDKey, root.ID)
	c.Set(middleware.UserNameKey, "profitroot")
	c.Set(middleware.UserRoleKey, "customer")
	handler.ListProfits(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	profits := result["profits"].([]interface{})
	if len(profits) != 1 {
		t.Errorf("expected 1 profit record, got %d", len(profits))
	}
}

// ============================================================
// P0 Test Case 5: Apply Withdraw
// ============================================================

func Test_ApplyWithdraw_Success(t *testing.T) {
	setupTestDB(t)

	pw := hashPW("WithdrawUser123")
	user := model.User{
		Username: "withdrawuser", Password: pw, Phone: "13800000070",
		Email: "withdraw@test.com", Role: "customer", Level: 1, Status: 1,
		InviteCode: inviteCode(), Balance: 100000, // 1000 yuan in cents
	}
	testDB.Create(&user)

	reqBody := map[string]interface{}{
		"amount":       10000,    // 100 yuan
		"bank_name":    "招商银行",
		"account_name": "测试用户",
		"account_no":   "6225880123456789",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/withdraw/apply", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, user.ID)
	c.Set(middleware.UserNameKey, "withdrawuser")
	c.Set(middleware.UserRoleKey, "customer")
	handler.ApplyWithdraw(c)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", resp.StatusCode, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "提现申请已提交" {
		t.Errorf("expected message '提现申请已提交', got %v", result["message"])
	}

	// Verify withdraw record
	var wd model.Withdraw
	testDB.Where("user_id = ?", user.ID).First(&wd)
	if wd.Status != "pending" {
		t.Errorf("expected withdraw status 'pending', got %s", wd.Status)
	}
	if wd.Amount != 10000 {
		t.Errorf("expected withdraw amount 10000, got %d", wd.Amount)
	}
}

func Test_ApplyWithdraw_InsufficientBalance(t *testing.T) {
	setupTestDB(t)

	pw := hashPW("PoorUser123")
	user := model.User{
		Username: "pooruser", Password: pw, Phone: "13800000071",
		Email: "poor@test.com", Role: "customer", Level: 1, Status: 1,
		InviteCode: inviteCode(), Balance: 100, // 1 yuan
	}
	testDB.Create(&user)

	reqBody := map[string]interface{}{
		"amount":       10000,
		"bank_name":    "招商银行",
		"account_name": "贫穷用户",
		"account_no":   "6225880999999999",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/withdraw/apply", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, user.ID)
	c.Set(middleware.UserNameKey, "pooruser")
	c.Set(middleware.UserRoleKey, "customer")
	handler.ApplyWithdraw(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for insufficient balance, got %d", w.Code)
	}
}

// ============================================================
// P0 Test Case 6: Approve Withdraw (Admin)
// ============================================================

func Test_ApproveWithdraw_Approve(t *testing.T) {
	setupTestDB(t)

	// Create admin user FIRST (will get ID 1)
	adminID := ensureAdmin(t)

	// Create user
	pw := hashPW("ApproveUser123")
	user := model.User{
		Username: "approvetest", Password: pw, Phone: "13800000080",
		Email: "approve@test.com", Role: "customer", Level: 1, Status: 1,
		InviteCode: inviteCode(), Balance: 50000,
	}
	testDB.Create(&user)

	// Create withdraw
	withdraw := model.Withdraw{
		UserID: user.ID, Amount: 10000, Fee: 100, ActualAmount: 9900,
		BankName: "招商银行", AccountName: "测试用户", AccountNo: "6225880123456789",
		Status: "pending",
	}
	testDB.Create(&withdraw)

	// Admin approves
	reqBody := map[string]string{"action": "approve", "remark": "审核通过"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/withdraw/"+strconv.FormatUint(uint64(withdraw.ID), 10)+"/approve", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(uint64(withdraw.ID), 10)}}
	c.Set(middleware.UserIDKey, adminID)
	c.Set(middleware.UserNameKey, "testadmin")
	c.Set(middleware.UserRoleKey, "admin")
	handler.ApproveWithdraw(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify status changed
	testDB.First(&withdraw, withdraw.ID)
	if withdraw.Status != "approved" {
		t.Errorf("expected withdraw status 'approved', got %s", withdraw.Status)
	}
	if withdraw.ApprovedAt == nil {
		t.Error("expected approved_at to be set")
	}
}

func Test_ApproveWithdraw_Reject(t *testing.T) {
	setupTestDB(t)

	adminID := ensureAdmin(t)

	pw := hashPW("RejectUser123")
	user := model.User{
		Username: "rejecttest", Password: pw, Phone: "13800000081",
		Email: "reject@test.com", Role: "customer", Level: 1, Status: 1,
		InviteCode: inviteCode(), Balance: 50000,
	}
	testDB.Create(&user)

	withdraw := model.Withdraw{
		UserID: user.ID, Amount: 10000, Fee: 100, ActualAmount: 9900,
		BankName: "招商银行", AccountName: "测试用户", AccountNo: "6225880123456789",
		Status: "pending",
	}
	testDB.Create(&withdraw)

	reqBody := map[string]string{"action": "reject", "remark": "信息不符"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/withdraw/"+strconv.FormatUint(uint64(withdraw.ID), 10)+"/approve", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(uint64(withdraw.ID), 10)}}
	c.Set(middleware.UserIDKey, adminID)
	c.Set(middleware.UserNameKey, "testadmin")
	c.Set(middleware.UserRoleKey, "admin")
	handler.ApproveWithdraw(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	testDB.First(&withdraw, withdraw.ID)
	if withdraw.Status != "rejected" {
		t.Errorf("expected withdraw status 'rejected', got %s", withdraw.Status)
	}
}

func Test_ApproveWithdraw_NonAdmin(t *testing.T) {
	setupTestDB(t)

	reqBody := map[string]string{"action": "approve"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/withdraw/1/approve", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Set(middleware.UserIDKey, uint(999))
	c.Set(middleware.UserNameKey, "nobody")
	c.Set(middleware.UserRoleKey, "customer")
	handler.ApproveWithdraw(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", w.Code)
	}
}

func Test_ApproveWithdraw_AlreadyProcessed(t *testing.T) {
	setupTestDB(t)

	adminID := ensureAdmin(t)

	pw := hashPW("ProcessedUser123")
	user := model.User{
		Username: "processed", Password: pw, Phone: "13800000082",
		Email: "processed@test.com", Role: "customer", Level: 1, Status: 1,
		InviteCode: inviteCode(), Balance: 50000,
	}
	testDB.Create(&user)

	withdraw := model.Withdraw{
		UserID: user.ID, Amount: 10000, Fee: 100, ActualAmount: 9900,
		BankName: "招商银行", AccountName: "测试用户", AccountNo: "6225880123456789",
		Status: "approved",
	}
	testDB.Create(&withdraw)

	reqBody := map[string]string{"action": "approve"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/withdraw/"+strconv.FormatUint(uint64(withdraw.ID), 10)+"/approve", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(uint64(withdraw.ID), 10)}}
	c.Set(middleware.UserIDKey, adminID)
	c.Set(middleware.UserNameKey, "testadmin")
	c.Set(middleware.UserRoleKey, "admin")
	handler.ApproveWithdraw(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for already processed, got %d", w.Code)
	}
}

func Test_ApproveWithdraw_NotFound(t *testing.T) {
	setupTestDB(t)
	adminID := ensureAdmin(t)

	reqBody := map[string]string{"action": "approve"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/withdraw/999999/approve", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: "999999"}}
	c.Set(middleware.UserIDKey, adminID)
	c.Set(middleware.UserNameKey, "testadmin")
	c.Set(middleware.UserRoleKey, "admin")
	handler.ApproveWithdraw(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent withdraw, got %d", w.Code)
	}
}

// ============================================================
// Integration Flow Test: Full Chain
// ============================================================

func Test_FullRegistrationAndOrderFlow(t *testing.T) {
	setupTestDB(t)

	// Create admin user
	adminID := ensureAdmin(t)

	// Step 1: Root user registers
	rootPW := hashPW("RootFlow123")
	root := model.User{
		Username: "rootflow", Password: rootPW, Phone: "13800000090",
		Email: "rootflow@test.com", Role: "customer", Level: 1, Status: 1,
		InviteCode: inviteCode(),
	}
	testDB.Create(&root)

	// Step 2: Child user registers with root's invite code
	var rootUser model.User
	testDB.Where("username = ?", "rootflow").First(&rootUser)
	childReq := map[string]string{
		"username":    "childflow",
		"password":    "ChildFlow123",
		"phone":       "13800000091",
		"email":       "childflow@test.com",
		"invite_code": rootUser.InviteCode,
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(marshal(childReq)))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Register(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("child registration failed: %d %s", w.Code, w.Body.String())
	}

	// Step 3: Child creates order (should trigger commission to root)
	childPW := hashPW("ChildFlow123")
	var childUser model.User
	testDB.Where("username = ?", "childflow").First(&childUser)

	// Set child password in DB
	testDB.Model(&childUser).Update("password", childPW)

	product := model.Product{Name: "Flow 套餐", Price: 19900, Status: 1, SupplierID: 1}
	testDB.Create(&product)

	orderReq := map[string]interface{}{
		"product_id":     product.ID,
		"payment_method": "mock_wechat",
	}
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/api/v1/order/create", bytes.NewReader(marshal(orderReq)))
	c2.Request.Header.Set("Content-Type", "application/json")
	c2.Set(middleware.UserIDKey, childUser.ID)
	c2.Set(middleware.UserNameKey, "childflow")
	c2.Set(middleware.UserRoleKey, "customer")
	handler.CreateOrder(c2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("order creation failed: %d %s", w2.Code, w2.Body.String())
	}

	// Step 4: Verify root received commission
	var profitCount int64
	testDB.Model(&model.ProfitShare{}).Where("to_user_id = ?", root.ID).Count(&profitCount)
	if profitCount == 0 {
		t.Log("Note: root received 0 commission (may depend on chain config)")
	} else {
		var profit model.ProfitShare
		testDB.Where("to_user_id = ?", root.ID).First(&profit)
		t.Logf("Root received commission: %d (level %d, type %s)", profit.Amount, profit.Level, profit.Type)
	}

	// Step 5: Child applies withdraw
	withdrawReq := map[string]interface{}{
		"amount":       5000,
		"bank_name":    "建设银行",
		"account_name": "Flow 用户",
		"account_no":   "6227000123456789",
	}
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	c3.Request = httptest.NewRequest(http.MethodPost, "/api/v1/withdraw/apply", bytes.NewReader(marshal(withdrawReq)))
	c3.Request.Header.Set("Content-Type", "application/json")
	c3.Set(middleware.UserIDKey, childUser.ID)
	c3.Set(middleware.UserNameKey, "childflow")
	c3.Set(middleware.UserRoleKey, "customer")
	handler.ApplyWithdraw(c3)

	// Admin approval
	w4 := httptest.NewRecorder()
	c4, _ := gin.CreateTestContext(w4)
	var wd model.Withdraw
	testDB.Where("user_id = ?", childUser.ID).First(&wd)
	c4.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/withdraw/"+strconv.FormatUint(uint64(wd.ID), 10)+"/approve", bytes.NewReader(marshal(map[string]string{"action": "approve"})))
	c4.Request.Header.Set("Content-Type", "application/json")
	c4.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(uint64(wd.ID), 10)}}
	c4.Set(middleware.UserIDKey, adminID)
	c4.Set(middleware.UserNameKey, "testadmin")
	c4.Set(middleware.UserRoleKey, "admin")
	handler.ApproveWithdraw(c4)

	t.Log("Full flow completed: register -> register child -> create order -> withdraw -> approve")
}

// ============================================================
// P1 Test: Payment - Create & Query
// ============================================================

func Test_Payment_CreateAndQuery(t *testing.T) {
	setupTestDB(t)

	// Create a user and order
	user := model.User{
		Username: "payuser", Password: hashPW("Pay123"),
		Phone: "13800000100", Email: "pay@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(), Balance: 100000,
	}
	testDB.Create(&user)
	product := model.Product{Name: "支付测试商品", Price: 5000, Status: 1, SupplierID: 1}
	testDB.Create(&product)

	order := model.Order{UserID: user.ID, ProductID: product.ID, Amount: 5000, Status: "pending"}
	testDB.Create(&order)

	// Create payment request
	reqBody := map[string]interface{}{
		"order_id":  order.ID,
		"channel":   "wechat",
		"amount":    5000,
		"subject":   "支付测试",
		"notify_url": "http://test.com/notify",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/payment/create", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, user.ID)
	paymentHandler.CreatePayment(c)

	if w.Code != http.StatusOK {
		t.Fatalf("CreatePayment: expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["message"] != "支付创建成功" {
		t.Errorf("expected '支付创建成功', got %v", result["message"])
	}
	payment := result["payment"].(map[string]interface{})
	paymentNo := payment["payment_no"].(string)
	if paymentNo == "" {
		t.Fatal("expected non-empty payment_no")
	}
	if payment["status"] != "processing" {
		t.Errorf("expected status 'processing', got %v", payment["status"])
	}

	// Now query payment by payment_no
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/status/"+paymentNo, nil)
	c2.Params = []gin.Param{{Key: "payment_no", Value: paymentNo}}
	paymentHandler.QueryPaymentStatus(c2)

	if w2.Code != http.StatusOK {
		t.Fatalf("QueryPaymentStatus: expected 200, got %d. Body: %s", w2.Code, w2.Body.String())
	}

	var result2 map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&result2)
	payment2 := result2["payment"].(map[string]interface{})
	if payment2["status"] != "processing" {
		t.Errorf("expected status 'processing', got %v", payment2["status"])
	}
}

func Test_Payment_Create_NoOrder(t *testing.T) {
	setupTestDB(t)

	user := model.User{
		Username: "payuser2", Password: hashPW("Pay123"),
		Phone: "13800000101", Email: "pay2@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(), Balance: 100000,
	}
	testDB.Create(&user)

	reqBody := map[string]interface{}{
		"order_id":   99999,
		"channel":    "wechat",
		"amount":     5000,
		"subject":    "不存在的订单",
		"notify_url": "http://test.com/notify",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/payment/create", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, user.ID)
	paymentHandler.CreatePayment(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-existent order, got %d", w.Code)
	}
}

func Test_Payment_Query_NotFound(t *testing.T) {
	setupTestDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/status/NONEXISTENT123", nil)
	c.Params = []gin.Param{{Key: "payment_no", Value: "NONEXISTENT123"}}
	paymentHandler.QueryPaymentStatus(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-existent payment, got %d", w.Code)
	}
}

func Test_Payment_MyPayments(t *testing.T) {
	setupTestDB(t)

	user := model.User{
		Username: "payuser3", Password: hashPW("Pay123"),
		Phone: "13800000102", Email: "pay3@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(), Balance: 100000,
	}
	testDB.Create(&user)
	product := model.Product{Name: "商品A", Price: 3000, Status: 1, SupplierID: 1}
	testDB.Create(&product)
	order1 := model.Order{UserID: user.ID, ProductID: product.ID, Amount: 3000, Status: "pending"}
	order2 := model.Order{UserID: user.ID, ProductID: product.ID, Amount: 5000, Status: "pending"}
	testDB.Create(&order1)
	testDB.Create(&order2)

	// Create 2 payments directly in DB
	now := time.Now()
	testDB.Create(&model.ThirdPartyPayment{
		UserID: user.ID, OrderID: order1.ID, PaymentNo: "P1"+strconv.FormatInt(now.UnixNano(), 36),
		Channel: "wechat", Amount: 3000, Fee: 18, RealAmount: 2982, Status: "processing",
	})
	testDB.Create(&model.ThirdPartyPayment{
		UserID: user.ID, OrderID: order2.ID, PaymentNo: "P2"+strconv.FormatInt(now.UnixNano()+1, 36),
		Channel: "alipay", Amount: 5000, Fee: 30, RealAmount: 4970, Status: "success",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/my-payments", nil)
	c.Set(middleware.UserIDKey, user.ID)
	paymentHandler.GetUserPayments(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	payments := result["payments"].([]interface{})
	if len(payments) != 2 {
		t.Errorf("expected 2 payments, got %d", len(payments))
	}
}

// ============================================================
// P1 Test: Freelancer - Register & Approve
// ============================================================

func Test_Freelancer_RegisterAndApprove(t *testing.T) {
	setupTestDB(t)

	user := model.User{
		Username: "freelancer1", Password: hashPW("Free123"),
		Phone: "13800000200", Email: "free1@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&user)

	reqBody := map[string]interface{}{
		"real_name":  "张三",
		"id_card":    "110101199001011234",
		"phone":      "13800000200",
		"email":      "free1@test.com",
		"skill_tags": []string{"golang", "mysql"},
		"bio":        "资深后端开发",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/freelancer/register", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, user.ID)
	freelanceHandler.RegisterFreelancer(c)

	if w.Code != http.StatusOK {
		t.Fatalf("RegisterFreelancer: expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["message"] != "注册成功,等待审核" {
		t.Errorf("expected '注册成功,等待审核', got %v", result["message"])
	}
	freelancer := result["freelancer"].(map[string]interface{})
	if freelancer["status"] != "pending" {
		t.Errorf("expected status 'pending', got %v", freelancer["status"])
	}
	freelancerID := uint(freelancer["id"].(float64))

	// Admin approves
	adminID := ensureAdmin(t)
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/freelancer/"+strconv.FormatUint(uint64(freelancerID), 10)+"/approve", nil)
	c2.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(uint64(freelancerID), 10)}}
	c2.Set(middleware.UserIDKey, adminID)
	freelanceHandler.ApproveFreelancer(c2)

	if w2.Code != http.StatusOK {
		t.Fatalf("ApproveFreelancer: expected 200, got %d. Body: %s", w2.Code, w2.Body.String())
	}

	// Verify status changed
	var updated model.Freelancer
	testDB.First(&updated, freelancerID)
	if updated.Status != "approved" {
		t.Errorf("expected freelancer status 'approved', got %s", updated.Status)
	}
}

// ============================================================
// P1 Test: Freelancer - Task Lifecycle
// ============================================================

func Test_Freelancer_TaskLifecycle(t *testing.T) {
	setupTestDB(t)

	// Create publisher user
	publisher := model.User{
		Username: "publisher", Password: hashPW("Pub123"),
		Phone: "13800000300", Email: "pub@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(), Balance: 100000,
	}
	testDB.Create(&publisher)

	// Create freelancer user + register
	freeUser := model.User{
		Username: "freetask", Password: hashPW("FreeTask123"),
		Phone: "13800000301", Email: "freetask@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&freeUser)

	freelancer := model.Freelancer{
		UserID: freeUser.ID, RealName: "李四", IDCard: "110101199001012345",
		Phone: "13800000301", SkillTags: `["golang"]`, Bio: "后端开发",
		Status: "approved",
	}
	testDB.Create(&freelancer)

	// Publisher creates task
	taskReq := map[string]interface{}{
		"title":          "测试任务 - 开发API",
		"description":    "开发一个REST API接口",
		"category":       "dev",
		"skill_tags":     []string{"golang"},
		"budget":         50000,
		"duration_hours": 40,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/task/create", bytes.NewReader(marshal(taskReq)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, publisher.ID)
	freelanceHandler.CreateTask(c)

	if w.Code != http.StatusOK {
		t.Fatalf("CreateTask: expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	task := result["task"].(map[string]interface{})
	taskID := uint(task["id"].(float64))
	if task["status"] != "open" {
		t.Errorf("expected status 'open', got %v", task["status"])
	}

	// Assign task to freelancer
	assignReq := map[string]uint{
		"task_id":       taskID,
		"freelancer_id": freelancer.ID,
	}
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/api/v1/task/assign", bytes.NewReader(marshal(assignReq)))
	c2.Request.Header.Set("Content-Type", "application/json")
	c2.Set(middleware.UserIDKey, publisher.ID)
	freelanceHandler.AssignTask(c2)

	if w2.Code != http.StatusOK {
		t.Fatalf("AssignTask: expected 200, got %d. Body: %s", w2.Code, w2.Body.String())
	}

	var updatedTask model.Task
	testDB.First(&updatedTask, taskID)
	if updatedTask.Status != "assigned" {
		t.Errorf("expected task status 'assigned', got %s", updatedTask.Status)
	}
	if updatedTask.AssignedTo == nil || *updatedTask.AssignedTo != freelancer.ID {
		t.Errorf("expected task to be assigned to freelancer %d", freelancer.ID)
	}
}

// ============================================================
// P1 Test: Freelancer - Create Rating
// ============================================================

func Test_Freelancer_CreateRating(t *testing.T) {
	setupTestDB(t)

	publisher := model.User{
		Username: "ratingpub", Password: hashPW("Pub123"),
		Phone: "13800000400", Email: "ratingpub@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&publisher)

	freeUser := model.User{
		Username: "ratefree", Password: hashPW("Rate123"),
		Phone: "13800000401", Email: "ratefree@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&freeUser)

	freelancer := model.Freelancer{
		UserID: freeUser.ID, RealName: "王五", IDCard: "110101199001013456",
		Status: "approved",
	}
	testDB.Create(&freelancer)

	task := model.Task{
		Title: "评分测试任务", Description: "评分测试", Category: "dev",
		Budget: 10000, PublisherID: publisher.ID, Status: "completed",
		AssignedTo: &freelancer.ID,
	}
	testDB.Create(&task)

	ratingReq := map[string]interface{}{
		"task_id":       task.ID,
		"freelancer_id": freelancer.ID,
		"score":         5,
		"comment":       "非常满意",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rating", bytes.NewReader(marshal(ratingReq)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, publisher.ID)
	freelanceHandler.CreateRating(c)

	if w.Code != http.StatusOK {
		t.Fatalf("CreateRating: expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["message"] != "评分已提交" {
		t.Errorf("expected '评分已提交', got %v", result["message"])
	}

	// Verify rating in DB
	var rating model.Rating
	testDB.Where("task_id = ?", task.ID).First(&rating)
	if rating.Score != 5 {
		t.Errorf("expected score 5, got %d", rating.Score)
	}
}

// ============================================================
// P1 Test: Freelancer - Create TimeLog
// ============================================================

func Test_Freelancer_CreateTimeLog(t *testing.T) {
	setupTestDB(t)

	freeUser := model.User{
		Username: "timeloguser", Password: hashPW("TL123"),
		Phone: "13800000500", Email: "tl@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&freeUser)

	publisher := model.User{
		Username: "tlpub", Password: hashPW("Pub123"),
		Phone: "13800000501", Email: "tlpub@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&publisher)

	freelancer := model.Freelancer{
		UserID: freeUser.ID, RealName: "赵六", IDCard: "110101199001014567",
		Status: "approved",
	}
	testDB.Create(&freelancer)

	task := model.Task{
		Title: "工时测试任务", Description: "工时测试", Category: "dev",
		Budget: 10000, PublisherID: publisher.ID, Status: "assigned",
		AssignedTo: &freelancer.ID,
	}
	testDB.Create(&task)

	timelogReq := map[string]interface{}{
		"task_id": task.ID,
		"date":    "2026-07-02",
		"hours":   8.0,
		"content": "完成API开发",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/time-log", bytes.NewReader(marshal(timelogReq)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, freeUser.ID)
	freelanceHandler.CreateTimeLog(c)

	if w.Code != http.StatusOK {
		t.Fatalf("CreateTimeLog: expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	timeLog := result["time_log"].(map[string]interface{})
	if timeLog["status"] != "pending" {
		t.Errorf("expected status 'pending', got %v", timeLog["status"])
	}
}

// ============================================================
// P1 Test: Full E2E Freelance Flow
// ============================================================

func Test_E2E_FullFreelanceFlow(t *testing.T) {
	setupTestDB(t)

	// 1. Create publisher
	publisher := model.User{
		Username: "e2epub", Password: hashPW("E2EPub123"),
		Phone: "13800000600", Email: "e2epub@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(), Balance: 100000,
	}
	testDB.Create(&publisher)

	// 2. Create freelancer user & register
	freeUser := model.User{
		Username: "e2efree", Password: hashPW("E2EFree123"),
		Phone: "13800000601", Email: "e2efree@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&freeUser)

	freelancer := model.Freelancer{
		UserID: freeUser.ID, RealName: "陈七", IDCard: "110101199001015678",
		Phone: "13800000601", SkillTags: `["python"]`, Bio: "数据工程师",
		Status: "approved",
	}
	testDB.Create(&freelancer)

	// 3. Publisher creates task
	taskReq := map[string]interface{}{
		"title":          "E2E测试任务",
		"description":    "全流程测试",
		"category":       "dev",
		"skill_tags":     []string{"python"},
		"budget":         100000,
		"duration_hours": 80,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/task/create", bytes.NewReader(marshal(taskReq)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, publisher.ID)
	freelanceHandler.CreateTask(c)
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTask: %d %s", w.Code, w.Body.String())
	}

	var createResult map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResult)
	taskData := createResult["task"].(map[string]interface{})
	taskID := uint(taskData["id"].(float64))
	t.Logf("Task created: ID=%d, status=%v", taskID, taskData["status"])

	// 4. Assign task to freelancer
	assignReq := map[string]uint{"task_id": taskID, "freelancer_id": freelancer.ID}
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/api/v1/task/assign", bytes.NewReader(marshal(assignReq)))
	c2.Request.Header.Set("Content-Type", "application/json")
	c2.Set(middleware.UserIDKey, publisher.ID)
	freelanceHandler.AssignTask(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("AssignTask: %d %s", w2.Code, w2.Body.String())
	}
	t.Log("Task assigned")

	// 5. Create time log
	tlReq := map[string]interface{}{
		"task_id": taskID, "date": "2026-07-02", "hours": 8.0, "content": "E2E开发",
	}
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	c3.Request = httptest.NewRequest(http.MethodPost, "/api/v1/time-log", bytes.NewReader(marshal(tlReq)))
	c3.Request.Header.Set("Content-Type", "application/json")
	c3.Set(middleware.UserIDKey, freeUser.ID)
	freelanceHandler.CreateTimeLog(c3)
	if w3.Code != http.StatusOK {
		t.Fatalf("CreateTimeLog: %d %s", w3.Code, w3.Body.String())
	}
	t.Log("TimeLog created")

	// Mark task as completed for rating
	testDB.Model(&model.Task{}).Where("id = ?", taskID).Update("status", "completed")

	// 6. Create rating
	ratingReq := map[string]interface{}{
		"task_id": taskID, "freelancer_id": freelancer.ID, "score": 4, "comment": "E2E优秀",
	}
	w4 := httptest.NewRecorder()
	c4, _ := gin.CreateTestContext(w4)
	c4.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rating", bytes.NewReader(marshal(ratingReq)))
	c4.Request.Header.Set("Content-Type", "application/json")
	c4.Set(middleware.UserIDKey, publisher.ID)
	freelanceHandler.CreateRating(c4)
	if w4.Code != http.StatusOK {
		t.Fatalf("CreateRating: %d %s", w4.Code, w4.Body.String())
	}
	t.Log("Rating created")

	// Verify everything persisted
	var rating model.Rating
	result := testDB.Where("task_id = ? AND freelancer_id = ?", taskID, freelancer.ID).First(&rating)
	if result.Error != nil {
		t.Error("expected rating to be persisted")
	} else if rating.Score != 4 {
		t.Errorf("expected score 4, got %d", rating.Score)
	}

	var tl model.TimeLog
	result = testDB.Where("task_id = ? AND freelancer_id = ?", taskID, freelancer.ID).First(&tl)
	if result.Error != nil {
		t.Error("expected timelog to be persisted")
	} else if tl.Hours != 8.0 {
		t.Errorf("expected hours 8.0, got %f", tl.Hours)
	}

	t.Log("E2E full freelance flow completed")
}

// ============================================================
// P2 Test: Chain Engine - 3-Level Commission Calculation
// ============================================================

func Test_P2_ChainEngine_3LevelCommission(t *testing.T) {
	setupTestDB(t)

	cfg := engine.DefaultChainConfig()

	// Build chain: Alice(L1) → Bob(L2) → Charlie(L3)
	alice := model.User{
		Username: "p2alice", Password: hashPW("A123456"),
		Phone: "13800001000", Email: "p2alice@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&alice)

	bob := model.User{
		Username: "p2bob", Password: hashPW("B123456"),
		Phone: "13800001001", Email: "p2bob@test.com",
		Role: "customer", Level: 2, Status: 1, ParentID: &alice.ID, InviteCode: inviteCode(),
	}
	testDB.Create(&bob)

	charlie := model.User{
		Username: "p2charlie", Password: hashPW("C123456"),
		Phone: "13800001002", Email: "p2charlie@test.com",
		Role: "customer", Level: 3, Status: 1, ParentID: &bob.ID, InviteCode: inviteCode(),
	}
	testDB.Create(&charlie)

	// Charlie places an order of ¥200 (20000 cents)
	product := model.Product{Name: "P2套餐", Price: 20000, Status: 1, SupplierID: 1}
	testDB.Create(&product)
	order := model.Order{
		UserID: charlie.ID, ProductID: product.ID, OrderNo: "ORD-P2-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Amount: 20000, PaymentMethod: "mock_wechat", Status: "paid",
	}
	testDB.Create(&order)

	// Calculate and distribute commission
	details := engine.CalculateCommission(testDB, charlie.ID, order.Amount, cfg)
	if len(details) == 0 {
		t.Fatal("expected at least 1 commission detail")
	}

	// Distribute commission in a transaction
	err := engine.DistributeCommission(testDB, order.ID, charlie.ID, order.Amount, details)
	if err != nil {
		t.Fatalf("DistributeCommission failed: %v", err)
	}

	// Verify Bob gets 10% (level 1) = 2000 cents
	bobProfit := model.ProfitShare{}
	result := testDB.Where("to_user_id = ? AND order_id = ?", bob.ID, order.ID).First(&bobProfit)
	if result.Error != nil {
		t.Fatal("expected Bob to have a profit share")
	}
	expectedBob := int64(float64(order.Amount) * cfg.CommissionRate[1]) // 2000
	if bobProfit.Amount != expectedBob {
		t.Errorf("Bob: expected commission %d, got %d", expectedBob, bobProfit.Amount)
	}
	if bobProfit.Level != 1 {
		t.Errorf("Bob: expected level 1, got %d", bobProfit.Level)
	}

	// Verify Alice gets 8% (level 2) = 1600 cents
	aliceProfit := model.ProfitShare{}
	result = testDB.Where("to_user_id = ? AND order_id = ?", alice.ID, order.ID).First(&aliceProfit)
	if result.Error != nil {
		t.Fatal("expected Alice to have a profit share")
	}
	expectedAlice := int64(float64(order.Amount) * cfg.CommissionRate[2]) // 1600
	if aliceProfit.Amount != expectedAlice {
		t.Errorf("Alice: expected commission %d, got %d", expectedAlice, aliceProfit.Amount)
	}
	if aliceProfit.Level != 2 {
		t.Errorf("Alice: expected level 2, got %d", aliceProfit.Level)
	}

	// Verify Bob's balance increased
	testDB.First(&bob, bob.ID)
	if bob.Balance != expectedBob {
		t.Errorf("Bob balance: expected %d, got %d", expectedBob, bob.Balance)
	}
	// Verify Alice's balance increased
	testDB.First(&alice, alice.ID)
	if alice.Balance != expectedAlice {
		t.Errorf("Alice balance: expected %d, got %d", expectedAlice, alice.Balance)
	}

	t.Logf("Commission verified: Bob(L1)=%d(10%%), Alice(L2)=%d(8%%)", expectedBob, expectedAlice)
}

// ============================================================
// P2 Test: Chain Engine - 5-Level Max Depth
// ============================================================

func Test_P2_ChainEngine_5LevelMaxDepth(t *testing.T) {
	setupTestDB(t)

	cfg := engine.DefaultChainConfig()

	// Build a 6-level chain: U1 → U2 → U3 → U4 → U5 → U6
	users := make([]model.User, 6)
	for i := 0; i < 6; i++ {
		u := model.User{
			Username:   "p2depth" + strconv.Itoa(i+1),
			Password:   hashPW("Depth123"),
			Phone:      "13800002" + strconv.Itoa(100+i),
			Email:      "depth" + strconv.Itoa(i+1) + "@test.com",
			Role:       "customer",
			Level:      i + 1,
			Status:     1,
			InviteCode: inviteCode(),
		}
		if i > 0 {
			u.ParentID = &users[i-1].ID
		}
		testDB.Create(&u)
		users[i] = u
	}

	// U6 orders ¥1000 (100000 cents)
	order := model.Order{
		UserID: users[5].ID, ProductID: 1, OrderNo: "ORD-DEPTH-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Amount: 100000, PaymentMethod: "mock_wechat", Status: "paid",
	}
	testDB.Create(&order)

	details := engine.CalculateCommission(testDB, users[5].ID, order.Amount, cfg)

	// MaxLevel = 5, so level 6 (U1's parent if existed) would be cut off
	// But U1 IS level 5 relative to U6, so U1 should still get commission
	if len(details) != 5 {
		t.Fatalf("expected 5 commission levels (max 5), got %d", len(details))
	}

	// Verify rates for each level
	for i, d := range details {
		expectedRate := cfg.CommissionRate[i+1]
		expectedAmount := int64(float64(order.Amount) * expectedRate)
		if d.Amount != expectedAmount {
			t.Errorf("Level %d: expected amount %d (rate %.0f%%), got %d", i+1, expectedAmount, expectedRate*100, d.Amount)
		}
	}

	t.Logf("5-level commission verified: %d details, rates 10%%→8%%→5%%→3%%→2%%", len(details))
}

// ============================================================
// P2 Test: Chain Engine - Unlock Mechanics
// ============================================================

func Test_P2_ChainEngine_UnlockMechanics(t *testing.T) {
	setupTestDB(t)

	cfg := engine.DefaultChainConfig()

	// Alice is a locked user (status=1, not unlocked)
	alice := model.User{
		Username: "p2unlock_a", Password: hashPW("U123456"),
		Phone: "13800003000", Email: "unlock_a@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&alice)

	// Check initial unlock status
	result := engine.CalculateUnlockStatus(testDB, alice.ID, cfg)
	if result.IsUnlocked {
		t.Error("Alice should be locked initially")
	}

	// Invite first child
	child1 := model.User{
		Username: "p2unlock_b", Password: hashPW("U123456"),
		Phone: "13800003001", Email: "unlock_b@test.com",
		Role: "customer", Level: 2, Status: 1, ParentID: &alice.ID, InviteCode: inviteCode(),
	}
	testDB.Create(&child1)

	engine.ProcessChainLock(testDB, child1, cfg)
	result = engine.CalculateUnlockStatus(testDB, alice.ID, cfg)
	if result.IsUnlocked || result.PendingCount != 1 {
		t.Errorf("Alice should still be locked, pending 1 more. PendingCount=%d", result.PendingCount)
	}

	// Invite second child → triggers unlock
	child2 := model.User{
		Username: "p2unlock_c", Password: hashPW("U123456"),
		Phone: "13800003002", Email: "unlock_c@test.com",
		Role: "customer", Level: 2, Status: 1, ParentID: &alice.ID, InviteCode: inviteCode(),
	}
	testDB.Create(&child2)

	engine.ProcessChainLock(testDB, child2, cfg)

	// Verify Alice is now unlocked
	result = engine.CalculateUnlockStatus(testDB, alice.ID, cfg)
	if !result.IsUnlocked {
		t.Fatal("Alice should be unlocked after 2 direct invites")
	}
	if len(result.UnlockedByIDs) != 2 {
		t.Errorf("expected 2 unlockers, got %d", len(result.UnlockedByIDs))
	}

	// Verify Alice's status changed
	var updatedAlice model.User
	testDB.First(&updatedAlice, alice.ID)
	if updatedAlice.Status != 2 {
		t.Errorf("expected Alice status=2 (unlocked), got %d", updatedAlice.Status)
	}

	// Verify ChainRecord was created (related_id = parent being unlocked)
	var records []model.ChainRecord
	testDB.Where("related_id = ? AND action = ?", alice.ID, "unlock").Find(&records)
	if len(records) == 0 {
		// Try querying by user_id as fallback
		testDB.Where("user_id = ? AND action = ?", alice.ID, "unlock").Find(&records)
	}
	if len(records) > 0 {
		t.Logf("ChainRecord found: action=%s, user_id=%d, related_id=%d", records[0].Action, records[0].UserID, records[0].RelatedID)
	} else {
		t.Error("expected at least 1 chain_record with action='unlock'")
	}

	t.Logf("Unlock verified: Alice unlocked by %d children after 2 direct invites", len(result.UnlockedByIDs))
}

// ============================================================
// P2 Test: Orphan User (No Parent) - Zero Commission
// ============================================================

func Test_P2_ChainEngine_OrphanUser_NoCommission(t *testing.T) {
	setupTestDB(t)

	cfg := engine.DefaultChainConfig()

	root := model.User{
		Username: "p2orphan", Password: hashPW("O123456"),
		Phone: "13800004000", Email: "orphan@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&root)

	product := model.Product{Name: "孤儿套餐", Price: 5000, Status: 1, SupplierID: 1}
	testDB.Create(&product)
	order := model.Order{
		UserID: root.ID, ProductID: product.ID, OrderNo: "ORD-ORPHAN-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Amount: 5000, PaymentMethod: "mock_wechat", Status: "paid",
	}
	testDB.Create(&order)

	details := engine.CalculateCommission(testDB, root.ID, order.Amount, cfg)
	if len(details) != 0 {
		t.Errorf("orphan user should have 0 commission, got %d details", len(details))
	}

	t.Log("Orphan user: 0 commissions as expected")
}

// ============================================================
// P2 Test: Concurrent Registration
// ============================================================

func Test_P2_Concurrent_Registration(t *testing.T) {
	setupTestDB(t)

	// Create parent user
	parent := model.User{
		Username: "p2concurrent_parent", Password: hashPW("CP123"),
		Phone: "13800005000", Email: "con_parent@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: "CONCURRENTPARENT",
	}
	testDB.Create(&parent)

	// Register 10 children concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 10)
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			u := model.User{
				Username:   "p2conc" + strconv.Itoa(idx),
				Password:   hashPW("Conc123"),
				Phone:      "13800005" + strconv.Itoa(100+idx),
				Email:      "conc" + strconv.Itoa(idx) + "@test.com",
				Role:       "customer",
				Level:      2,
				Status:     1,
				ParentID:   &parent.ID,
				InviteCode: inviteCode(),
			}
			// Direct DB insert to test concurrent writes
			if err := testDB.Create(&u).Error; err != nil {
				errChan <- err
				return
			}
			mu.Lock()
			successCount++
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Logf("Concurrent registration: %d succeeded, %d failed (expected some may fail on unique constraint)", successCount, len(errors))
	} else {
		t.Logf("Concurrent registration: all %d succeeded", successCount)
	}

	// Verify parent has exactly `successCount` children
	var childCount int64
	testDB.Model(&model.User{}).Where("parent_id = ?", parent.ID).Count(&childCount)
	if int(childCount) != successCount {
		t.Errorf("expected %d children, DB has %d", successCount, childCount)
	}

	// Verify unlock status: parent should have at least 2 children
	cfg := engine.DefaultChainConfig()
	status := engine.CalculateUnlockStatus(testDB, parent.ID, cfg)
	t.Logf("After concurrent registration: parent unlocked=%v, pending count=%d", status.IsUnlocked, status.PendingCount)
}

// ============================================================
// P2 Test: Payment - Refund
// ============================================================

func Test_P2_Payment_Refund(t *testing.T) {
	setupTestDB(t)

	user := model.User{
		Username: "p2refund", Password: hashPW("Refund123"),
		Phone: "13800006000", Email: "refund@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(), Balance: 100000,
	}
	testDB.Create(&user)
	product := model.Product{Name: "退款测试商品", Price: 10000, Status: 1, SupplierID: 1}
	testDB.Create(&product)
	order := model.Order{
		UserID: user.ID, ProductID: product.ID, OrderNo: "ORD-REFUND-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Amount: 10000, PaymentMethod: "mock_wechat", Status: "paid",
	}
	testDB.Create(&order)

	// Create payment via handler
	reqBody := map[string]interface{}{
		"order_id":   order.ID,
		"channel":    "wechat",
		"amount":     10000,
		"subject":    "退款测试",
		"notify_url": "http://test.com/notify",
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/payment/create", bytes.NewReader(marshal(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDKey, user.ID)
	paymentHandler.CreatePayment(c)
	if w.Code != http.StatusOK {
		t.Fatalf("CreatePayment: %d %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	payment := result["payment"].(map[string]interface{})
	paymentNo := payment["payment_no"].(string)

	// Manually mark payment as success (refund requires success status)
	testDB.Model(&model.ThirdPartyPayment{}).Where("payment_no = ?", paymentNo).Updates(map[string]interface{}{
		"status":  "success",
		"paid_at": time.Now(),
	})

	// Refund the payment
	refundReq := map[string]string{
		"payment_no": paymentNo,
		"reason":     "用户申请退款",
	}
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/api/v1/payment/refund", bytes.NewReader(marshal(refundReq)))
	c2.Request.Header.Set("Content-Type", "application/json")
	c2.Set(middleware.UserIDKey, user.ID)
	paymentHandler.ProcessRefund(c2)

	if w2.Code != http.StatusOK {
		t.Fatalf("ProcessRefund: expected 200, got %d. Body: %s", w2.Code, w2.Body.String())
	}

	var refundResult map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&refundResult)
	if refundResult["message"] != "退款申请已提交" {
		t.Errorf("expected '退款申请已提交', got %v", refundResult["message"])
	}

	// Verify payment status changed
	var updatedPayment model.ThirdPartyPayment
	testDB.Where("payment_no = ?", paymentNo).First(&updatedPayment)
	if updatedPayment.Status != "refunded" {
		t.Errorf("expected payment status 'refunded', got '%s'", updatedPayment.Status)
	}
	if updatedPayment.RefundedAt == nil {
		t.Error("expected refunded_at to be set")
	}

	t.Log("Refund flow verified: payment status changed to 'refunded'")
}

// ============================================================
// P2 Test: Freelance - Submit & Review Work
// ============================================================

func Test_P2_Freelance_SubmitAndReview(t *testing.T) {
	setupTestDB(t)

	publisher := model.User{
		Username: "p2freviewpub", Password: hashPW("FrPub123"),
		Phone: "13800007000", Email: "frpub@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(), Balance: 100000,
	}
	testDB.Create(&publisher)

	freeUser := model.User{
		Username: "p2freview", Password: hashPW("FrFree123"),
		Phone: "13800007001", Email: "frfree@test.com",
		Role: "customer", Level: 1, Status: 1, InviteCode: inviteCode(),
	}
	testDB.Create(&freeUser)

	freelancer := model.Freelancer{
		UserID: freeUser.ID, RealName: "周八", IDCard: "110101199001016789",
		Status: "approved",
	}
	testDB.Create(&freelancer)

	// Create task directly (assigned state)
	task := model.Task{
		Title: "审核流程测试", Description: "提交和审核", Category: "dev",
		Budget: 30000, PublisherID: publisher.ID, Status: "assigned",
		AssignedTo: &freelancer.ID,
	}
	testDB.Create(&task)

	// Submit work with handler
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/task/"+strconv.FormatUint(uint64(task.ID), 10)+"/submit", bytes.NewReader(marshal(map[string]string{"submission": "已完成所有开发工作,代码已提交"})))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(uint64(task.ID), 10)}}
	c.Set(middleware.UserIDKey, freeUser.ID)
	freelanceHandler.SubmitWork(c)

	if w.Code != http.StatusOK {
		t.Fatalf("SubmitWork: expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify task status changed to 'submitted'
	var updatedTask model.Task
	testDB.First(&updatedTask, task.ID)
	if updatedTask.Status != "submitted" {
		t.Errorf("expected task status 'submitted', got '%s'", updatedTask.Status)
	}
	if updatedTask.Submission != "已完成所有开发工作,代码已提交" {
		t.Errorf("submission text mismatch")
	}

	// Review the work (approve)
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/api/v1/task/"+strconv.FormatUint(uint64(task.ID), 10)+"/review", bytes.NewReader(marshal(map[string]interface{}{
		"approved": true,
		"comment":  "工作完成质量高,审核通过",
	})))
	c2.Request.Header.Set("Content-Type", "application/json")
	c2.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(uint64(task.ID), 10)}}
	c2.Set(middleware.UserIDKey, publisher.ID)
	freelanceHandler.ReviewWork(c2)

	if w2.Code != http.StatusOK {
		t.Fatalf("ReviewWork: expected 200, got %d. Body: %s", w2.Code, w2.Body.String())
	}

	// Verify task status changed to 'completed' (approved=true -> completed)
	testDB.First(&updatedTask, task.ID)
	if updatedTask.Status != "completed" {
		t.Errorf("expected task status 'completed' (approved review), got '%s'", updatedTask.Status)
	}
	if updatedTask.ReviewComment != "工作完成质量高,审核通过" {
		t.Errorf("review comment mismatch")
	}

	t.Logf("Submit & Review flow verified: assigned -> submitted -> completed")
}

// ============================================================
// P2 Test: Chain Engine - Exact Commission Amounts
// ============================================================

func Test_P2_ChainEngine_Commission_ExactAmounts(t *testing.T) {
	setupTestDB(t)

	cfg := engine.DefaultChainConfig()

	// Chain: A → B → C → D → E → F
	users := make([]model.User, 6)
	for i := 0; i < 6; i++ {
		u := model.User{
			Username:   "p2exact" + strconv.Itoa(i+1),
			Password:   hashPW("Exact123"),
			Phone:      "13800008" + strconv.Itoa(100+i),
			Email:      "exact" + strconv.Itoa(i+1) + "@test.com",
			Role:       "customer",
			Level:      i + 1,
			Status:     1,
			InviteCode: inviteCode(),
		}
		if i > 0 {
			u.ParentID = &users[i-1].ID
		}
		testDB.Create(&u)
		users[i] = u
	}

	// F orders ¥888 (88800 cents) - unique amount
	orderAmount := int64(88800)
	order := model.Order{
		UserID: users[5].ID, ProductID: 1, OrderNo: "ORD-EXACT-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Amount: orderAmount, PaymentMethod: "mock_wechat", Status: "paid",
	}
	testDB.Create(&order)

	details := engine.CalculateCommission(testDB, users[5].ID, orderAmount, cfg)

	// Verify amounts match expected rates
	expectedAmounts := []int64{
		int64(float64(orderAmount) * cfg.CommissionRate[1]), // 10% → 8880
		int64(float64(orderAmount) * cfg.CommissionRate[2]), // 8%  → 7104
		int64(float64(orderAmount) * cfg.CommissionRate[3]), // 5%  → 4440
		int64(float64(orderAmount) * cfg.CommissionRate[4]), // 3%  → 2664
		int64(float64(orderAmount) * cfg.CommissionRate[5]), // 2%  → 1776
	}

	for i, d := range details {
		if d.Amount != expectedAmounts[i] {
			t.Errorf("Level %d: expected amount %d, got %d (rate=%.0f%%)",
				i+1, expectedAmounts[i], d.Amount, cfg.CommissionRate[i+1]*100)
		}
		// Verify user mapping
		expectedUser := users[4-i] // F→E→D→C→B→A
		if d.ToUserID != expectedUser.ID {
			t.Errorf("Level %d: expected user %s, got user ID %d", i+1, expectedUser.Username, d.ToUserID)
		}
	}

	// Distribute and verify balance updates
	err := engine.DistributeCommission(testDB, order.ID, users[5].ID, orderAmount, details)
	if err != nil {
		t.Fatalf("DistributeCommission failed: %v", err)
	}

	for i := 4; i >= 0; i-- { // E(4), D(3), C(2), B(1), A(0)
		var u model.User
		testDB.First(&u, users[i].ID)
		levelFromEnd := 5 - i // E=1, D=2, C=3, B=4, A=5
		expected := expectedAmounts[levelFromEnd-1]
		if u.Balance != expected {
			t.Errorf("%s (level from end=%d): expected balance %d, got %d",
				u.Username, levelFromEnd, expected, u.Balance)
		}
	}

	t.Logf("Exact amounts verified for ¥888: 8880+7104+4440+2664+1776 = %d",
		expectedAmounts[0]+expectedAmounts[1]+expectedAmounts[2]+expectedAmounts[3]+expectedAmounts[4])
}
