package jsonpatch

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// CreatePatch generates a JSON Patch document (RFC 6902) that transforms
// the original JSON document into the modified JSON document.
// Both arguments must be valid JSON bytes.
func CreatePatch(original, modified []byte) (Patch, error) {
	var origDoc, modDoc interface{}

	if err := json.Unmarshal(original, &origDoc); err != nil {
		return nil, fmt.Errorf("failed to decode original document: %w", err)
	}
	if err := json.Unmarshal(modified, &modDoc); err != nil {
		return nil, fmt.Errorf("failed to decode modified document: %w", err)
	}

	patch := Patch{}
	diff(&patch, "", origDoc, modDoc)
	return patch, nil
}

// CreatePatchFromValues generates a JSON Patch from two already-parsed JSON values.
func CreatePatchFromValues(original, modified interface{}) Patch {
	patch := Patch{}
	diff(&patch, "", normalizeJSON(original), normalizeJSON(modified))
	return patch
}

// diff recursively computes the differences between two JSON values
// and appends the corresponding operations to the patch.
func diff(patch *Patch, path string, original, modified interface{}) {
	// Fast path for primitives — avoids reflect.DeepEqual overhead.
	switch o := original.(type) {
	case nil:
		if modified == nil {
			return
		}
	case bool:
		if m, ok := modified.(bool); ok && o == m {
			return
		}
	case float64:
		if m, ok := modified.(float64); ok && o == m {
			return
		}
	case string:
		if m, ok := modified.(string); ok && o == m {
			return
		}
	default:
		// Composite types — fall through to structural comparison
		if jsonEqual(original, modified) {
			return
		}
	}

	origObj, origIsObj := original.(map[string]interface{})
	modObj, modIsObj := modified.(map[string]interface{})

	origArr, origIsArr := original.([]interface{})
	modArr, modIsArr := modified.([]interface{})

	switch {
	case origIsObj && modIsObj:
		diffObjects(patch, path, origObj, modObj)
	case origIsArr && modIsArr:
		diffArrays(patch, path, origArr, modArr)
	default:
		// Types differ or primitive values differ — replace
		*patch = append(*patch, newReplaceOp(path, modified))
	}
}

// diffObjects computes the diff between two JSON objects.
func diffObjects(patch *Patch, path string, original, modified map[string]interface{}) {
	// Collect all keys from both objects for deterministic ordering
	keys := make(map[string]bool)
	for k := range original {
		keys[k] = true
	}
	for k := range modified {
		keys[k] = true
	}

	sortedKeys := make([]string, 0, len(keys))
	for k := range keys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		childPath := path + "/" + escapePointerToken(key)
		origVal, inOrig := original[key]
		modVal, inMod := modified[key]

		switch {
		case inOrig && inMod:
			// Key exists in both — recurse
			diff(patch, childPath, origVal, modVal)
		case inOrig && !inMod:
			// Key removed
			*patch = append(*patch, newRemoveOp(childPath))
		case !inOrig && inMod:
			// Key added
			*patch = append(*patch, newAddOp(childPath, modVal))
		}
	}
}

// diffArrays computes the diff between two JSON arrays.
// Uses a simple approach: compare element by element, then handle length differences.
func diffArrays(patch *Patch, path string, original, modified []interface{}) {
	commonLen := len(original)
	if len(modified) < commonLen {
		commonLen = len(modified)
	}

	// Compare common elements
	for i := 0; i < commonLen; i++ {
		childPath := path + "/" + strconv.Itoa(i)
		diff(patch, childPath, original[i], modified[i])
	}

	// Handle extra elements in modified (additions)
	if len(modified) > len(original) {
		for i := len(original); i < len(modified); i++ {
			childPath := path + "/-"
			*patch = append(*patch, newAddOp(childPath, modified[i]))
		}
	}

	// Handle extra elements in original (removals)
	// Remove from the end to avoid index shifting issues
	if len(original) > len(modified) {
		for i := len(original) - 1; i >= len(modified); i-- {
			childPath := path + "/" + strconv.Itoa(i)
			*patch = append(*patch, newRemoveOp(childPath))
		}
	}
}

// newAddOp creates an "add" operation.
func newAddOp(path string, value interface{}) Operation {
	op, _ := NewOperation(OpAdd, path, value)
	return op
}

// newRemoveOp creates a "remove" operation.
func newRemoveOp(path string) Operation {
	return Operation{
		Op:      OpRemove,
		Path:    path,
		hasPath: true,
	}
}

// newReplaceOp creates a "replace" operation.
func newReplaceOp(path string, value interface{}) Operation {
	op, _ := NewOperation(OpReplace, path, value)
	return op
}
