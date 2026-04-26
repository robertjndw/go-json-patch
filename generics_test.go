package jsonpatch

import (
	"encoding/json"
	"testing"
)

// customBytes is a custom type with underlying type []byte, used to test the
// ~[]byte part of the Document constraint.
type customBytes []byte

// customString is a custom type with underlying type string, used to test the
// ~string part of the Document constraint.
type customString string

// ---------------------------------------------------------------------------
// Apply – string inputs
// ---------------------------------------------------------------------------

func TestApply_StringInputs(t *testing.T) {
	doc := `{"foo": "bar"}`
	patch := `[{"op": "add", "path": "/baz", "value": "qux"}]`

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}

	// result should be a string
	assertJSONEqual(t, `{"baz": "qux", "foo": "bar"}`, result)
}

func TestApply_StringInputs_Replace(t *testing.T) {
	doc := `{"name": "Alice", "age": 30}`
	patch := `[{"op": "replace", "path": "/name", "value": "Bob"}]`

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}

	assertJSONEqual(t, `{"name": "Bob", "age": 30}`, result)
}

func TestApply_StringInputs_Remove(t *testing.T) {
	doc := `{"foo": "bar", "baz": "qux"}`
	patch := `[{"op": "remove", "path": "/baz"}]`

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}

	assertJSONEqual(t, `{"foo": "bar"}`, result)
}

func TestApply_StringInputs_MultipleOps(t *testing.T) {
	doc := `{"name": "Alice", "email": "alice@example.com"}`
	patch := `[
		{"op": "replace", "path": "/name", "value": "Bob"},
		{"op": "add", "path": "/phone", "value": "+1-555-0100"},
		{"op": "remove", "path": "/email"}
	]`

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}

	assertJSONEqual(t, `{"name": "Bob", "phone": "+1-555-0100"}`, result)
}

func TestApply_StringInputs_Array(t *testing.T) {
	doc := `{"tags": ["go", "json"]}`
	patch := `[{"op": "add", "path": "/tags/-", "value": "patch"}]`

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}

	assertJSONEqual(t, `{"tags": ["go", "json", "patch"]}`, result)
}

// ---------------------------------------------------------------------------
// ApplyPatch – string inputs
// ---------------------------------------------------------------------------

func TestApplyPatch_StringInput(t *testing.T) {
	patchOps, err := DecodePatch(`[{"op": "add", "path": "/baz", "value": "qux"}]`)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(`{"foo": "bar"}`, patchOps)
	if err != nil {
		t.Fatal(err)
	}

	assertJSONEqual(t, `{"baz": "qux", "foo": "bar"}`, result)
}

// ---------------------------------------------------------------------------
// DecodePatch – string input
// ---------------------------------------------------------------------------

func TestDecodePatch_StringInput(t *testing.T) {
	patch, err := DecodePatch(`[
		{"op": "add",     "path": "/foo", "value": 1},
		{"op": "remove",  "path": "/bar"},
		{"op": "replace", "path": "/baz", "value": "qux"}
	]`)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(patch))
	}
	if patch[0].Op != OpAdd {
		t.Errorf("expected first op to be %q, got %q", OpAdd, patch[0].Op)
	}
	if patch[1].Op != OpRemove {
		t.Errorf("expected second op to be %q, got %q", OpRemove, patch[1].Op)
	}
	if patch[2].Op != OpReplace {
		t.Errorf("expected third op to be %q, got %q", OpReplace, patch[2].Op)
	}
}

// ---------------------------------------------------------------------------
// CreatePatch – string inputs
// ---------------------------------------------------------------------------

func TestCreatePatch_StringInputs(t *testing.T) {
	original := `{"foo": "bar"}`
	modified := `{"foo": "bar", "baz": "qux"}`

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the patch can be applied (using string ApplyPatch)
	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, modified, result)
}

func TestCreatePatch_StringInputs_Nested(t *testing.T) {
	original := `{"user": {"name": "Alice", "role": "viewer"}}`
	modified := `{"user": {"name": "Alice", "role": "admin"}}`

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, modified, result)
}

// ---------------------------------------------------------------------------
// Custom types (testing the ~ approximation constraint)
// ---------------------------------------------------------------------------

func TestApply_CustomBytesType(t *testing.T) {
	doc := customBytes(`{"foo": "bar"}`)
	patch := customBytes(`[{"op": "add", "path": "/baz", "value": "qux"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}

	// result is customBytes
	assertJSONEqual(t, `{"baz": "qux", "foo": "bar"}`, string(result))
}

func TestApply_CustomStringType(t *testing.T) {
	doc := customString(`{"foo": "bar"}`)
	patch := customString(`[{"op": "add", "path": "/baz", "value": "qux"}]`)

	result, err := Apply(doc, patch)
	if err != nil {
		t.Fatal(err)
	}

	// result is customString
	assertJSONEqual(t, `{"baz": "qux", "foo": "bar"}`, string(result))
}

func TestCreatePatch_CustomStringType(t *testing.T) {
	original := customString(`{"foo": "bar"}`)
	modified := customString(`{"foo": "baz"}`)

	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	if len(patch) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(patch))
	}
	if patch[0].Op != OpReplace {
		t.Errorf("expected replace op, got %q", patch[0].Op)
	}
}

// ---------------------------------------------------------------------------
// Round-trip: string → patch → string
// ---------------------------------------------------------------------------

func TestRoundTrip_StringWorkflow(t *testing.T) {
	// A complete workflow using only strings — no []byte anywhere
	original := `{"items": [1, 2, 3], "count": 3}`
	modified := `{"items": [1, 2, 3, 4], "count": 4}`

	// Create patch from string documents
	patch, err := CreatePatch(original, modified)
	if err != nil {
		t.Fatal(err)
	}

	// Marshal the patch to JSON (for transport/storage)
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		t.Fatal(err)
	}

	// Decode from string
	decoded, err := DecodePatch(string(patchJSON))
	if err != nil {
		t.Fatal(err)
	}

	// Apply to string document
	result, err := ApplyPatch(original, decoded)
	if err != nil {
		t.Fatal(err)
	}

	assertJSONEqual(t, modified, result)
}

// ---------------------------------------------------------------------------
// Error cases with string inputs
// ---------------------------------------------------------------------------

func TestApply_StringInputs_InvalidJSON(t *testing.T) {
	_, err := Apply(`not valid json`, `[{"op": "add", "path": "/foo", "value": 1}]`)
	if err == nil {
		t.Fatal("expected error for invalid JSON document")
	}
}

func TestApply_StringInputs_InvalidPatch(t *testing.T) {
	_, err := Apply(`{"foo": "bar"}`, `not a patch`)
	if err == nil {
		t.Fatal("expected error for invalid patch JSON")
	}
}

func TestCreatePatch_StringInputs_InvalidOriginal(t *testing.T) {
	_, err := CreatePatch(`not json`, `{"foo": "bar"}`)
	if err == nil {
		t.Fatal("expected error for invalid original JSON")
	}
}
