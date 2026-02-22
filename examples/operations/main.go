// Example: Building patch operations programmatically.
package main

import (
	"encoding/json"
	"fmt"
	"log"

	jsonpatch "github.com/robertjndw/go-json-patch"
)

func main() {
	// -------------------------------------------------------------------------
	// 1. Build a patch using helper constructors
	// -------------------------------------------------------------------------
	addOp, err := jsonpatch.NewOperation(jsonpatch.OpAdd, "/city", "San Francisco")
	if err != nil {
		log.Fatalf("NewOperation failed: %v", err)
	}

	replaceOp, err := jsonpatch.NewOperation(jsonpatch.OpReplace, "/name", "Bob")
	if err != nil {
		log.Fatalf("NewOperation failed: %v", err)
	}

	removeOp := jsonpatch.NewRemoveOperation("/temp")

	// move and copy have dedicated constructors
	moveOp := jsonpatch.NewMoveOperation("/draft", "/published")
	copyOp := jsonpatch.NewCopyOperation("/published", "/archive/2026")

	patch := jsonpatch.Patch{addOp, replaceOp, removeOp, moveOp, copyOp}

	fmt.Println("=== Built patch ===")
	b, _ := json.MarshalIndent(patch, "", "  ")
	fmt.Println(string(b))

	// -------------------------------------------------------------------------
	// 2. All six operations applied to a document
	// -------------------------------------------------------------------------
	doc := []byte(`{
		"name":      "Alice",
		"temp":      "temporary",
		"draft":     "draft content",
		"archive":   {}
	}`)

	// Remove the move/copy ops from our patch for this simpler demo
	demoPatch := jsonpatch.Patch{
		mustOp(jsonpatch.NewOperation(jsonpatch.OpTest, "/name", "Alice")),
		mustOp(jsonpatch.NewOperation(jsonpatch.OpAdd, "/city", "New York")),
		mustOp(jsonpatch.NewOperation(jsonpatch.OpReplace, "/name", "Bob")),
		jsonpatch.NewRemoveOperation("/temp"),
		jsonpatch.NewCopyOperation("/draft", "/draft_backup"),
		jsonpatch.NewMoveOperation("/draft", "/published"),
	}

	result, err := jsonpatch.ApplyPatch(doc, demoPatch)
	if err != nil {
		log.Fatalf("ApplyPatch failed: %v", err)
	}

	fmt.Println("\n=== All six operations applied ===")
	printJSON(result)

	// -------------------------------------------------------------------------
	// 3. Serialize a patch to JSON and decode it back
	// -------------------------------------------------------------------------
	patchJSON, err := jsonpatch.MarshalPatch(demoPatch)
	if err != nil {
		log.Fatalf("MarshalPatch failed: %v", err)
	}

	roundTripped, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		log.Fatalf("DecodePatch failed: %v", err)
	}

	fmt.Printf("\n=== Round-tripped patch: %d operations ===\n", len(roundTripped))
	for _, op := range roundTripped {
		fmt.Printf("  op=%-8s  path=%s\n", op.Op, op.Path)
	}

	// -------------------------------------------------------------------------
	// 4. test operation failure is descriptive
	// -------------------------------------------------------------------------
	_, err = jsonpatch.Apply(
		[]byte(`{"status": "active"}`),
		[]byte(`[{"op": "test", "path": "/status", "value": "inactive"}]`),
	)
	if err != nil {
		fmt.Printf("\n=== Test failure message ===\n%v\n", err)
	}
}

func mustOp(op jsonpatch.Operation, err error) jsonpatch.Operation {
	if err != nil {
		log.Fatalf("NewOperation: %v", err)
	}
	return op
}

func printJSON(data []byte) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Println(string(data))
		return
	}
	out, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(out))
}
