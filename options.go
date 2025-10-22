package loggable

// Options customize plugin behavior
type Options struct {
	// LazyUpdate enables saving only when significant fields changed
	LazyUpdate bool
	// LazyIgnoreFields are field names to ignore when comparing for LazyUpdate
	LazyIgnoreFields []string

	// ComputeDiff stores only the changed fields into RawDiff during updates
	ComputeDiff bool

	// TableName allows overriding the default table name for ChangeLog
	TableName string

	// ActorProvider returns the actor identifier from the current context/DB
	ActorProvider func(scope Scope) string
}

// Option is a function to mutate Options
type Option func(*Options)

// WithLazyUpdate enables LazyUpdate with a set of ignored fields
func WithLazyUpdate(ignoreFields ...string) Option {
	return func(o *Options) {
		o.LazyUpdate = true
		o.LazyIgnoreFields = append([]string(nil), ignoreFields...)
	}
}

// WithComputeDiff enables ComputeDiff behavior
func WithComputeDiff() Option {
	return func(o *Options) { o.ComputeDiff = true }
}

// WithTableName sets custom table name for ChangeLog
func WithTableName(name string) Option { return func(o *Options) { o.TableName = name } }

// WithActorProvider sets a custom actor provider
func WithActorProvider(p func(scope Scope) string) Option { return func(o *Options) { o.ActorProvider = p } }

func (o *Options) clone() *Options {
	c := *o
	c.LazyIgnoreFields = append([]string(nil), o.LazyIgnoreFields...)
	return &c
}

func defaultOptions() *Options {
	return &Options{
		TableName: "change_logs",
		ActorProvider: func(scope Scope) string {
			return CurrentActor(scope.DB())
		},
	}
}


