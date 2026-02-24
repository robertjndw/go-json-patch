# go-json-patch

A Go library to create and apply [RFC 6902](https://datatracker.ietf.org/doc/html/rfc6902) JSON Patch documents.

## Installation

```bash
go get github.com/robertjndw/go-json-patch
```

## Features

- **Apply** a JSON Patch document to a target JSON document
- **Create** a JSON Patch by diffing two JSON documents
- Full support for all six RFC 6902 operations: `add`, `remove`, `replace`, `move`, `copy`, `test`
- Strict JSON Pointer (RFC 6901) parsing with `~` and `/` escaping and escape validation
- Atomic patch application — if any operation fails, the entire patch is rejected
- Handles `null` values, nested objects, and arrays correctly

## Usage

### Apply a Patch

```go
package main

import (
    "fmt"
    "log"

    jsonpatch "github.com/robertjndw/go-json-patch"
)

func main() {
    original := []byte(`{"foo": "bar"}`)
    patch := []byte(`[
        {"op": "add", "path": "/baz", "value": "qux"},
        {"op": "replace", "path": "/foo", "value": "updated"}
    ]`)

    result, err := jsonpatch.Apply(original, patch)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(result))
    // Output: {"baz":"qux","foo":"updated"}
}
```

### Create a Patch (Diff)

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    jsonpatch "github.com/robertjndw/go-json-patch"
)

func main() {
    original := []byte(`{"foo": "bar", "baz": "qux"}`)
    modified := []byte(`{"foo": "bar", "baz": "updated", "new": true}`)

    patch, err := jsonpatch.CreatePatch(original, modified)
    if err != nil {
        log.Fatal(err)
    }

    patchJSON, _ := json.MarshalIndent(patch, "", "  ")
    fmt.Println(string(patchJSON))
    // Output:
    // [
    //   {"op": "replace", "path": "/baz", "value": "updated"},
    //   {"op": "add", "path": "/new", "value": true}
    // ]
}
```

### Decode and Apply a Patch Separately

```go
// Decode a patch document once, apply it multiple times
patch, err := jsonpatch.DecodePatch(patchJSON)
if err != nil {
    log.Fatal(err)
}

result, err := jsonpatch.ApplyPatch(documentJSON, patch)
if err != nil {
    log.Fatal(err)
}
```

### Build Operations Programmatically

```go
patch := jsonpatch.Patch{}

op1, _ := jsonpatch.NewOperation(jsonpatch.OpAdd, "/foo", "bar")
op2, _ := jsonpatch.NewOperation(jsonpatch.OpReplace, "/count", 42)
op3 := jsonpatch.NewMoveOperation("/old", "/new")
op4 := jsonpatch.NewCopyOperation("/source", "/target")

patch = append(patch, op1, op2, op3, op4)

patchJSON, _ := jsonpatch.MarshalPatch(patch)
```

## Supported Operations

| Operation | Description |
|-----------|-------------|
| `add`     | Add a value to an object or insert into an array |
| `remove`  | Remove a value at a target location |
| `replace` | Replace the value at a target location |
| `move`    | Remove a value from one location and add it to another |
| `copy`    | Copy a value from one location to another |
| `test`    | Test that a value at a target location equals a specified value |

## Specification Compliance

This library aims for strict RFC 6902 compliance including:

- All operations from Section 4 (add, remove, replace, move, copy, test)
- Mandatory `path` member validation for all operations and `from` member validation for move/copy
- JSON Pointer (RFC 6901) path resolution with proper `~0` and `~1` escaping and strict escape validation
- Error handling per Section 5 (atomic patch application)
- Array index handling including the `-` end-of-array append syntax
- Deep equality comparison for the `test` operation per Section 4.6
- Ignoring unrecognized members in operation objects per Section 4

## Performance & Benchmarking

The library includes comprehensive benchmarks covering all major operations at various scales. Contributors can use these to verify performance improvements or regressions.

### Running Benchmarks

Run all benchmarks with memory allocation stats:

```bash
go test -bench=. -benchmem
```

### Comparing Performance Changes

To compare performance before and after a change, use [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

1. Install benchstat:
   ```bash
   go install golang.org/x/perf/cmd/benchstat@latest
   ```

2. Collect baseline benchmarks:
   ```bash
   go test -bench=. -benchmem -count=5 > old.txt
   ```

3. Make your changes and collect new benchmarks:
   ```bash
   go test -bench=. -benchmem -count=5 > new.txt
   ```

4. Compare results:
   ```bash
   benchstat old.txt new.txt
   ```

This will show you the performance delta for each benchmark function, including ns/op (time), B/op (memory), and allocs/op changes.

### Benchmark Coverage

The benchmarks include:
- **Apply operations**: Single ops, multi-op sequences, large documents, large patches, nested structures, array operations
- **CreatePatch**: Small and large objects, identical documents, arrays, deep nesting, round-trip scenarios
- **Serialization**: DecodePatch, MarshalPatch
- **Pointer operations**: Parsing, evaluation, modification
- **End-to-end**: Realistic complex object scenarios

## License
MIT
