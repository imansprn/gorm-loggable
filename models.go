package loggable

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Operation represents the kind of change recorded
type Operation string

const (
	OperationCreate Operation = "create"
	OperationUpdate Operation = "update"
	OperationDelete Operation = "delete"
)

// JSONB provides a convenient type to marshal/unmarshal arbitrary payloads
type JSONB map[string]any

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (j *JSONB) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, j)
	}
}

// ChangeLog is the audit log entry persisted by the plugin
type ChangeLog struct {
	ID         uint      `gorm:"primaryKey"`
	CreatedAt  time.Time `gorm:"index"`
	Table      string    `gorm:"size:255;index"`
	PrimaryKey string    `gorm:"size:255;index"`
	Operation  Operation `gorm:"size:16;index"`
	Actor      string    `gorm:"size:255;index"`
	RawObject  JSONB     `gorm:"type:json"`
	Meta       JSONB     `gorm:"type:json"`
	RawDiff    JSONB     `gorm:"type:json"`
}

var changeLogTableName = "change_logs"

func (ChangeLog) TableName() string { return changeLogTableName }

// LoggableModel is embedded in models to opt-in to logging and to allow Meta override
type LoggableModel struct{}

// Meta returns an optional metadata payload that will be stored alongside the change
func (LoggableModel) Meta() any { return nil }
