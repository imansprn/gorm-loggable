package loggable

import "gorm.io/gorm"

// Register attaches the plugin to the DB
func Register(db *gorm.DB, opts ...Option) (*Plugin, error) {
	p := New(opts...)
	return p, db.Use(p)
}
