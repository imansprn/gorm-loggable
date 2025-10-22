package loggable

import (
	"fmt"
	"reflect"

	"gorm.io/gorm"
)

// Plugin implements gorm.Plugin for registration
type Plugin struct{ opts *Options }

func New(opts ...Option) *Plugin {
	merged := defaultOptions()
	for _, o := range opts {
		o(merged)
	}
	return &Plugin{opts: merged}
}

func (p *Plugin) Name() string { return "loggable" }

func (p *Plugin) Initialize(db *gorm.DB) error {
	// set table name if customized
	if p.opts.TableName != "" {
		changeLogTableName = p.opts.TableName
	}

	// auto migrate ChangeLog (after table name set)
	if err := db.AutoMigrate(&ChangeLog{}); err != nil {
		return fmt.Errorf("loggable: migrate ChangeLog: %w", err)
	}

	// register callbacks after core gorm callbacks
	createName := p.Name() + ":after_create"
	updateName := p.Name() + ":after_update"
	deleteName := p.Name() + ":after_delete"

	db.Callback().Create().After("gorm:create").Register(createName, p.afterCreate)
	db.Callback().Update().After("gorm:update").Register(updateName, p.afterUpdate)
	db.Callback().Delete().After("gorm:delete").Register(deleteName, p.afterDelete)

	return nil
}

func isLoggableModel(stmt *gorm.Statement) bool {
	if stmt == nil || stmt.Schema == nil || !stmt.ReflectValue.IsValid() {
		return false
	}

	// Skip ChangeLog table to prevent recursion
	if stmt.Table == changeLogTableName {
		return false
	}

	v := stmt.ReflectValue
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false
	}

	// Check if LoggableModel is embedded in the struct by looking at the struct type
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous && field.Type.Name() == "LoggableModel" {
			return true
		}
	}

	return false
}

func (p *Plugin) afterCreate(db *gorm.DB) {
	// Skip if we're already in a log creation operation to prevent recursion
	if db.Statement.Table == changeLogTableName {
		return
	}

	if !isLoggableModel(db.Statement) {
		return
	}

	// Use a new DB session without callbacks to prevent recursion
	dbWithoutCallbacks := db.Session(&gorm.Session{
		SkipHooks: true,
	})
	p.persistLog(dbWithoutCallbacks, OperationCreate, nil)
}

func (p *Plugin) afterUpdate(db *gorm.DB) {
	// Skip if we're already in a log creation operation to prevent recursion
	if db.Statement.Table == changeLogTableName {
		return
	}

	if !isLoggableModel(db.Statement) {
		return
	}

	// Use a new DB session without callbacks to prevent recursion
	dbWithoutCallbacks := db.Session(&gorm.Session{
		SkipHooks: true,
		NewDB:     true,
	})

	p.persistLog(dbWithoutCallbacks, OperationUpdate, nil)
}

func (p *Plugin) afterDelete(db *gorm.DB) {
	// Skip if we're already in a log creation operation to prevent recursion
	if db.Statement.Table == changeLogTableName {
		return
	}

	if !isLoggableModel(db.Statement) {
		return
	}

	// Use a new DB session without callbacks to prevent recursion
	dbWithoutCallbacks := db.Session(&gorm.Session{
		SkipHooks: true,
	})
	p.persistLog(dbWithoutCallbacks, OperationDelete, nil)
}

func (p *Plugin) persistLog(db *gorm.DB, op Operation, prev any) {
	stmt := db.Statement
	if stmt == nil || stmt.Schema == nil {
		return
	}

	// Skip if we're already in a log creation operation to prevent recursion
	if stmt.Table == changeLogTableName {
		return
	}

	// Build base log
	log := ChangeLog{
		Table:      stmt.Table,
		PrimaryKey: primaryKeyString(stmt),
		Operation:  op,
		Actor:      p.opts.ActorProvider(Scope{db: db}),
	}

	// Build RawObject from exported fields
	log.RawObject = toJSONMap(stmt)

	// Build Meta if model has Meta() any
	if meta := callMetaIfPresent(stmt); meta != nil {
		if m, ok := structToJSONB(meta); ok {
			log.Meta = m
		}
	}

	// Compute diff on update if enabled
	if op == OperationUpdate && p.opts.ComputeDiff {
		log.RawDiff = computeDiffFromTags(stmt)
	}

	// LazyUpdate skip if enabled and no significant change (only for update)
	if op == OperationUpdate && p.opts.LazyUpdate {
		if len(significantChanges(stmt, p.opts.LazyIgnoreFields)) == 0 {
			return
		}
	}

	// Use a new DB session without callbacks to prevent recursion
	dbWithoutCallbacks := db.Session(&gorm.Session{
		SkipHooks: true,
		NewDB:     true,
	})

	if err := dbWithoutCallbacks.Create(&log).Error; err != nil {
		db.Logger.Error(db.Statement.Context, "loggable: persist error: %v", err)
	}
}
