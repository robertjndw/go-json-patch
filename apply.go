package jsonpatch

import (
	"encoding/json"
	"fmt"
)

// Apply applies a JSON Patch document (as raw JSON bytes) to a target JSON
// document (as raw JSON bytes). It returns the patched document as raw JSON
// bytes. Operations are applied sequentially; if any operation fails, the
// entire patch is aborted and an error is returned (atomic semantics per
// RFC 5789).
func Apply(docJSON, patchJSON []byte) ([]byte, error) {
	patch, err := DecodePatch(patchJSON)
	if err != nil {
		return nil, err
	}
	return ApplyPatch(docJSON, patch)
}

// ApplyWithOptions is like Apply but accepts functional options.
func ApplyWithOptions(docJSON, patchJSON []byte, opts ...Option) ([]byte, error) {
	patch, err := DecodePatch(patchJSON)
	if err != nil {
		return nil, err
	}
	return ApplyPatchWithOptions(docJSON, patch, opts...)
}

// ApplyPatch applies a decoded Patch to a target JSON document (as raw JSON bytes).
// It returns the patched document as raw JSON bytes.
func ApplyPatch(docJSON []byte, patch Patch) ([]byte, error) {
	return applyPatchInternal(docJSON, patch, defaultOptions())
}

// ApplyPatchWithOptions is like ApplyPatch but accepts functional options.
func ApplyPatchWithOptions(docJSON []byte, patch Patch, opts ...Option) ([]byte, error) {
	return applyPatchInternal(docJSON, patch, buildOptions(opts))
}

// applyPatchInternal is the shared implementation for ApplyPatch and ApplyPatchWithOptions.
func applyPatchInternal(docJSON []byte, patch Patch, opts ApplyOptions) ([]byte, error) {
	var doc interface{}
	if err := json.Unmarshal(docJSON, &doc); err != nil {
		return nil, fmt.Errorf("failed to decode target document: %w", err)
	}

	var err error
	for i, op := range patch {
		doc, err = applyOperation(doc, op, opts)
		if err != nil {
			return nil, &InvalidOperationError{
				Index: i,
				Op:    op.Op,
				Path:  op.Path,
				Cause: err,
			}
		}
	}

	result, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	return result, nil
}

// applyOperation applies a single operation to the document.
func applyOperation(doc interface{}, op Operation, opts ApplyOptions) (interface{}, error) {
	switch op.Op {
	case OpAdd:
		return applyAdd(doc, op, opts)
	case OpRemove:
		return applyRemove(doc, op, opts)
	case OpReplace:
		return applyReplace(doc, op, opts)
	case OpMove:
		return applyMove(doc, op, opts)
	case OpCopy:
		return applyCopy(doc, op, opts)
	case OpTest:
		return applyTest(doc, op, opts)
	default:
		return nil, fmt.Errorf("unknown operation %q", op.Op)
	}
}

// applyAdd implements the "add" operation (Section 4.1).
func applyAdd(doc interface{}, op Operation, opts ApplyOptions) (interface{}, error) {
	path := op.parsedPath
	if !op.parsed {
		var err error
		path, err = ParsePointer(op.Path)
		if err != nil {
			return nil, err
		}
	}

	value, err := op.GetValue()
	if err != nil {
		return nil, err
	}

	if opts.EnsurePathExistsOnAdd {
		doc = ensurePathExists(doc, path)
	}

	return path.Set(doc, value)
}

// applyRemove implements the "remove" operation (Section 4.2).
func applyRemove(doc interface{}, op Operation, opts ApplyOptions) (interface{}, error) {
	path := op.parsedPath
	if !op.parsed {
		var err error
		path, err = ParsePointer(op.Path)
		if err != nil {
			return nil, err
		}
	}

	result, err := path.Remove(doc)
	if err != nil && opts.AllowMissingPathOnRemove {
		// Check if it's a path-not-found error; if so, treat as no-op.
		if isPathNotFound(err) {
			return doc, nil
		}
	}
	return result, err
}

// isPathNotFound reports whether err (or any wrapped error) is a PathNotFoundError.
func isPathNotFound(err error) bool {
	for e := err; e != nil; {
		if _, ok := e.(*PathNotFoundError); ok {
			return true
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

// applyReplace implements the "replace" operation (Section 4.3).
// Functionally identical to a "remove" followed by "add" at the same location.
func applyReplace(doc interface{}, op Operation, opts ApplyOptions) (interface{}, error) {
	path := op.parsedPath
	if !op.parsed {
		var err error
		path, err = ParsePointer(op.Path)
		if err != nil {
			return nil, err
		}
	}

	// Verify the target exists
	if _, err := path.Evaluate(doc); err != nil {
		return nil, fmt.Errorf("target location does not exist: %w", err)
	}

	value, err := op.GetValue()
	if err != nil {
		return nil, err
	}

	// For replace on an object member, we set directly (replaces existing).
	// For replace on an array element, we need to replace in-place (not insert).
	if path.IsRoot() {
		return value, nil
	}

	parent, err := path.Parent().Evaluate(doc)
	if err != nil {
		return nil, err
	}

	key := path.Last()

	switch node := parent.(type) {
	case map[string]interface{}:
		node[key] = value
		return doc, nil
	case []interface{}:
		idx, err := resolveArrayIndex(key, len(node))
		if err != nil {
			return nil, err
		}
		node[idx] = value
		return doc, nil
	default:
		return nil, fmt.Errorf("cannot replace value in %T", parent)
	}
}

// applyMove implements the "move" operation (Section 4.4).
// Functionally identical to "remove" from the source, then "add" at the target.
func applyMove(doc interface{}, op Operation, opts ApplyOptions) (interface{}, error) {
	fromPtr := op.parsedFrom
	pathPtr := op.parsedPath
	if !op.parsed {
		var err error
		fromPtr, err = ParsePointer(op.From)
		if err != nil {
			return nil, err
		}
		pathPtr, err = ParsePointer(op.Path)
		if err != nil {
			return nil, err
		}
	}

	// The "from" location MUST NOT be a proper prefix of the "path" location
	if fromPtr.IsPrefixOf(pathPtr) {
		return nil, fmt.Errorf("\"from\" location %q must not be a proper prefix of \"path\" location %q",
			op.From, op.Path)
	}

	// Get the value at the "from" location
	value, err := fromPtr.Evaluate(doc)
	if err != nil {
		return nil, fmt.Errorf("\"from\" location does not exist: %w", err)
	}

	// No deep copy needed: Remove either calls delete(node, key) for maps
	// (which doesn't invalidate the value reference) or constructs a new
	// backing slice for arrays — in both cases the original reference is valid.

	// Remove from the source
	doc, err = fromPtr.Remove(doc)
	if err != nil {
		return nil, err
	}

	// Add to the target
	return pathPtr.Set(doc, value)
}

// applyCopy implements the "copy" operation (Section 4.5).
// Functionally identical to an "add" operation using the value from "from".
func applyCopy(doc interface{}, op Operation, opts ApplyOptions) (interface{}, error) {
	fromPtr := op.parsedFrom
	pathPtr := op.parsedPath
	if !op.parsed {
		var err error
		fromPtr, err = ParsePointer(op.From)
		if err != nil {
			return nil, err
		}
		pathPtr, err = ParsePointer(op.Path)
		if err != nil {
			return nil, err
		}
	}

	// Get the value at the "from" location
	value, err := fromPtr.Evaluate(doc)
	if err != nil {
		return nil, fmt.Errorf("\"from\" location does not exist: %w", err)
	}

	// Deep copy the value — copy shares a value between two locations, so
	// mutation through one path could affect the other.
	value = deepCopy(value)

	// Add at the target location
	return pathPtr.Set(doc, value)
}

// applyTest implements the "test" operation (Section 4.6).
func applyTest(doc interface{}, op Operation, opts ApplyOptions) (interface{}, error) {
	path := op.parsedPath
	if !op.parsed {
		var err error
		path, err = ParsePointer(op.Path)
		if err != nil {
			return nil, err
		}
	}

	// Get the value at the target location
	actual, err := path.Evaluate(doc)
	if err != nil {
		return nil, fmt.Errorf("target location does not exist: %w", err)
	}

	// Get the expected value
	expected, err := op.GetValue()
	if err != nil {
		return nil, err
	}

	// Compare values using deep equality
	if !jsonEqual(actual, expected) {
		return nil, &TestFailedError{
			Path:     op.Path,
			Expected: expected,
			Actual:   actual,
		}
	}

	return doc, nil
}

// jsonEqual compares two JSON-compatible values for equality per RFC 6902 Section 4.6.
// All callers are expected to pass values already produced by encoding/json
// (i.e., numbers are float64, maps are map[string]interface{}, etc.).
// Uses a recursive type-switch to avoid reflection overhead.
func jsonEqual(a, b interface{}) bool {
	switch av := a.(type) {
	case nil:
		return b == nil
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, va := range av {
			vb, exists := bv[k]
			if !exists || !jsonEqual(va, vb) {
				return false
			}
		}
		return true
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for i, va := range av {
			if !jsonEqual(va, bv[i]) {
				return false
			}
		}
		return true
	default:
		// Fallback for unexpected types — should not occur with JSON-normalized data.
		return a == b
	}
}

// normalizeJSON normalizes a value by round-tripping through JSON serialization.
// This ensures consistent types (e.g., all numbers become float64).
// Used by CreatePatchFromValues to normalize caller-supplied values.
func normalizeJSON(v interface{}) interface{} {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}

// ensurePathExists creates intermediate objects along the pointer's parent
// path so that a subsequent Set will not fail due to a missing parent.
// Only object (map) intermediates are created; array intermediates are not.
func ensurePathExists(doc interface{}, ptr Pointer) interface{} {
	if ptr.IsRoot() || len(ptr.tokens) <= 1 {
		return doc
	}
	if doc == nil {
		doc = make(map[string]interface{})
	}
	current := doc
	// Walk all tokens except the last (which is the key being added).
	for _, token := range ptr.tokens[:len(ptr.tokens)-1] {
		switch node := current.(type) {
		case map[string]interface{}:
			next, ok := node[token]
			if !ok {
				child := make(map[string]interface{})
				node[token] = child
				current = child
			} else {
				current = next
			}
		default:
			// Cannot create intermediates inside arrays or scalars.
			return doc
		}
	}
	return doc
}
