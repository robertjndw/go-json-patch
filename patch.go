// Package jsonpatch implements RFC 6902 JSON Patch operations.
//
// JSON Patch defines a JSON document structure for expressing a sequence of
// operations to apply to a JSON document. This package provides two main
// capabilities:
//
//   - Apply: Apply a JSON Patch document to a target JSON document
//   - CreatePatch: Generate a JSON Patch by comparing two JSON documents (diff)
//
// Usage:
//
//	// Apply a patch
//	patched, err := jsonpatch.Apply(originalJSON, patchJSON)
//
//	// Create a patch by comparing two documents
//	patch, err := jsonpatch.CreatePatch(originalJSON, modifiedJSON)
package jsonpatch

import (
	"encoding/json"
	"fmt"
)

// OpType represents the type of JSON Patch operation.
type OpType string

const (
	// OpAdd represents the "add" operation.
	OpAdd OpType = "add"
	// OpRemove represents the "remove" operation.
	OpRemove OpType = "remove"
	// OpReplace represents the "replace" operation.
	OpReplace OpType = "replace"
	// OpMove represents the "move" operation.
	OpMove OpType = "move"
	// OpCopy represents the "copy" operation.
	OpCopy OpType = "copy"
	// OpTest represents the "test" operation.
	OpTest OpType = "test"
)

// operationCache holds pre-parsed fields for an Operation.
// It is heap-allocated only when the operation has been validated via
// Validate or DecodePatch, so operations produced internally by CreatePatch
// carry zero extra overhead.
type operationCache struct {
	parsedPath    Pointer
	parsedFrom    Pointer
	parsedValue   interface{}
	parsedValueOK bool // true once parsedValue is set (distinguishes cached-nil from not-yet-cached)
}

// Operation represents a single JSON Patch operation as defined in RFC 6902.
type Operation struct {
	// Op is the operation to perform. It MUST be one of "add", "remove",
	// "replace", "move", "copy", or "test".
	Op OpType `json:"op"`

	// Path is a JSON Pointer (RFC 6901) string that references the target
	// location where the operation is performed.
	Path string `json:"path"`

	// Value specifies the value to be used by the operation.
	// Required for "add", "replace", and "test" operations.
	Value *json.RawMessage `json:"value,omitempty"`

	// From is a JSON Pointer string that references the source location.
	// Required for "move" and "copy" operations.
	From string `json:"from,omitempty"`

	// hasPath tracks whether the "path" key was present in the original JSON.
	hasPath bool

	// hasFrom tracks whether the "from" key was present in the original JSON,
	// distinguishing between an absent key and an explicit empty string (root pointer).
	hasFrom bool

	// hasValue tracks whether the "value" key was present in the original JSON,
	// distinguishing between an absent key and an explicit null.
	hasValue bool

	// cache holds pre-parsed and pre-decoded fields when populated via
	// Validate or DecodePatch. It is nil for operations that have not been
	// validated (e.g., ops built by CreatePatch).
	cache *operationCache
}

// UnmarshalJSON implements custom JSON unmarshaling for Operation to properly
// distinguish between an absent "value" field and a "value" field set to null.
func (o *Operation) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a raw map to detect key presence
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// rejectNull returns an error if the raw JSON value is "null".
	// This prevents json.Unmarshal from silently accepting null into a string.
	rejectNull := func(raw json.RawMessage, field string) error {
		if string(raw) == "null" {
			return fmt.Errorf("invalid %q field: must be a string", field)
		}
		return nil
	}

	if opRaw, ok := raw["op"]; ok {
		if err := rejectNull(opRaw, "op"); err != nil {
			return err
		}
		var op string
		if err := json.Unmarshal(opRaw, &op); err != nil {
			return fmt.Errorf("invalid \"op\" field: must be a string")
		}
		o.Op = OpType(op)
	}

	if pathRaw, ok := raw["path"]; ok {
		o.hasPath = true
		if err := rejectNull(pathRaw, "path"); err != nil {
			return err
		}
		var path string
		if err := json.Unmarshal(pathRaw, &path); err != nil {
			return fmt.Errorf("invalid \"path\" field: must be a string")
		}
		o.Path = path
	}

	if fromRaw, ok := raw["from"]; ok {
		o.hasFrom = true
		if err := rejectNull(fromRaw, "from"); err != nil {
			return err
		}
		var from string
		if err := json.Unmarshal(fromRaw, &from); err != nil {
			return fmt.Errorf("invalid \"from\" field: must be a string")
		}
		o.From = from
	}

	if valRaw, ok := raw["value"]; ok {
		o.hasValue = true
		v := json.RawMessage(valRaw)
		o.Value = &v
	}

	return nil
}

// HasValue reports whether the operation has a "value" field
// (including explicit null).
func (o Operation) HasValue() bool {
	return o.hasValue || o.Value != nil
}

// HasFrom reports whether the operation has a "from" field
// (including an explicit empty string meaning root pointer).
func (o Operation) HasFrom() bool {
	return o.hasFrom
}

// Validate validates the operation, checking that all required fields are
// present and that pointer strings are well-formed. It also caches parsed
// pointers and values for efficient subsequent application.
//
// For struct-literal operations (not created via NewOperation / DecodePatch),
// Validate infers field presence: if Op is a recognised operation type then
// hasPath is assumed true (root "" is a valid path for all RFC 6902 ops);
// hasFrom is inferred when From is non-empty or Op is move/copy.
func (o *Operation) Validate() error {
	// Infer hasPath for recognised ops when not already set (struct literal).
	if !o.hasPath && o.Op != "" {
		switch o.Op {
		case OpAdd, OpRemove, OpReplace, OpMove, OpCopy, OpTest:
			o.hasPath = true
		}
	}
	// Infer hasFrom when From is populated or the op requires it.
	if !o.hasFrom {
		if o.From != "" {
			o.hasFrom = true
		} else if o.Op == OpMove || o.Op == OpCopy {
			// Empty from ("") is the root pointer — mark as explicitly set
			// only if the caller actually set From to "".
			// We can't distinguish uninitialised "" from intentional "" on a
			// plain struct literal, so for move/copy we assume it's set.
			o.hasFrom = true
		}
	}
	// Infer hasValue when Value is non-nil.
	if !o.hasValue && o.Value != nil {
		o.hasValue = true
	}
	return validateAndCacheOperation(o)
}

// Patch represents a JSON Patch document — an ordered list of operations.
type Patch []Operation

// NewOperation creates a new Operation with the given parameters.
// Pass a non-nil pointer to indicate the value is present (including JSON null).
// To create an operation without a value (e.g., remove), pass nil.
func NewOperation(op OpType, path string, value interface{}) (Operation, error) {
	o := Operation{
		Op:       op,
		Path:     path,
		hasPath:  true,
		hasValue: true,
	}
	// Always marshal the value — json.Marshal(nil) produces "null", which is valid.
	b, err := json.Marshal(value)
	if err != nil {
		return Operation{}, fmt.Errorf("failed to marshal value: %w", err)
	}
	raw := json.RawMessage(b)
	o.Value = &raw
	return o, nil
}

// NewMoveOperation creates a new move Operation.
func NewMoveOperation(from, path string) Operation {
	return Operation{
		Op:      OpMove,
		Path:    path,
		From:    from,
		hasPath: true,
		hasFrom: true,
	}
}

// NewCopyOperation creates a new copy Operation.
func NewCopyOperation(from, path string) Operation {
	return Operation{
		Op:      OpCopy,
		Path:    path,
		From:    from,
		hasPath: true,
		hasFrom: true,
	}
}

// NewRemoveOperation creates a new remove Operation.
func NewRemoveOperation(path string) Operation {
	return Operation{
		Op:      OpRemove,
		Path:    path,
		hasPath: true,
	}
}

// GetValue returns the operation's value. If the value has been pre-cached
// (e.g. via DecodePatch or a previous apply), the cached value is returned
// directly; otherwise it is parsed from the raw JSON and, when a cache is
// present, stored for future calls (lazy caching).
func (o *Operation) GetValue() (interface{}, error) {
	if !o.HasValue() {
		return nil, fmt.Errorf("operation has no value")
	}
	if o.cache != nil {
		if o.cache.parsedValueOK {
			return o.cache.parsedValue, nil
		}
		// Lazy-cache the value so repeated apply calls don't re-unmarshal.
		var v interface{}
		if err := json.Unmarshal(*o.Value, &v); err != nil {
			return nil, fmt.Errorf("failed to unmarshal value: %w", err)
		}
		o.cache.parsedValue = v
		o.cache.parsedValueOK = true
		return v, nil
	}
	var v interface{}
	if err := json.Unmarshal(*o.Value, &v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}
	return v, nil
}

// DecodePatch parses a JSON Patch document from raw JSON bytes.
func DecodePatch(patchJSON []byte) (Patch, error) {
	var patch Patch
	if err := json.Unmarshal(patchJSON, &patch); err != nil {
		return nil, fmt.Errorf("failed to decode patch document: %w", err)
	}

	// Validate and cache parsed pointer fields for each operation.
	// Value fields are cached lazily on the first GetValue call.
	for i := range patch {
		if err := validateAndCachePointersOnly(&patch[i]); err != nil {
			return nil, fmt.Errorf("invalid operation at index %d: %w", i, err)
		}
	}

	return patch, nil
}

// MarshalPatch serializes a Patch to JSON bytes.
func MarshalPatch(patch Patch) ([]byte, error) {
	return json.Marshal(patch)
}

// validateOperation checks that an operation has the required fields.
// It does not cache parsed results; use validateAndCacheOperation for that.
func validateOperation(op Operation) error {
	return validateAndCacheOperation(&op)
}

// validateAndCacheOperation validates the operation and caches parsed pointers
// and the unmarshaled value so that apply* functions can reuse them.
func validateAndCacheOperation(op *Operation) error {
	// All operations MUST have exactly one "op" member (RFC 6902 Section 4).
	if op.Op == "" {
		return fmt.Errorf("operation must contain a non-empty \"op\" member")
	}

	// All operations MUST have a "path" member (RFC 6902 Section 4).
	if !op.hasPath {
		return fmt.Errorf("%q operation must contain a \"path\" member", op.Op)
	}

	var pathPtr Pointer
	var fromPtr Pointer
	var err error

	switch op.Op {
	case OpAdd, OpReplace, OpTest:
		if !op.HasValue() {
			return fmt.Errorf("%q operation must contain a \"value\" member", op.Op)
		}
		pathPtr, err = ParsePointer(op.Path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
	case OpRemove:
		pathPtr, err = ParsePointer(op.Path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
	case OpMove, OpCopy:
		if !op.hasFrom {
			return fmt.Errorf("%q operation must contain a \"from\" member", op.Op)
		}
		pathPtr, err = ParsePointer(op.Path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
		fromPtr, err = ParsePointer(op.From)
		if err != nil {
			return fmt.Errorf("invalid from: %w", err)
		}
	default:
		return fmt.Errorf("unknown operation %q", op.Op)
	}

	// Allocate cache and populate parsed pointers and the unmarshaled value.
	c := &operationCache{
		parsedPath: pathPtr,
		parsedFrom: fromPtr,
	}
	if op.HasValue() {
		var v interface{}
		if err := json.Unmarshal(*op.Value, &v); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}
		c.parsedValue = v
		c.parsedValueOK = true
	}
	op.cache = c
	return nil
}

// validateAndCachePointersOnly validates the operation and caches only the
// parsed Pointer values. The value field is NOT eagerly unmarshalled; it will
// be lazily parsed on the first GetValue call. This keeps DecodePatch cheap
// while preserving apply performance for repeated Apply on the same Patch.
func validateAndCachePointersOnly(op *Operation) error {
	if op.Op == "" {
		return fmt.Errorf("operation must contain a non-empty \"op\" member")
	}
	if !op.hasPath {
		return fmt.Errorf("%q operation must contain a \"path\" member", op.Op)
	}

	var pathPtr Pointer
	var fromPtr Pointer
	var err error

	switch op.Op {
	case OpAdd, OpReplace, OpTest:
		if !op.HasValue() {
			return fmt.Errorf("%q operation must contain a \"value\" member", op.Op)
		}
		pathPtr, err = ParsePointer(op.Path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
	case OpRemove:
		pathPtr, err = ParsePointer(op.Path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
	case OpMove, OpCopy:
		if !op.hasFrom {
			return fmt.Errorf("%q operation must contain a \"from\" member", op.Op)
		}
		pathPtr, err = ParsePointer(op.Path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
		fromPtr, err = ParsePointer(op.From)
		if err != nil {
			return fmt.Errorf("invalid from: %w", err)
		}
	default:
		return fmt.Errorf("unknown operation %q", op.Op)
	}

	op.cache = &operationCache{
		parsedPath: pathPtr,
		parsedFrom: fromPtr,
	}
	return nil
}
