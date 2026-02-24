package jsonpatch

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers – reusable test fixtures
// ---------------------------------------------------------------------------

// makeObject builds a flat JSON object with n string keys.
//
//	{"key_0":"val_0", "key_1":"val_1", …}
func makeObject(n int) []byte {
	m := make(map[string]string, n)
	for i := range n {
		m["key_"+strconv.Itoa(i)] = "val_" + strconv.Itoa(i)
	}
	b, _ := json.Marshal(m)
	return b
}

// makeArray builds a JSON array with n integer elements.
//
//	[0, 1, 2, …]
func makeArray(n int) []byte {
	a := make([]int, n)
	for i := range n {
		a[i] = i
	}
	b, _ := json.Marshal(a)
	return b
}

// makeNestedObject builds a deeply-nested JSON object.
//
//	{"a":{"a":{"a":… "leaf" …}}}
func makeNestedObject(depth int) []byte {
	var inner interface{} = "leaf"
	for range depth {
		inner = map[string]interface{}{"a": inner}
	}
	b, _ := json.Marshal(inner)
	return b
}

// modifyObject changes roughly half the keys in the object.
func modifyObject(original []byte) []byte {
	var m map[string]interface{}
	_ = json.Unmarshal(original, &m)
	i := 0
	for k := range m {
		if i%2 == 0 {
			m[k] = "changed"
		}
		i++
	}
	b, _ := json.Marshal(m)
	return b
}

// ---------------------------------------------------------------------------
// Benchmark: Apply
// ---------------------------------------------------------------------------

func BenchmarkApply_SingleAdd(b *testing.B) {
	doc := []byte(`{"foo":"bar"}`)
	patch := []byte(`[{"op":"add","path":"/baz","value":"qux"}]`)
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(doc, patch)
	}
}

func BenchmarkApply_SingleReplace(b *testing.B) {
	doc := []byte(`{"foo":"bar"}`)
	patch := []byte(`[{"op":"replace","path":"/foo","value":"baz"}]`)
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(doc, patch)
	}
}

func BenchmarkApply_SingleRemove(b *testing.B) {
	doc := []byte(`{"foo":"bar","baz":"qux"}`)
	patch := []byte(`[{"op":"remove","path":"/baz"}]`)
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(doc, patch)
	}
}

func BenchmarkApply_Move(b *testing.B) {
	doc := []byte(`{"foo":{"bar":"baz"},"qux":{"corge":"grault"}}`)
	patch := []byte(`[{"op":"move","from":"/foo/bar","path":"/qux/thud"}]`)
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(doc, patch)
	}
}

func BenchmarkApply_Copy(b *testing.B) {
	doc := []byte(`{"foo":{"bar":"baz"}}`)
	patch := []byte(`[{"op":"copy","from":"/foo/bar","path":"/foo/qux"}]`)
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(doc, patch)
	}
}

func BenchmarkApply_Test(b *testing.B) {
	doc := []byte(`{"foo":"bar"}`)
	patch := []byte(`[{"op":"test","path":"/foo","value":"bar"}]`)
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(doc, patch)
	}
}

func BenchmarkApply_MultipleOps(b *testing.B) {
	doc := []byte(`{"foo":"bar","baz":"qux"}`)
	patch := []byte(`[
		{"op":"replace","path":"/foo","value":"new"},
		{"op":"add","path":"/added","value":true},
		{"op":"remove","path":"/baz"},
		{"op":"add","path":"/arr","value":[1,2,3]}
	]`)
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(doc, patch)
	}
}

// BenchmarkApply_LargeDocument tests applying a single operation to
// documents of increasing size.
func BenchmarkApply_LargeDocument(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, n := range sizes {
		doc := makeObject(n)
		patch := []byte(`[{"op":"add","path":"/newkey","value":"newval"}]`)
		b.Run(fmt.Sprintf("keys_%d", n), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = Apply(doc, patch)
			}
		})
	}
}

// BenchmarkApply_LargePatch tests applying many operations to a small document.
func BenchmarkApply_LargePatch(b *testing.B) {
	counts := []int{10, 50, 100}
	for _, n := range counts {
		doc := []byte(`{}`)
		ops := make([]Operation, n)
		for i := range n {
			ops[i], _ = NewOperation(OpAdd, "/key_"+strconv.Itoa(i), "val")
		}
		patchJSON, _ := json.Marshal(ops)

		b.Run(fmt.Sprintf("ops_%d", n), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = Apply(doc, patchJSON)
			}
		})
	}
}

// BenchmarkApply_DeepNested benchmarks apply on deeply-nested documents.
func BenchmarkApply_DeepNested(b *testing.B) {
	depths := []int{5, 10, 20}
	for _, d := range depths {
		doc := makeNestedObject(d)
		// Build a pointer that reaches the leaf.
		path := ""
		for range d {
			path += "/a"
		}
		patch := []byte(fmt.Sprintf(`[{"op":"replace","path":"%s","value":"replaced"}]`, path))
		b.Run(fmt.Sprintf("depth_%d", d), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = Apply(doc, patch)
			}
		})
	}
}

// BenchmarkApply_ArrayInsert benchmarks inserting into arrays of various sizes.
func BenchmarkApply_ArrayInsert(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, n := range sizes {
		arr := makeArray(n)
		doc := []byte(fmt.Sprintf(`{"items":%s}`, arr))
		patch := []byte(`[{"op":"add","path":"/items/0","value":999}]`)
		b.Run(fmt.Sprintf("len_%d", n), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = Apply(doc, patch)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmark: ApplyPatch (pre-decoded patch, avoids repeated decode cost)
// ---------------------------------------------------------------------------

func BenchmarkApplyPatch_PreDecoded(b *testing.B) {
	doc := []byte(`{"foo":"bar"}`)
	patchJSON := []byte(`[
		{"op":"add","path":"/x","value":1},
		{"op":"replace","path":"/foo","value":"baz"},
		{"op":"add","path":"/y","value":[1,2,3]},
		{"op":"test","path":"/foo","value":"baz"}
	]`)
	patch, _ := DecodePatch(patchJSON)
	b.ResetTimer()
	for b.Loop() {
		_, _ = ApplyPatch(doc, patch)
	}
}

// ---------------------------------------------------------------------------
// Benchmark: CreatePatch (diff)
// ---------------------------------------------------------------------------

func BenchmarkCreatePatch_SmallObject(b *testing.B) {
	original := []byte(`{"foo":"bar","baz":"qux"}`)
	modified := []byte(`{"foo":"changed","baz":"qux","added":"new"}`)
	b.ResetTimer()
	for b.Loop() {
		_, _ = CreatePatch(original, modified)
	}
}

func BenchmarkCreatePatch_IdenticalObjects(b *testing.B) {
	doc := makeObject(100)
	b.ResetTimer()
	for b.Loop() {
		_, _ = CreatePatch(doc, doc)
	}
}

func BenchmarkCreatePatch_LargeObject(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, n := range sizes {
		original := makeObject(n)
		modified := modifyObject(original)
		b.Run(fmt.Sprintf("keys_%d", n), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = CreatePatch(original, modified)
			}
		})
	}
}

func BenchmarkCreatePatch_ArrayGrow(b *testing.B) {
	original := makeArray(50)
	modified := makeArray(100)
	b.ResetTimer()
	for b.Loop() {
		_, _ = CreatePatch(original, modified)
	}
}

func BenchmarkCreatePatch_ArrayShrink(b *testing.B) {
	original := makeArray(100)
	modified := makeArray(50)
	b.ResetTimer()
	for b.Loop() {
		_, _ = CreatePatch(original, modified)
	}
}

func BenchmarkCreatePatch_DeepNested(b *testing.B) {
	depths := []int{5, 10, 20}
	for _, d := range depths {
		original := makeNestedObject(d)
		// Change the innermost value.
		var doc interface{}
		_ = json.Unmarshal(original, &doc)
		cur := doc
		for i := range d {
			m := cur.(map[string]interface{})
			if i == d-1 {
				m["a"] = "changed"
			} else {
				cur = m["a"]
			}
		}
		modified, _ := json.Marshal(doc)

		b.Run(fmt.Sprintf("depth_%d", d), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = CreatePatch(original, modified)
			}
		})
	}
}

// BenchmarkCreatePatch_RoundTrip measures diff + apply together.
func BenchmarkCreatePatch_RoundTrip(b *testing.B) {
	original := makeObject(100)
	modified := modifyObject(original)
	b.ResetTimer()
	for b.Loop() {
		patch, _ := CreatePatch(original, modified)
		_, _ = ApplyPatch(original, patch)
	}
}

// ---------------------------------------------------------------------------
// Benchmark: DecodePatch
// ---------------------------------------------------------------------------

func BenchmarkDecodePatch_Small(b *testing.B) {
	patchJSON := []byte(`[{"op":"add","path":"/foo","value":"bar"}]`)
	b.ResetTimer()
	for b.Loop() {
		_, _ = DecodePatch(patchJSON)
	}
}

func BenchmarkDecodePatch_Large(b *testing.B) {
	counts := []int{10, 50, 100}
	for _, n := range counts {
		ops := make([]map[string]interface{}, n)
		for i := range n {
			ops[i] = map[string]interface{}{
				"op":    "add",
				"path":  "/key_" + strconv.Itoa(i),
				"value": i,
			}
		}
		patchJSON, _ := json.Marshal(ops)

		b.Run(fmt.Sprintf("ops_%d", n), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = DecodePatch(patchJSON)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmark: MarshalPatch
// ---------------------------------------------------------------------------

func BenchmarkMarshalPatch(b *testing.B) {
	counts := []int{1, 10, 50}
	for _, n := range counts {
		ops := make(Patch, n)
		for i := range n {
			ops[i], _ = NewOperation(OpAdd, "/key_"+strconv.Itoa(i), "val")
		}
		b.Run(fmt.Sprintf("ops_%d", n), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = MarshalPatch(ops)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmark: ParsePointer
// ---------------------------------------------------------------------------

func BenchmarkParsePointer_Simple(b *testing.B) {
	for b.Loop() {
		_, _ = ParsePointer("/foo/bar")
	}
}

func BenchmarkParsePointer_Deep(b *testing.B) {
	depths := []int{5, 10, 20}
	for _, d := range depths {
		ptr := ""
		for i := range d {
			ptr += "/token" + strconv.Itoa(i)
		}
		b.Run(fmt.Sprintf("depth_%d", d), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_, _ = ParsePointer(ptr)
			}
		})
	}
}

func BenchmarkParsePointer_WithEscapes(b *testing.B) {
	// "~0" → "~", "~1" → "/"
	for b.Loop() {
		_, _ = ParsePointer("/foo~0bar/baz~1qux/deep~0~1path")
	}
}

// ---------------------------------------------------------------------------
// Benchmark: Pointer.Evaluate
// ---------------------------------------------------------------------------

func BenchmarkPointerEvaluate(b *testing.B) {
	doc := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "value",
			},
		},
	}
	ptr, _ := ParsePointer("/a/b/c")
	b.ResetTimer()
	for b.Loop() {
		_, _ = ptr.Evaluate(doc)
	}
}

// ---------------------------------------------------------------------------
// Benchmark: Pointer.Set
// ---------------------------------------------------------------------------

func BenchmarkPointerSet(b *testing.B) {
	ptr, _ := ParsePointer("/a/b/c")
	b.ResetTimer()
	for b.Loop() {
		doc := map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": "old",
				},
			},
		}
		_, _ = ptr.Set(doc, "new")
	}
}

// ---------------------------------------------------------------------------
// Benchmark: Pointer.Remove
// ---------------------------------------------------------------------------

func BenchmarkPointerRemove(b *testing.B) {
	ptr, _ := ParsePointer("/a/b/c")
	b.ResetTimer()
	for b.Loop() {
		doc := map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": "value",
				},
			},
		}
		_, _ = ptr.Remove(doc)
	}
}

// ---------------------------------------------------------------------------
// Benchmark: Operation.UnmarshalJSON
// ---------------------------------------------------------------------------

func BenchmarkOperationUnmarshal(b *testing.B) {
	data := []byte(`{"op":"replace","path":"/foo/bar","value":{"nested":true}}`)
	b.ResetTimer()
	for b.Loop() {
		var op Operation
		_ = json.Unmarshal(data, &op)
	}
}

// ---------------------------------------------------------------------------
// Benchmark: End-to-End realistic scenario
// ---------------------------------------------------------------------------

func BenchmarkEndToEnd_RealisticObject(b *testing.B) {
	original := []byte(`{
		"name": "John Doe",
		"age": 30,
		"email": "john@example.com",
		"address": {
			"street": "123 Main St",
			"city": "Springfield",
			"state": "IL",
			"zip": "62701"
		},
		"phones": [
			{"type": "home", "number": "555-1234"},
			{"type": "work", "number": "555-5678"}
		],
		"tags": ["admin", "user"],
		"active": true
	}`)

	modified := []byte(`{
		"name": "John Doe",
		"age": 31,
		"email": "john.doe@newdomain.com",
		"address": {
			"street": "456 Oak Ave",
			"city": "Springfield",
			"state": "IL",
			"zip": "62702"
		},
		"phones": [
			{"type": "home", "number": "555-1234"},
			{"type": "work", "number": "555-9999"},
			{"type": "mobile", "number": "555-0000"}
		],
		"tags": ["admin", "user", "manager"],
		"active": true,
		"role": "supervisor"
	}`)

	b.Run("CreatePatch", func(b *testing.B) {
		for b.Loop() {
			_, _ = CreatePatch(original, modified)
		}
	})

	b.Run("Apply", func(b *testing.B) {
		patch, _ := CreatePatch(original, modified)
		patchJSON, _ := MarshalPatch(patch)
		b.ResetTimer()
		for b.Loop() {
			_, _ = Apply(original, patchJSON)
		}
	})

	b.Run("RoundTrip", func(b *testing.B) {
		for b.Loop() {
			patch, _ := CreatePatch(original, modified)
			_, _ = ApplyPatch(original, patch)
		}
	})
}
