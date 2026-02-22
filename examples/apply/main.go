// Example: Applying a JSON Patch to a document.
package main

import (
	"encoding/json"
	"fmt"
	"log"

	jsonpatch "github.com/robertjndw/go-json-patch"
)

func main() {
	// -------------------------------------------------------------------------
	// 1. Basic patch apply
	// -------------------------------------------------------------------------
	original := []byte(`{
		"name": "Alice",
		"age": 30,
		"email": "alice@example.com"
	}`)

	patch := []byte(`[
		{"op": "replace", "path": "/name",  "value": "Bob"},
		{"op": "add",     "path": "/phone", "value": "+1-555-0100"},
		{"op": "remove",  "path": "/email"}
	]`)

	result, err := jsonpatch.Apply(original, patch)
	if err != nil {
		log.Fatalf("Apply failed: %v", err)
	}

	fmt.Println("=== Basic apply ===")
	printJSON(result)

	// -------------------------------------------------------------------------
	// 2. Applying to an array document
	// -------------------------------------------------------------------------
	arrayDoc := []byte(`{"tags": ["go", "json"]}`)
	arrayPatch := []byte(`[
		{"op": "add",    "path": "/tags/-",  "value": "patch"},
		{"op": "remove", "path": "/tags/0"}
	]`)

	result, err = jsonpatch.Apply(arrayDoc, arrayPatch)
	if err != nil {
		log.Fatalf("Array apply failed: %v", err)
	}

	fmt.Println("\n=== Array operations ===")
	printJSON(result)

	// -------------------------------------------------------------------------
	// 3. Atomic failure — if any operation fails, none are applied
	// -------------------------------------------------------------------------
	doc := []byte(`{"status": "pending"}`)
	badPatch := []byte(`[
		{"op": "replace", "path": "/status", "value": "done"},
		{"op": "test",    "path": "/status", "value": "pending"}
	]`)

	_, err = jsonpatch.Apply(doc, badPatch)
	if err != nil {
		fmt.Printf("\n=== Atomic failure (expected) ===\nerror: %v\n", err)
	}

	// -------------------------------------------------------------------------
	// 4. Decode a patch once, apply it multiple times
	// -------------------------------------------------------------------------
	reusablePatch, err := jsonpatch.DecodePatch([]byte(`[
		{"op": "add", "path": "/processed", "value": true}
	]`))
	if err != nil {
		log.Fatalf("DecodePatch failed: %v", err)
	}

	docs := [][]byte{
		[]byte(`{"id": 1}`),
		[]byte(`{"id": 2}`),
		[]byte(`{"id": 3}`),
	}

	fmt.Println("\n=== Reusing a decoded patch ===")
	for _, d := range docs {
		r, err := jsonpatch.ApplyPatch(d, reusablePatch)
		if err != nil {
			log.Fatalf("ApplyPatch failed: %v", err)
		}
		fmt.Printf("  %s -> %s\n", d, r)
	}
}

func printJSON(data []byte) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Println(string(data))
		return
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}
