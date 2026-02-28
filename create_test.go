package jsonpatch

import (
	"encoding/json"
	"testing"
)

func TestCreatePatch_AddObjectMember(t *testing.T) {
	original := []byte(`{"foo": "bar"}`)
	modified := []byte(`{"foo": "bar", "baz": "qux"}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	// Apply the generated patch and verify the result
	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_RemoveObjectMember(t *testing.T) {
	original := []byte(`{"baz": "qux", "foo": "bar"}`)
	modified := []byte(`{"foo": "bar"}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_ReplaceValue(t *testing.T) {
	original := []byte(`{"baz": "qux", "foo": "bar"}`)
	modified := []byte(`{"baz": "boo", "foo": "bar"}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_NestedObjects(t *testing.T) {
	original := []byte(`{
		"foo": {"bar": "baz", "waldo": "fred"},
		"qux": {"corge": "grault"}
	}`)
	modified := []byte(`{
		"foo": {"bar": "baz"},
		"qux": {"corge": "grault", "thud": "fred"}
	}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_ArrayModification(t *testing.T) {
	original := []byte(`{"foo": ["bar", "baz"]}`)
	modified := []byte(`{"foo": ["bar", "qux", "baz"]}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_ArrayRemoval(t *testing.T) {
	original := []byte(`{"foo": ["bar", "qux", "baz"]}`)
	modified := []byte(`{"foo": ["bar", "baz"]}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_TypeChange(t *testing.T) {
	original := []byte(`{"foo": "bar"}`)
	modified := []byte(`{"foo": 42}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_NullValue(t *testing.T) {
	original := []byte(`{"foo": "bar"}`)
	modified := []byte(`{"foo": null}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_ComplexDocument(t *testing.T) {
	original := []byte(`{
		"title": "Hello",
		"author": {"name": "John", "email": "john@example.com"},
		"tags": ["go", "json"],
		"published": false
	}`)
	modified := []byte(`{
		"title": "Hello World",
		"author": {"name": "John", "email": "john@newdomain.com"},
		"tags": ["go", "json", "patch"],
		"published": true,
		"views": 100
	}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_IdenticalDocuments(t *testing.T) {
	original := []byte(`{"foo": "bar", "baz": [1, 2, 3]}`)

	patch, err := CreatePatch(original, original)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 0 {
		t.Errorf("expected empty patch for identical documents, got %d operations", len(patch))
	}
}

func TestCreatePatch_EmptyToNonEmpty(t *testing.T) {
	original := []byte(`{}`)
	modified := []byte(`{"foo": "bar", "baz": 42}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_NonEmptyToEmpty(t *testing.T) {
	original := []byte(`{"foo": "bar", "baz": 42}`)
	modified := []byte(`{}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_SpecialCharactersInKeys(t *testing.T) {
	original := []byte(`{"a/b": 1, "c~d": 2}`)
	modified := []byte(`{"a/b": 10, "c~d": 20}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestCreatePatch_RootArrays(t *testing.T) {
	original := []byte(`[1, 2, 3]`)
	modified := []byte(`[1, 4, 3, 5]`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, string(modified), string(result))
}

func TestMarshalPatch(t *testing.T) {
	patch := Patch{
		{Op: OpAdd, Path: "/foo"},
		{Op: OpRemove, Path: "/bar"},
	}

	data, err := MarshalPatch(patch)
	if err != nil {
		t.Fatal(err)
	}

	// Verify it's valid JSON
	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 operations, got %d", len(result))
	}
}

func TestDecodePatch(t *testing.T) {
	raw := []byte(`[
		{"op": "add", "path": "/foo", "value": "bar"},
		{"op": "remove", "path": "/baz"},
		{"op": "replace", "path": "/qux", "value": 42},
		{"op": "move", "from": "/a", "path": "/b"},
		{"op": "copy", "from": "/c", "path": "/d"},
		{"op": "test", "path": "/e", "value": true}
	]`)

	patch, err := DecodePatch(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 6 {
		t.Fatalf("expected 6 operations, got %d", len(patch))
	}

	expectedOps := []OpType{OpAdd, OpRemove, OpReplace, OpMove, OpCopy, OpTest}
	for i, op := range patch {
		if op.Op != expectedOps[i] {
			t.Errorf("operation %d: expected %q, got %q", i, expectedOps[i], op.Op)
		}
	}
}

func TestDecodePatch_NonStringPathRejected(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"path as number", `[{"op":"remove","path":123}]`},
		{"path as object", `[{"op":"add","path":{},"value":"x"}]`},
		{"path as array", `[{"op":"test","path":[],"value":true}]`},
		{"path as boolean", `[{"op":"replace","path":false,"value":1}]`},
		{"path as null", `[{"op":"remove","path":null}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePatch([]byte(tt.raw))
			if err == nil {
				t.Fatalf("expected DecodePatch to reject non-string path: %s", tt.raw)
			}
		})
	}
}

func TestDecodePatch_NonStringFromRejected(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"from as number", `[{"op":"move","from":123,"path":"/a"}]`},
		{"from as object", `[{"op":"copy","from":{},"path":"/a"}]`},
		{"from as array", `[{"op":"move","from":[],"path":"/a"}]`},
		{"from as boolean", `[{"op":"copy","from":true,"path":"/a"}]`},
		{"from as null", `[{"op":"move","from":null,"path":"/a"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePatch([]byte(tt.raw))
			if err == nil {
				t.Fatalf("expected DecodePatch to reject non-string from: %s", tt.raw)
			}
		})
	}
}

func TestDecodePatch_DuplicateMemberBehavior(t *testing.T) {
	// Duplicate object members have undefined handling in JSON Patch (RFC 6902 Appendix A.13).
	// Go's encoding/json keeps the last occurrence; this test documents and guards that behavior.
	raw := []byte(`[{"op":"remove","path":"/first","path":"/second"}]`)

	patch, err := DecodePatch(raw)
	if err != nil {
		t.Fatalf("expected decode to succeed with duplicate path key under encoding/json semantics: %v", err)
	}

	if len(patch) != 1 {
		t.Fatalf("expected one operation, got %d", len(patch))
	}

	if patch[0].Path != "/second" {
		t.Fatalf("expected last duplicate key to win, got path=%q", patch[0].Path)
	}
}

func TestCreatePatch_InvalidOriginalJSON(t *testing.T) {
	_, err := CreatePatch([]byte(`not json`), []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for invalid original JSON")
	}
}

// ---------------------------------------------------------------------------
// Operation.Validate() and HasFrom()
// ---------------------------------------------------------------------------

func TestValidate_StructLiteral(t *testing.T) {
	// Struct literal without hasPath set — Validate should infer it.
	raw := json.RawMessage(`"hello"`)
	op := Operation{
		Op:    OpAdd,
		Path:  "/foo",
		Value: &raw,
	}
	if err := op.Validate(); err != nil {
		t.Fatalf("Validate() should succeed for well-formed struct literal: %v", err)
	}
	if !op.parsed {
		t.Fatal("expected parsed to be true after Validate()")
	}
}

func TestValidate_MoveStructLiteral(t *testing.T) {
	op := Operation{
		Op:   OpMove,
		From: "/a",
		Path: "/b",
	}
	if err := op.Validate(); err != nil {
		t.Fatalf("Validate() should succeed: %v", err)
	}
}

func TestValidate_CopyStructLiteralWithRootFrom(t *testing.T) {
	// From="" (root) should be accepted for copy.
	op := Operation{
		Op:   OpCopy,
		From: "",
		Path: "/dup",
	}
	if err := op.Validate(); err != nil {
		t.Fatalf("Validate() should succeed for copy with root from: %v", err)
	}
}

func TestValidate_RemoveStructLiteral(t *testing.T) {
	op := Operation{
		Op:   OpRemove,
		Path: "/foo",
	}
	if err := op.Validate(); err != nil {
		t.Fatalf("Validate() should succeed: %v", err)
	}
}

func TestValidate_MissingOp(t *testing.T) {
	op := Operation{Path: "/foo"}
	if err := op.Validate(); err == nil {
		t.Fatal("expected error for missing op")
	}
}

func TestValidate_InvalidPath(t *testing.T) {
	raw := json.RawMessage(`1`)
	op := Operation{
		Op:    OpAdd,
		Path:  "no-slash",
		Value: &raw,
	}
	if err := op.Validate(); err == nil {
		t.Fatal("expected error for invalid path without leading slash")
	}
}

func TestValidate_CachesPointers(t *testing.T) {
	raw := json.RawMessage(`"val"`)
	op := Operation{
		Op:    OpReplace,
		Path:  "/a/b",
		Value: &raw,
	}
	if err := op.Validate(); err != nil {
		t.Fatal(err)
	}
	if op.parsedPath.String() != "/a/b" {
		t.Errorf("expected cached path /a/b, got %s", op.parsedPath.String())
	}
}

func TestHasFrom(t *testing.T) {
	op1 := NewMoveOperation("/a", "/b")
	if !op1.HasFrom() {
		t.Error("expected HasFrom() true for NewMoveOperation")
	}

	op2 := Operation{Op: OpAdd, Path: "/x"}
	if op2.HasFrom() {
		t.Error("expected HasFrom() false for operation without from")
	}
}

// ---------------------------------------------------------------------------
// Cached value behaviour
// ---------------------------------------------------------------------------

func TestCachedValue_DecodedPatchUsesCache(t *testing.T) {
	patchJSON := []byte(`[{"op": "add", "path": "/foo", "value": {"nested": true}}]`)
	patch, err := DecodePatch(patchJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !patch[0].parsed {
		t.Fatal("expected parsed=true after DecodePatch")
	}
	val, err := patch[0].GetValue()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", val)
	}
	if m["nested"] != true {
		t.Errorf("expected nested=true, got %v", m["nested"])
	}
}

func TestCreatePatch_InvalidModifiedJSON(t *testing.T) {
	_, err := CreatePatch([]byte(`{}`), []byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid modified JSON")
	}
}

func TestCreateAndApply_RoundTrip(t *testing.T) {
	// A comprehensive round-trip test: create a patch from two documents,
	// then apply it to the original and verify we get the modified version.
	docs := []struct {
		name     string
		original string
		modified string
	}{
		{
			"simple object",
			`{"a": 1, "b": 2}`,
			`{"a": 1, "b": 3, "c": 4}`,
		},
		{
			"nested object",
			`{"a": {"b": {"c": 1}}}`,
			`{"a": {"b": {"c": 2, "d": 3}}}`,
		},
		{
			"array",
			`[1, 2, 3, 4, 5]`,
			`[1, 3, 5, 7]`,
		},
		{
			"mixed types",
			`{"str": "hello", "num": 42, "bool": true, "null": null, "arr": [1,2], "obj": {"k":"v"}}`,
			`{"str": "world", "num": 43, "bool": false, "null": "not null", "arr": [3], "obj": {"k":"v2","k2":"v3"}}`,
		},
	}

	for _, tt := range docs {
		t.Run(tt.name, func(t *testing.T) {
			patch, err := CreatePatch([]byte(tt.original), []byte(tt.modified))
			if err != nil {
				t.Fatalf("CreatePatch failed: %v", err)
			}

			result, err := ApplyPatch([]byte(tt.original), patch)
			if err != nil {
				// Print the patch for debugging
				patchJSON, _ := json.MarshalIndent(patch, "", "  ")
				t.Fatalf("ApplyPatch failed: %v\npatch: %s", err, patchJSON)
			}

			assertJSONEqual(t, tt.modified, string(result))
		})
	}
}
