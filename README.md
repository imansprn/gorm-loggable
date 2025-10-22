# gorm-loggable

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue.svg)](https://golang.org/)
[![GORM Version](https://img.shields.io/badge/gorm-v2-green.svg)](https://gorm.io/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/gobliggg/gorm-loggable)](https://goreportcard.com/report/github.com/gobliggg/gorm-loggable)

A GORM v2 plugin for automatic audit logging. Track model changes in a `change_logs` table with minimal setup.

## Features

- 🔍 **Automatic Tracking**: Records create, update, delete operations
- 🏷️ **Selective Diffing**: Store only fields tagged with `gorm-loggable:"true"`
- ⚡ **Lazy Updates**: Skip logs when only ignored fields change
- 👤 **Actor Attribution**: Track who made changes via context or custom provider
- 🎯 **Zero Config**: Works out of the box with sensible defaults

## Installation

```bash
go get github.com/gobliggg/gorm-loggable
```

## Quick start

```go
package main

import (
	"context"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	loggable "github.com/gobliggg/gorm-loggable"
)

type User struct {
	ID   uint
	Name string `gorm-loggable:"true"`
	loggable.LoggableModel
}

func main() {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_, _ = loggable.Register(db,
		loggable.WithComputeDiff(),
		loggable.WithLazyUpdate("UpdatedAt"),
	)
	_ = db.AutoMigrate(&User{}, &loggable.ChangeLog{})

	ctx := loggable.WithActor(context.Background(), "user-123")
	db = db.WithContext(ctx)

	db.Create(&User{Name: "John"})
	db.Model(&User{ID: 1}).Update("Name", "Jack")
	db.Delete(&User{}, 1)
}
```

## Options
- WithComputeDiff: stores only changed tagged fields into `ChangeLog.RawDiff`
- WithLazyUpdate(ignoreFields...): skips logging updates that only change ignored fields
- WithTableName(name): custom table name for `ChangeLog`
- WithActorProvider(func(Scope) string): custom actor attribution source

## Embedding and Meta
Embed `LoggableModel` to opt-in. Optionally implement `Meta() any` on your model to attach additional metadata to each log entry.

```go
type Account struct {
	ID   uint
	Org  string
	loggable.LoggableModel
}

func (a Account) Meta() any {
	return struct{ Org string }{Org: a.Org}
}
```

## Table
Default table is `change_logs`. Override via `WithTableName("my_change_logs")`.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by the original [gorm-loggable](https://github.com/sas1024/gorm-loggable) plugin
- Built for GORM v2 compatibility
- Tested with SQLite, should work with other GORM-supported databases

