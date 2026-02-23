package jsonpatch

import (
	"encoding/json"
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
