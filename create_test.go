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

func TestCreatePatch_InvalidOriginalJSON(t *testing.T) {
	_, err := CreatePatch([]byte(`not json`), []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for invalid original JSON")
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
