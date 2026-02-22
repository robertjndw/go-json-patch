package jsonpatch

import (
	"encoding/json"
	"fmt"
	"reflect"
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

// ApplyPatch applies a decoded Patch to a target JSON document (as raw JSON bytes).
// It returns the patched document as raw JSON bytes.
func ApplyPatch(docJSON []byte, patch Patch) ([]byte, error) {
	var doc interface{}
	if err := json.Unmarshal(docJSON, &doc); err != nil {
		return nil, fmt.Errorf("failed to decode target document: %w", err)
	}

	var err error
	for i, op := range patch {
		doc, err = applyOperation(doc, op)
		if err != nil {
			return nil, fmt.Errorf("operation %d (%s %s) failed: %w", i, op.Op, op.Path, err)
		}
	}

	result, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	return result, nil
}

// applyOperation applies a single operation to the document.
func applyOperation(doc interface{}, op Operation) (interface{}, error) {
	switch op.Op {
	case OpAdd:
		return applyAdd(doc, op)
	case OpRemove:
		return applyRemove(doc, op)
	case OpReplace:
		return applyReplace(doc, op)
	case OpMove:
		return applyMove(doc, op)
	case OpCopy:
		return applyCopy(doc, op)
	case OpTest:
		return applyTest(doc, op)
	default:
		return nil, fmt.Errorf("unknown operation %q", op.Op)
	}
}

// applyAdd implements the "add" operation (Section 4.1).
func applyAdd(doc interface{}, op Operation) (interface{}, error) {
	path, err := ParsePointer(op.Path)
	if err != nil {
		return nil, err
	}

	value, err := op.GetValue()
	if err != nil {
		return nil, err
	}

	return path.Set(doc, value)
}

// applyRemove implements the "remove" operation (Section 4.2).
func applyRemove(doc interface{}, op Operation) (interface{}, error) {
	path, err := ParsePointer(op.Path)
	if err != nil {
		return nil, err
	}

	return path.Remove(doc)
}

// applyReplace implements the "replace" operation (Section 4.3).
// Functionally identical to a "remove" followed by "add" at the same location.
func applyReplace(doc interface{}, op Operation) (interface{}, error) {
	path, err := ParsePointer(op.Path)
	if err != nil {
		return nil, err
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
func applyMove(doc interface{}, op Operation) (interface{}, error) {
	fromPtr, err := ParsePointer(op.From)
	if err != nil {
		return nil, err
	}

	pathPtr, err := ParsePointer(op.Path)
	if err != nil {
		return nil, err
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

	// Deep copy the value to avoid mutation issues
	value = deepCopy(value)

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
func applyCopy(doc interface{}, op Operation) (interface{}, error) {
	fromPtr, err := ParsePointer(op.From)
	if err != nil {
		return nil, err
	}

	pathPtr, err := ParsePointer(op.Path)
	if err != nil {
		return nil, err
	}

	// Get the value at the "from" location
	value, err := fromPtr.Evaluate(doc)
	if err != nil {
		return nil, fmt.Errorf("\"from\" location does not exist: %w", err)
	}

	// Deep copy the value
	value = deepCopy(value)

	// Add at the target location
	return pathPtr.Set(doc, value)
}

// applyTest implements the "test" operation (Section 4.6).
func applyTest(doc interface{}, op Operation) (interface{}, error) {
	path, err := ParsePointer(op.Path)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("test failed: value at %q does not match: got %v, expected %v",
			op.Path, actual, expected)
	}

	return doc, nil
}

// jsonEqual compares two JSON-compatible values for equality per RFC 6902 Section 4.6.
func jsonEqual(a, b interface{}) bool {
	// Normalize through JSON round-trip to ensure consistent types
	na := normalizeJSON(a)
	nb := normalizeJSON(b)
	return reflect.DeepEqual(na, nb)
}

// normalizeJSON normalizes a value by round-tripping through JSON serialization.
// This ensures consistent types (e.g., all numbers become float64).
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
