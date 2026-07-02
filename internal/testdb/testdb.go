// Package database provides a testable DB wrapper
package database

import (
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/linqi/chain2plus1/pkg/model"

	// CGO-free SQLite driver for GORM
	"github.com/glebarez/sqlite"
)

var (
	testDB     *gorm.DB
	testDBOnce sync.Once
	testErr    error
)

// InitTestDB initializes an in-memory SQLite database with all models
func InitTestDB() (*gorm.DB, error) {
	testDBOnce.Do(func() {
		database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Discard, // Silence test SQL logs
		})
		if err != nil {
			testErr = err
			return
		}

		// Auto-migrate all models
		err = database.AutoMigrate(
			&model.User{},
			&model.Product{},
			&model.Supplier{},
			&model.Order{},
			&model.ProfitShare{},
			&model.Withdraw{},
			&model.ThirdPartyPayment{},
			&model.ChainRecord{},
			&model.Freelancer{},
			&model.Task{},
			&model.TimeLog{},
			&model.Settlement{},
			&model.Rating{},
		&model.AuditLog{},
		)
		testErr = err
		testDB = database
	})
	return testDB, testErr
}

// ResetTestDB drops and recreates all tables
func ResetTestDB(db *gorm.DB) error {
	db.Migrator().DropTable(
		&model.User{}, &model.Product{}, &model.Supplier{},
		&model.Order{}, &model.ProfitShare{}, &model.ChainRecord{},
		&model.Withdraw{}, &model.ThirdPartyPayment{},
		&model.Freelancer{}, &model.Task{}, &model.TimeLog{},
		&model.Settlement{}, &model.Rating{},
		&model.AuditLog{},
	)
	return InitTestDBInternal(db)
}

func InitTestDBInternal(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Product{},
		&model.Supplier{},
		&model.Order{},
		&model.ProfitShare{},
		&model.Withdraw{},
		&model.ThirdPartyPayment{},
		&model.ChainRecord{},
		&model.Freelancer{},
		&model.Task{},
		&model.TimeLog{},
		&model.Settlement{},
		&model.Rating{},
		&model.AuditLog{},
	)
}
