package jsonpatch

// ApplyOptions configures the behaviour of patch application.
type ApplyOptions struct {
	// AllowMissingPathOnRemove treats a missing target path on "remove"
	// as a no-op instead of returning an error. This is the most commonly
	// requested permissive behaviour (mirrors evanphx/json-patch).
	AllowMissingPathOnRemove bool

	// EnsurePathExistsOnAdd auto-creates intermediate objects along the path
	// when applying an "add" operation, instead of failing if a parent does
	// not exist.
	EnsurePathExistsOnAdd bool
}

// Option is a functional option for ApplyPatchWithOptions / ApplyWithOptions.
type Option func(*ApplyOptions)

// WithAllowMissingPathOnRemove returns an Option that silently ignores
// "remove" operations whose target path does not exist.
func WithAllowMissingPathOnRemove() Option {
	return func(o *ApplyOptions) {
		o.AllowMissingPathOnRemove = true
	}
}

// WithEnsurePathExistsOnAdd returns an Option that auto-creates intermediate
// objects when applying an "add" operation.
func WithEnsurePathExistsOnAdd() Option {
	return func(o *ApplyOptions) {
		o.EnsurePathExistsOnAdd = true
	}
}

// defaultOptions returns the zero-value options (strict RFC 6902 semantics).
func defaultOptions() ApplyOptions {
	return ApplyOptions{}
}

// buildOptions folds a list of functional options into an ApplyOptions value.
func buildOptions(opts []Option) ApplyOptions {
	o := defaultOptions()
	for _, fn := range opts {
		fn(&o)
	}
	return o
}
