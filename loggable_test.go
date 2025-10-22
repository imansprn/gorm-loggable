package loggable

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TUser struct {
	ID   uint
	Name string `gorm-loggable:"true"`
	LoggableModel
}

func TestCreateUpdateDelete(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Enable debug mode
	db = db.Debug()

	_, err = Register(db, WithComputeDiff())
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Verify table exists after migration
	if err := db.AutoMigrate(&TUser{}, &ChangeLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Check if tables exist
	if !db.Migrator().HasTable(&TUser{}) {
		t.Fatalf("TUser table does not exist")
	}
	if !db.Migrator().HasTable(&ChangeLog{}) {
		t.Fatalf("ChangeLog table does not exist")
	}

	ctx := WithActor(context.Background(), "tester")
	db = db.WithContext(ctx)

	// Create a user
	user := &TUser{Name: "John"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	// Print user ID for debugging
	t.Logf("Created user with ID: %v", user.ID)

	var logs []ChangeLog
	if err := db.Find(&logs).Error; err != nil {
		t.Fatalf("find logs: %v", err)
	}

	// Debug output
	t.Logf("Found %d logs", len(logs))
	for i, log := range logs {
		t.Logf("Log %d: op=%s table=%s pk=%s", i, log.Operation, log.Table, log.PrimaryKey)
	}

	if len(logs) != 1 || logs[0].Operation != OperationCreate {
		t.Fatalf("expected one create log, got: %+v", logs)
	}

	if err := db.Model(&TUser{ID: 1}).Update("Name", "Jack").Error; err != nil {
		t.Fatalf("update: %v", err)
	}
	logs = nil
	if err := db.Order("id asc").Find(&logs).Error; err != nil {
		t.Fatalf("find logs: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least two logs, got: %d", len(logs))
	}
	if logs[len(logs)-1].Operation != OperationUpdate {
		t.Fatalf("expected last log update, got: %s", logs[len(logs)-1].Operation)
	}
	if logs[len(logs)-1].RawDiff == nil || logs[len(logs)-1].RawDiff["Name"] != "Jack" {
		t.Fatalf("expected diff with Name=Jack, got: %+v", logs[len(logs)-1].RawDiff)
	}

	if err := db.Delete(&TUser{}, 1).Error; err != nil {
		t.Fatalf("delete: %v", err)
	}
	logs = nil
	if err := db.Order("id asc").Find(&logs).Error; err != nil {
		t.Fatalf("find logs: %v", err)
	}
	if logs[len(logs)-1].Operation != OperationDelete {
		t.Fatalf("expected last log delete, got: %s", logs[len(logs)-1].Operation)
	}
}
