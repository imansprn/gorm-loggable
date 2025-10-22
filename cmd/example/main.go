package main

import (
  "context"
  "fmt"

  loggable "github.com/imansprn/gorm-loggable"
  "gorm.io/driver/sqlite"
  "gorm.io/gorm"
)

type User struct {
  ID   uint
  Name string `gorm-loggable:"true"`
  loggable.LoggableModel
}

func (u User) Meta() any {
  return struct {
    Scope string
  }{
    Scope: "example",
  }
}

func main() {
  // Open GORM with sqlite (memory)
  db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
  if err != nil {
    panic(err)
  }

  // Register plugin FIRST
  _, err = loggable.Register(db,
    loggable.WithComputeDiff(),
    loggable.WithLazyUpdate("UpdatedAt"),
    loggable.WithTableName("change_logs"),
  )
  if err != nil {
    panic(err)
  }

  // Migrate models
  if err := db.AutoMigrate(&User{}, &loggable.ChangeLog{}); err != nil {
    panic(err)
  }

  // Attach actor to context
  ctx := loggable.WithActor(context.Background(), "demo-user")
  db = db.WithContext(ctx)

  // Perform some operations
  if err := db.Create(&User{Name: "John"}).Error; err != nil {
    panic(err)
  }
  if err := db.Model(&User{ID: 1}).Update("Name", "Jack").Error; err != nil {
    panic(err)
  }
  if err := db.Delete(&User{}, 1).Error; err != nil {
    panic(err)
  }

  // Read change logs
  var logs []loggable.ChangeLog
  if err := db.Order("id asc").Find(&logs).Error; err != nil {
    panic(err)
  }

  for _, l := range logs {
    fmt.Printf("[%s] table=%s pk=%s actor=%s diff=%v meta=%v\n", l.Operation, l.Table, l.PrimaryKey, l.Actor, l.RawDiff, l.Meta)
  }
}
