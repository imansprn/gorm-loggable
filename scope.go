package loggable

import "gorm.io/gorm"

// Scope wraps *gorm.DB and provides helpers used internally
type Scope struct{ db *gorm.DB }

func (s Scope) DB() *gorm.DB { return s.db }


