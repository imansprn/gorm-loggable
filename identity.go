package loggable

import (
	"context"

	"gorm.io/gorm"
)

type ctxKey string

const actorKey ctxKey = "loggable-actor"

// WithActor injects an actor identifier into context for attribution
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorKey, actor)
}

// CurrentActor reads actor from the DB's context if present
func CurrentActor(db *gorm.DB) string {
	if db == nil || db.Statement == nil || db.Statement.Context == nil {
		return ""
	}
	if v := db.Statement.Context.Value(actorKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
