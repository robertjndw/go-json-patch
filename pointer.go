package jsonpatch

import (
	"fmt"
	"strconv"
	"strings"
)

// Pointer represents a JSON Pointer as defined in RFC 6901.
// It references a specific value within a JSON document.
type Pointer struct {
	tokens []string
}

// ParsePointer parses a JSON Pointer string (RFC 6901) into a Pointer.
// The pointer must be empty or start with "/".
func ParsePointer(s string) (Pointer, error) {
	if s == "" {
		return Pointer{}, nil
	}
	if s[0] != '/' {
		return Pointer{}, fmt.Errorf("json pointer must start with '/': %q", s)
	}

	parts := strings.Split(s[1:], "/")
	tokens := make([]string, len(parts))
	for i, part := range parts {
		if err := validatePointerToken(part); err != nil {
			return Pointer{}, fmt.Errorf("invalid JSON Pointer %q: %w", s, err)
		}
		tokens[i] = unescapePointerToken(part)
	}
	return Pointer{tokens: tokens}, nil
}

// String returns the JSON Pointer as a string.
func (p Pointer) String() string {
	if len(p.tokens) == 0 {
		return ""
	}
	// Pre-estimate capacity to avoid reallocations.
	size := 0
	for _, token := range p.tokens {
		size += 1 + len(token) // "/" + token (conservative; escaping may add chars)
	}
	var sb strings.Builder
	sb.Grow(size)
	for _, token := range p.tokens {
		sb.WriteByte('/')
		sb.WriteString(escapePointerToken(token))
	}
	return sb.String()
}

// IsRoot returns true if the pointer references the root of the document.
func (p Pointer) IsRoot() bool {
	return len(p.tokens) == 0
}

// Parent returns the pointer to the parent of the current target.
func (p Pointer) Parent() Pointer {
	if len(p.tokens) == 0 {
		return p
	}
	return Pointer{tokens: p.tokens[:len(p.tokens)-1]}
}

// Last returns the last token of the pointer (the key or index of the target).
func (p Pointer) Last() string {
	if len(p.tokens) == 0 {
		return ""
	}
	return p.tokens[len(p.tokens)-1]
}

// Append returns a new pointer with the given token appended.
func (p Pointer) Append(token string) Pointer {
	newTokens := make([]string, len(p.tokens)+1)
	copy(newTokens, p.tokens)
	newTokens[len(p.tokens)] = token
	return Pointer{tokens: newTokens}
}

// IsPrefixOf returns true if p is a proper prefix of other.
func (p Pointer) IsPrefixOf(other Pointer) bool {
	if len(p.tokens) >= len(other.tokens) {
		return false
	}
	for i, token := range p.tokens {
		if token != other.tokens[i] {
			return false
		}
	}
	return true
}

// Evaluate resolves the pointer against a JSON document and returns the value.
func (p Pointer) Evaluate(doc interface{}) (interface{}, error) {
	current := doc
	for _, token := range p.tokens {
		switch node := current.(type) {
		case map[string]interface{}:
			val, ok := node[token]
			if !ok {
				return nil, &PathNotFoundError{Path: p.String()}
			}
			current = val
		case []interface{}:
			idx, err := resolveArrayIndex(token, len(node))
			if err != nil {
				return nil, err
			}
			current = node[idx]
		default:
			return nil, fmt.Errorf("cannot index into %T with token %q", current, token)
		}
	}
	return current, nil
}

// Set sets the value at the location referenced by the pointer in the document.
// It returns the modified document.
func (p Pointer) Set(doc interface{}, value interface{}) (interface{}, error) {
	if p.IsRoot() {
		return value, nil
	}

	parentPtr := p.Parent()
	parent, err := parentPtr.Evaluate(doc)
	if err != nil {
		return nil, fmt.Errorf("parent path %q does not exist: %w", parentPtr.String(), err)
	}

	key := p.Last()

	switch node := parent.(type) {
	case map[string]interface{}:
		node[key] = value
		return doc, nil
	case []interface{}:
		if key == "-" {
			// Append to the end of the array
			newArr := append(node, value)
			return parentPtr.replaceValue(doc, newArr)
		}
		idx, err := resolveArrayIndex(key, len(node)+1) // +1 because we can insert at the end
		if err != nil {
			return nil, err
		}
		if idx > len(node) {
			return nil, fmt.Errorf("index %d out of bounds for array of length %d", idx, len(node))
		}
		// Insert at index
		newArr := make([]interface{}, len(node)+1)
		copy(newArr[:idx], node[:idx])
		newArr[idx] = value
		copy(newArr[idx+1:], node[idx:])
		return parentPtr.replaceValue(doc, newArr)
	default:
		return nil, fmt.Errorf("cannot set value in %T", parent)
	}
}

// Remove removes the value at the location referenced by the pointer.
// It returns the modified document.
func (p Pointer) Remove(doc interface{}) (interface{}, error) {
	if p.IsRoot() {
		return nil, fmt.Errorf("cannot remove root document")
	}

	parentPtr := p.Parent()
	parent, err := parentPtr.Evaluate(doc)
	if err != nil {
		return nil, fmt.Errorf("parent path %q does not exist: %w", parentPtr.String(), err)
	}

	key := p.Last()

	switch node := parent.(type) {
	case map[string]interface{}:
		if _, ok := node[key]; !ok {
			return nil, &PathNotFoundError{Path: p.String()}
		}
		delete(node, key)
		return doc, nil
	case []interface{}:
		idx, err := resolveArrayIndex(key, len(node))
		if err != nil {
			return nil, err
		}
		if idx >= len(node) {
			return nil, fmt.Errorf("index %d out of bounds for array of length %d", idx, len(node))
		}
		newArr := make([]interface{}, len(node)-1)
		copy(newArr, node[:idx])
		copy(newArr[idx:], node[idx+1:])
		return parentPtr.replaceValue(doc, newArr)
	default:
		return nil, fmt.Errorf("cannot remove value from %T", parent)
	}
}

// replaceValue replaces the value at this pointer's location within the document.
func (p Pointer) replaceValue(doc interface{}, newValue interface{}) (interface{}, error) {
	if p.IsRoot() {
		return newValue, nil
	}

	parent, err := p.Parent().Evaluate(doc)
	if err != nil {
		return nil, err
	}

	key := p.Last()

	switch node := parent.(type) {
	case map[string]interface{}:
		node[key] = newValue
		return doc, nil
	case []interface{}:
		idx, err := resolveArrayIndex(key, len(node))
		if err != nil {
			return nil, err
		}
		node[idx] = newValue
		return doc, nil
	default:
		return nil, fmt.Errorf("cannot replace value in %T", parent)
	}
}

// resolveArrayIndex converts a JSON Pointer token to an array index.
// The "-" token is NOT handled here; callers that need to support it
// (e.g., add/Set) must handle it before calling this function.
func resolveArrayIndex(token string, arrayLen int) (int, error) {
	if token == "-" {
		return 0, fmt.Errorf("the \"-\" token is not valid for this operation (only valid for add target)")
	}
	// Leading zeros are not allowed per RFC 6901
	if len(token) > 1 && token[0] == '0' {
		return 0, fmt.Errorf("array index must not have leading zeros: %q", token)
	}
	idx, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("invalid array index %q: %w", token, err)
	}
	if idx < 0 {
		return 0, fmt.Errorf("array index must not be negative: %d", idx)
	}
	if idx >= arrayLen {
		return 0, &IndexOutOfBoundsError{Index: idx, Length: arrayLen}
	}
	return idx, nil
}

// validatePointerToken checks that a raw (still-escaped) token has valid escape
// sequences per RFC 6901: ~ MUST be followed by '0' or '1'.
func validatePointerToken(raw string) error {
	for i := 0; i < len(raw); i++ {
		if raw[i] == '~' {
			if i+1 >= len(raw) {
				return fmt.Errorf("invalid escape: '~' at end of token %q", raw)
			}
			next := raw[i+1]
			if next != '0' && next != '1' {
				return fmt.Errorf("invalid escape sequence '~%c' in token %q", next, raw)
			}
			i++ // skip the next character, already validated
		}
	}
	return nil
}

// escapePointerToken encodes a token for use in a JSON Pointer string.
// Per RFC 6901: ~ is escaped as ~0, / is escaped as ~1.
func escapePointerToken(token string) string {
	if !strings.ContainsAny(token, "~/") {
		return token
	}
	token = strings.ReplaceAll(token, "~", "~0")
	token = strings.ReplaceAll(token, "/", "~1")
	return token
}

// unescapePointerToken decodes a token from a JSON Pointer string.
// Per RFC 6901: ~1 is unescaped to /, ~0 is unescaped to ~.
// Order matters: ~1 must be processed before ~0.
func unescapePointerToken(token string) string {
	if !strings.Contains(token, "~") {
		return token
	}
	token = strings.ReplaceAll(token, "~1", "/")
	token = strings.ReplaceAll(token, "~0", "~")
	return token
}

// deepCopy creates a deep copy of a JSON-compatible value.
// It recursively copies maps and slices; primitives (string, float64, bool, nil)
// are immutable and returned as-is.
func deepCopy(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, v := range val {
			m[k] = deepCopy(v)
		}
		return m
	case []interface{}:
		a := make([]interface{}, len(val))
		for i, v := range val {
			a[i] = deepCopy(v)
		}
		return a
	default:
		return v
	}
}
