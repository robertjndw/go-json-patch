package jsonpatch

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// RFC 6902 Appendix A examples

func TestApply_A1_AddingObjectMember(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "add", "path": "/baz", "value": "qux"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"baz": "qux", "foo": "bar"}`, string(result))
}

func TestApply_A2_AddingArrayElement(t *testing.T) {
	doc := []byte(`{"foo": ["bar", "baz"]}`)
	patch := []byte(`[{"op": "add", "path": "/foo/1", "value": "qux"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": ["bar", "qux", "baz"]}`, string(result))
}

func TestApply_A3_RemovingObjectMember(t *testing.T) {
	doc := []byte(`{"baz": "qux", "foo": "bar"}`)
	patch := []byte(`[{"op": "remove", "path": "/baz"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": "bar"}`, string(result))
}

func TestApply_A4_RemovingArrayElement(t *testing.T) {
	doc := []byte(`{"foo": ["bar", "qux", "baz"]}`)
	patch := []byte(`[{"op": "remove", "path": "/foo/1"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": ["bar", "baz"]}`, string(result))
}

func TestApply_A5_ReplacingValue(t *testing.T) {
	doc := []byte(`{"baz": "qux", "foo": "bar"}`)
	patch := []byte(`[{"op": "replace", "path": "/baz", "value": "boo"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"baz": "boo", "foo": "bar"}`, string(result))
}

func TestApply_A6_MovingValue(t *testing.T) {
	doc := []byte(`{
		"foo": {"bar": "baz", "waldo": "fred"},
		"qux": {"corge": "grault"}
	}`)
	patch := []byte(`[{"op": "move", "from": "/foo/waldo", "path": "/qux/thud"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{
		"foo": {"bar": "baz"},
		"qux": {"corge": "grault", "thud": "fred"}
	}`, string(result))
}

func TestApply_A7_MovingArrayElement(t *testing.T) {
	doc := []byte(`{"foo": ["all", "grass", "cows", "eat"]}`)
	patch := []byte(`[{"op": "move", "from": "/foo/1", "path": "/foo/3"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": ["all", "cows", "eat", "grass"]}`, string(result))
}

func TestApply_A8_TestingValueSuccess(t *testing.T) {
	doc := []byte(`{"baz": "qux", "foo": ["a", 2, "c"]}`)
	patch := []byte(`[
		{"op": "test", "path": "/baz", "value": "qux"},
		{"op": "test", "path": "/foo/1", "value": 2}
	]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"baz": "qux", "foo": ["a", 2, "c"]}`, string(result))
}

func TestApply_A9_TestingValueError(t *testing.T) {
	doc := []byte(`{"baz": "qux"}`)
	patch := []byte(`[{"op": "test", "path": "/baz", "value": "bar"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected test operation to fail")
	}
}

func TestApply_A10_AddingNestedMemberObject(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "add", "path": "/child", "value": {"grandchild": {}}}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": "bar", "child": {"grandchild": {}}}`, string(result))
}

func TestApply_A11_IgnoringUnrecognizedElements(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "add", "path": "/baz", "value": "qux", "xyz": 123}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": "bar", "baz": "qux"}`, string(result))
}

func TestApply_A12_AddingToNonexistentTarget(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "add", "path": "/baz/bat", "value": "qux"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error when adding to nonexistent target")
	}
}

func TestApply_A14_TildeEscapeOrdering(t *testing.T) {
	doc := []byte(`{"/": 9, "~1": 10}`)
	patch := []byte(`[{"op": "test", "path": "/~01", "value": 10}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"/": 9, "~1": 10}`, string(result))
}

func TestApply_A15_ComparingStringsAndNumbers(t *testing.T) {
	doc := []byte(`{"/": 9, "~1": 10}`)
	patch := []byte(`[{"op": "test", "path": "/~01", "value": "10"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error: string '10' should not equal number 10")
	}
}

func TestApply_A16_AddingArrayValue(t *testing.T) {
	doc := []byte(`{"foo": ["bar"]}`)
	patch := []byte(`[{"op": "add", "path": "/foo/-", "value": ["abc", "def"]}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": ["bar", ["abc", "def"]]}`, string(result))
}

// Additional edge case tests

func TestApply_CopyOperation(t *testing.T) {
	doc := []byte(`{"foo": {"bar": "baz"}, "qux": {}}`)
	patch := []byte(`[{"op": "copy", "from": "/foo/bar", "path": "/qux/bar"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": {"bar": "baz"}, "qux": {"bar": "baz"}}`, string(result))
}

func TestApply_ReplaceRoot(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "replace", "path": "", "value": {"baz": "qux"}}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"baz": "qux"}`, string(result))
}

func TestApply_AddRoot(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "add", "path": "", "value": {"baz": "qux"}}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"baz": "qux"}`, string(result))
}

func TestApply_ReplaceNonexistent(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "replace", "path": "/baz", "value": "qux"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error when replacing nonexistent path")
	}
}

func TestApply_RemoveNonexistent(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "remove", "path": "/baz"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error when removing nonexistent path")
	}
}

func TestApply_MoveCannotMoveIntoChild(t *testing.T) {
	doc := []byte(`{"foo": {"bar": "baz"}}`)
	patch := []byte(`[{"op": "move", "from": "/foo", "path": "/foo/bar/baz"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error: cannot move a value into one of its children")
	}
}

func TestApply_MultipleOperationsSequential(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[
		{"op": "add", "path": "/baz", "value": "qux"},
		{"op": "replace", "path": "/foo", "value": "updated"},
		{"op": "add", "path": "/arr", "value": [1, 2, 3]},
		{"op": "remove", "path": "/arr/1"},
		{"op": "test", "path": "/arr", "value": [1, 3]}
	]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": "updated", "baz": "qux", "arr": [1, 3]}`, string(result))
}

func TestApply_AtomicFailure(t *testing.T) {
	// Per RFC 6902 Section 5: if any operation fails, the entire patch fails
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[
		{"op": "add", "path": "/baz", "value": "qux"},
		{"op": "test", "path": "/baz", "value": "wrong"}
	]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected test failure to cause entire patch to fail")
	}
}

func TestApply_AddReplaceExistingMember(t *testing.T) {
	// Per RFC 6902 Section 4.1: if the target location specifies an object
	// member that does exist, that member's value is replaced.
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "add", "path": "/foo", "value": "baz"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": "baz"}`, string(result))
}

func TestApply_InvalidOperation(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "invalid", "path": "/foo"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error for invalid operation")
	}
}

func TestApply_InvalidPatchJSON(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`not json`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error for invalid patch JSON")
	}
}

func TestApply_InvalidDocJSON(t *testing.T) {
	doc := []byte(`not json`)
	patch := []byte(`[{"op": "add", "path": "/foo", "value": "bar"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error for invalid document JSON")
	}
}

func TestApply_AddMissingValue(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "add", "path": "/baz"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error: add operation requires value")
	}
}

func TestApply_MoveMissingFrom(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "move", "path": "/baz"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error: move operation requires from")
	}
}

func TestApply_ReplaceArrayElement(t *testing.T) {
	doc := []byte(`{"foo": [1, 2, 3]}`)
	patch := []byte(`[{"op": "replace", "path": "/foo/1", "value": 99}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": [1, 99, 3]}`, string(result))
}

func TestApply_AddToEndOfArray(t *testing.T) {
	doc := []byte(`{"foo": [1, 2]}`)
	patch := []byte(`[{"op": "add", "path": "/foo/-", "value": 3}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": [1, 2, 3]}`, string(result))
}

func TestApply_CopyArray(t *testing.T) {
	doc := []byte(`{"foo": [1, 2, 3]}`)
	patch := []byte(`[{"op": "copy", "from": "/foo", "path": "/bar"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": [1, 2, 3], "bar": [1, 2, 3]}`, string(result))
}

func TestApply_TestWithNull(t *testing.T) {
	doc := []byte(`{"foo": null}`)
	patch := []byte(`[{"op": "test", "path": "/foo", "value": null}]`)

	_, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
}

func TestApply_TestWithBoolean(t *testing.T) {
	doc := []byte(`{"foo": true, "bar": false}`)
	patch := []byte(`[
		{"op": "test", "path": "/foo", "value": true},
		{"op": "test", "path": "/bar", "value": false}
	]`)

	_, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
}

func TestApply_TestWithNestedObject(t *testing.T) {
	doc := []byte(`{"foo": {"bar": [1, 2, 3]}}`)
	patch := []byte(`[{"op": "test", "path": "/foo", "value": {"bar": [1, 2, 3]}}]`)

	_, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
}

// --- Targeted compliance tests ---

func TestApply_MissingPathIsRejected(t *testing.T) {
	// RFC 6902: every operation object MUST have exactly one "path" member.
	tests := []struct {
		name  string
		patch string
	}{
		{"add without path", `[{"op": "add", "value": "x"}]`},
		{"remove without path", `[{"op": "remove"}]`},
		{"replace without path", `[{"op": "replace", "value": "x"}]`},
		{"move without path", `[{"op": "move", "from": "/foo"}]`},
		{"copy without path", `[{"op": "copy", "from": "/foo"}]`},
		{"test without path", `[{"op": "test", "value": "bar"}]`},
	}
	doc := []byte(`{"foo": "bar"}`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Apply(doc, []byte(tt.patch))
			if err == nil {
				t.Fatalf("expected error for patch without path: %s", tt.patch)
			}
		})
	}
}

func TestApply_MoveFromRootPointer(t *testing.T) {
	// from:"" is the root pointer and must be accepted by validation.
	// Moving root to a child is blocked by the prefix check (correctly),
	// but the operation must not be rejected for a missing "from".
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "move", "from": "", "path": "/new"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error: root is a prefix of /new")
	}
	// The error must be about prefix, NOT about missing "from" member.
	if got := err.Error(); !strings.Contains(got, "prefix") {
		t.Fatalf("expected prefix error, got: %v", err)
	}
}

func TestApply_CopyFromRootPointer(t *testing.T) {
	// from:"" is the root pointer and must be accepted for copy.
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "copy", "from": "", "path": "/dup"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatalf("copy with from='' (root) should succeed: %v", err)
	}
	assertJSONEqual(t, `{"foo": "bar", "dup": {"foo": "bar"}}`, string(result))
}

func TestApply_DashTokenOnlyValidForAdd(t *testing.T) {
	// The "-" token references the nonexistent element after the last array element.
	// It is only valid as the final token for an "add" target path.
	doc := []byte(`{"foo": [1, 2, 3]}`)

	tests := []struct {
		name  string
		patch string
	}{
		{"test on /-", `[{"op": "test", "path": "/foo/-", "value": 3}]`},
		{"replace on /-", `[{"op": "replace", "path": "/foo/-", "value": 99}]`},
		{"remove on /-", `[{"op": "remove", "path": "/foo/-"}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Apply(doc, []byte(tt.patch))
			if err == nil {
				t.Fatalf("expected error for %s: %s", tt.name, tt.patch)
			}
		})
	}
}

func TestApply_AddWithDashStillWorks(t *testing.T) {
	doc := []byte(`{"foo": [1, 2]}`)
	patch := []byte(`[{"op": "add", "path": "/foo/-", "value": 3}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": [1, 2, 3]}`, string(result))
}

func TestApply_MissingOpIsRejected(t *testing.T) {
	// RFC 6902 Section 4: operation objects MUST have exactly one "op" member.
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"path": "/foo", "value": "baz"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error for operation without op member")
	}
}

func TestApply_NonStringOpRejected(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	tests := []struct {
		name  string
		patch string
	}{
		{"op as number", `[{"op": 123, "path": "/foo", "value": "x"}]`},
		{"op as boolean", `[{"op": true, "path": "/foo", "value": "x"}]`},
		{"op as null", `[{"op": null, "path": "/foo", "value": "x"}]`},
		{"op as array", `[{"op": [], "path": "/foo", "value": "x"}]`},
		{"op as object", `[{"op": {}, "path": "/foo", "value": "x"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Apply(doc, []byte(tt.patch))
			if err == nil {
				t.Fatalf("expected error for non-string op: %s", tt.patch)
			}
		})
	}
}

func TestApply_A13_InvalidPatchDocument(t *testing.T) {
	// RFC 6902 Appendix A.13: A JSON Patch document that is not an array
	// is an invalid patch document.
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`{"op": "add", "path": "/baz", "value": "qux"}`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error for patch document that is not an array")
	}
}

// assertJSONEqual compares two JSON strings for semantic equality.
func assertJSONEqual(t *testing.T, expected, actual string) {
	t.Helper()
	var e, a interface{}
	if err := json.Unmarshal([]byte(expected), &e); err != nil {
		t.Fatalf("invalid expected JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(actual), &a); err != nil {
		t.Fatalf("invalid actual JSON: %v\nraw: %s", err, actual)
	}
	if !jsonEqual(e, a) {
		t.Errorf("JSON not equal:\n  expected: %s\n  actual:   %s", expected, actual)
	}
}

// ---------------------------------------------------------------------------
// Structured error types
// ---------------------------------------------------------------------------

func TestStructuredError_TestFailedError(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "test", "path": "/foo", "value": "wrong"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error")
	}

	var opErr *InvalidOperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected InvalidOperationError, got %T: %v", err, err)
	}
	if opErr.Index != 0 || opErr.Op != OpTest {
		t.Errorf("unexpected operation error: index=%d, op=%s", opErr.Index, opErr.Op)
	}

	var testErr *TestFailedError
	if !errors.As(err, &testErr) {
		t.Fatalf("expected TestFailedError, got %T: %v", err, err)
	}
	if testErr.Path != "/foo" {
		t.Errorf("expected path /foo, got %s", testErr.Path)
	}
	if testErr.Expected != "wrong" {
		t.Errorf("expected expected=wrong, got %v", testErr.Expected)
	}
	if testErr.Actual != "bar" {
		t.Errorf("expected actual=bar, got %v", testErr.Actual)
	}
}

func TestStructuredError_PathNotFoundError(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "remove", "path": "/missing"}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error")
	}

	var pnf *PathNotFoundError
	if !errors.As(err, &pnf) {
		t.Fatalf("expected PathNotFoundError, got %T: %v", err, err)
	}
}

func TestStructuredError_IndexOutOfBoundsError(t *testing.T) {
	doc := []byte(`{"arr": [1, 2]}`)
	patch := []byte(`[{"op": "replace", "path": "/arr/5", "value": 99}]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error")
	}

	var idx *IndexOutOfBoundsError
	if !errors.As(err, &idx) {
		t.Fatalf("expected IndexOutOfBoundsError, got %T: %v", err, err)
	}
	if idx.Index != 5 || idx.Length != 2 {
		t.Errorf("expected index=5 length=2, got index=%d length=%d", idx.Index, idx.Length)
	}
}

func TestStructuredError_InvalidOperationError(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[
		{"op": "add", "path": "/x", "value": 1},
		{"op": "remove", "path": "/nonexistent"}
	]`)

	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error")
	}

	var opErr *InvalidOperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected InvalidOperationError, got %T: %v", err, err)
	}
	if opErr.Index != 1 {
		t.Errorf("expected index 1, got %d", opErr.Index)
	}
	if opErr.Op != OpRemove {
		t.Errorf("expected op remove, got %s", opErr.Op)
	}
}

// ---------------------------------------------------------------------------
// ApplyWithOptions
// ---------------------------------------------------------------------------

func TestApplyWithOptions_AllowMissingPathOnRemove(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "remove", "path": "/nonexistent"}]`)

	// Without option — should fail.
	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error without option")
	}

	// With option — should succeed and return document unchanged.
	result, err := ApplyWithOptions(doc, patch, WithAllowMissingPathOnRemove())
	if err != nil {
		t.Fatalf("expected no error with AllowMissingPathOnRemove, got: %v", err)
	}
	assertJSONEqual(t, `{"foo": "bar"}`, string(result))
}

func TestApplyWithOptions_AllowMissingPathOnRemove_ExistingPath(t *testing.T) {
	doc := []byte(`{"foo": "bar", "baz": "qux"}`)
	patch := []byte(`[{"op": "remove", "path": "/baz"}]`)

	result, err := ApplyWithOptions(doc, patch, WithAllowMissingPathOnRemove())
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": "bar"}`, string(result))
}

func TestApplyWithOptions_EnsurePathExistsOnAdd(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patch := []byte(`[{"op": "add", "path": "/a/b/c", "value": "deep"}]`)

	// Without option — should fail.
	_, err := Apply(doc, patch)
	if err == nil {
		t.Fatal("expected error without option")
	}

	// With option — should auto-create intermediates.
	result, err := ApplyWithOptions(doc, patch, WithEnsurePathExistsOnAdd())
	if err != nil {
		t.Fatalf("expected no error with EnsurePathExistsOnAdd, got: %v", err)
	}
	assertJSONEqual(t, `{"foo": "bar", "a": {"b": {"c": "deep"}}}`, string(result))
}

func TestApplyPatchWithOptions(t *testing.T) {
	doc := []byte(`{"foo": "bar"}`)
	patchJSON := []byte(`[
		{"op": "remove", "path": "/missing"},
		{"op": "add", "path": "/baz", "value": "qux"}
	]`)
	patch, err := DecodePatch(patchJSON)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatchWithOptions(doc, patch, WithAllowMissingPathOnRemove())
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{"foo": "bar", "baz": "qux"}`, string(result))
}

// ---------------------------------------------------------------------------
// jsonEqual
// ---------------------------------------------------------------------------

func TestJsonEqual_Nil(t *testing.T) {
	if !jsonEqual(nil, nil) {
		t.Error("nil should equal nil")
	}
	if jsonEqual(nil, "x") {
		t.Error("nil should not equal string")
	}
}

func TestJsonEqual_Bool(t *testing.T) {
	if !jsonEqual(true, true) {
		t.Error("true should equal true")
	}
	if jsonEqual(true, false) {
		t.Error("true should not equal false")
	}
	if jsonEqual(true, "true") {
		t.Error("bool should not equal string")
	}
}

func TestJsonEqual_Float64(t *testing.T) {
	if !jsonEqual(1.5, 1.5) {
		t.Error("1.5 should equal 1.5")
	}
	if jsonEqual(1.5, 2.5) {
		t.Error("1.5 should not equal 2.5")
	}
}

func TestJsonEqual_String(t *testing.T) {
	if !jsonEqual("abc", "abc") {
		t.Error("abc should equal abc")
	}
	if jsonEqual("abc", "def") {
		t.Error("abc should not equal def")
	}
}

func TestJsonEqual_Map(t *testing.T) {
	a := map[string]interface{}{"x": float64(1), "y": "two"}
	b := map[string]interface{}{"x": float64(1), "y": "two"}
	c := map[string]interface{}{"x": float64(1)}

	if !jsonEqual(a, b) {
		t.Error("identical maps should be equal")
	}
	if jsonEqual(a, c) {
		t.Error("maps with different keys should not be equal")
	}
}

func TestJsonEqual_Slice(t *testing.T) {
	a := []interface{}{float64(1), "two", true}
	b := []interface{}{float64(1), "two", true}
	c := []interface{}{float64(1), "two", false}

	if !jsonEqual(a, b) {
		t.Error("identical slices should be equal")
	}
	if jsonEqual(a, c) {
		t.Error("slices with different elements should not be equal")
	}
}

func TestJsonEqual_NestedStructures(t *testing.T) {
	a := map[string]interface{}{
		"arr": []interface{}{float64(1), map[string]interface{}{"k": "v"}},
	}
	b := map[string]interface{}{
		"arr": []interface{}{float64(1), map[string]interface{}{"k": "v"}},
	}
	if !jsonEqual(a, b) {
		t.Error("deeply nested equal structures should match")
	}
}
