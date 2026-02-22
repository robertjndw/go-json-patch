// Example: Creating a JSON Patch by diffing two documents.
package main

import (
	"encoding/json"
	"fmt"
	"log"

	jsonpatch "github.com/robertjndw/go-json-patch"
)

func main() {
	// -------------------------------------------------------------------------
	// 1. Basic object diff
	// -------------------------------------------------------------------------
	original := []byte(`{
		"title":     "Hello",
		"body":      "World",
		"published": false
	}`)

	modified := []byte(`{
		"title":     "Hello World",
		"body":      "World",
		"published": true,
		"tags":      ["go", "patch"]
	}`)

	patch, err := jsonpatch.CreatePatch(original, modified)
	if err != nil {
		log.Fatalf("CreatePatch failed: %v", err)
	}

	fmt.Println("=== Object diff ===")
	printPatch(patch)

	// Verify: applying the patch to original should produce modified
	result, err := jsonpatch.ApplyPatch(original, patch)
	if err != nil {
		log.Fatalf("ApplyPatch failed: %v", err)
	}
	fmt.Println("Applied result:")
	printJSON(result)

	// -------------------------------------------------------------------------
	// 2. Nested object diff
	// -------------------------------------------------------------------------
	origUser := []byte(`{
		"user": {
			"name":  "Alice",
			"email": "alice@example.com",
			"role":  "viewer"
		}
	}`)

	modUser := []byte(`{
		"user": {
			"name":  "Alice",
			"email": "alice@newdomain.com",
			"role":  "admin"
		}
	}`)

	userPatch, err := jsonpatch.CreatePatch(origUser, modUser)
	if err != nil {
		log.Fatalf("CreatePatch failed: %v", err)
	}

	fmt.Println("\n=== Nested object diff ===")
	printPatch(userPatch)

	// -------------------------------------------------------------------------
	// 3. Identical documents produce an empty patch
	// -------------------------------------------------------------------------
	emptyPatch, err := jsonpatch.CreatePatch(original, original)
	if err != nil {
		log.Fatalf("CreatePatch failed: %v", err)
	}

	fmt.Printf("\n=== Identical documents -> %d operations ===\n", len(emptyPatch))

	// -------------------------------------------------------------------------
	// 4. Array diff
	// -------------------------------------------------------------------------
	origArr := []byte(`{"items": [1, 2, 3, 4, 5]}`)
	modArr := []byte(`{"items": [1, 2, 99, 4]}`)

	arrPatch, err := jsonpatch.CreatePatch(origArr, modArr)
	if err != nil {
		log.Fatalf("CreatePatch failed: %v", err)
	}

	fmt.Println("\n=== Array diff ===")
	printPatch(arrPatch)
}

func printPatch(patch jsonpatch.Patch) {
	b, _ := json.MarshalIndent(patch, "", "  ")
	fmt.Println(string(b))
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
