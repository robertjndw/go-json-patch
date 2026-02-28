package jsonpatch

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	stack := make([]string, 0, 16) // token stack, pre-allocated
	diff(&patch, &stack, origDoc, modDoc)
	return patch, nil
}

// CreatePatchFromValues generates a JSON Patch from two already-parsed JSON values.
func CreatePatchFromValues(original, modified interface{}) Patch {
	patch := Patch{}
	stack := make([]string, 0, 16)
	diff(&patch, &stack, normalizeJSON(original), normalizeJSON(modified))
	return patch
}

// stackToPath serializes a token stack to a JSON Pointer string.
func stackToPath(stack *[]string) string {
	s := *stack
	if len(s) == 0 {
		return ""
	}
	// Estimate capacity: "/" per token + average token length.
	size := 0
	for _, t := range s {
		size += 1 + len(t)
	}
	var sb strings.Builder
	sb.Grow(size)
	for _, t := range s {
		sb.WriteByte('/')
		sb.WriteString(escapePointerToken(t))
	}
	return sb.String()
}

// diff recursively computes the differences between two JSON values
// and appends the corresponding operations to the patch.
func diff(patch *Patch, stack *[]string, original, modified interface{}) {
	// Fast path for primitives — avoids type-switch overhead.
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
		diffObjects(patch, stack, origObj, modObj)
	case origIsArr && modIsArr:
		diffArrays(patch, stack, origArr, modArr)
	default:
		// Types differ or primitive values differ — replace
		*patch = append(*patch, newReplaceOp(stackToPath(stack), modified))
	}
}

// diffObjects computes the diff between two JSON objects.
func diffObjects(patch *Patch, stack *[]string, original, modified map[string]interface{}) {
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
		*stack = append(*stack, key)
		origVal, inOrig := original[key]
		modVal, inMod := modified[key]

		switch {
		case inOrig && inMod:
			// Key exists in both — recurse
			diff(patch, stack, origVal, modVal)
		case inOrig && !inMod:
			// Key removed
			*patch = append(*patch, newRemoveOp(stackToPath(stack)))
		case !inOrig && inMod:
			// Key added
			*patch = append(*patch, newAddOp(stackToPath(stack), modVal))
		}
		*stack = (*stack)[:len(*stack)-1] // pop
	}
}

// diffArrays computes the diff between two JSON arrays.
// Uses a simple approach: compare element by element, then handle length differences.
func diffArrays(patch *Patch, stack *[]string, original, modified []interface{}) {
	commonLen := len(original)
	if len(modified) < commonLen {
		commonLen = len(modified)
	}

	// Compare common elements
	for i := 0; i < commonLen; i++ {
		*stack = append(*stack, strconv.Itoa(i))
		diff(patch, stack, original[i], modified[i])
		*stack = (*stack)[:len(*stack)-1] // pop
	}

	// Handle extra elements in modified (additions)
	if len(modified) > len(original) {
		*stack = append(*stack, "-")
		path := stackToPath(stack)
		*stack = (*stack)[:len(*stack)-1] // pop
		for i := len(original); i < len(modified); i++ {
			*patch = append(*patch, newAddOp(path, modified[i]))
		}
	}

	// Handle extra elements in original (removals)
	// Remove from the end to avoid index shifting issues
	if len(original) > len(modified) {
		for i := len(original) - 1; i >= len(modified); i-- {
			*stack = append(*stack, strconv.Itoa(i))
			*patch = append(*patch, newRemoveOp(stackToPath(stack)))
			*stack = (*stack)[:len(*stack)-1] // pop
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
