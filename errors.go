package jsonpatch

import "fmt"

// TestFailedError is returned when a "test" operation finds a mismatch.
type TestFailedError struct {
	Path     string
	Expected interface{}
	Actual   interface{}
}

func (e *TestFailedError) Error() string {
	return fmt.Sprintf("test failed: value at %q does not match: got %v, expected %v",
		e.Path, e.Actual, e.Expected)
}

// PathNotFoundError is returned when a JSON Pointer path does not exist in the document.
type PathNotFoundError struct {
	Path string
}

func (e *PathNotFoundError) Error() string {
	return fmt.Sprintf("path not found: %q", e.Path)
}

// IndexOutOfBoundsError is returned when an array index is out of range.
type IndexOutOfBoundsError struct {
	Index  int
	Length int
}

func (e *IndexOutOfBoundsError) Error() string {
	return fmt.Sprintf("array index %d out of bounds (length %d)", e.Index, e.Length)
}

// InvalidOperationError wraps an error that occurred while processing a specific
// operation within a patch. Index is the zero-based position of the operation
// in the patch array.
type InvalidOperationError struct {
	Index int
	Op    OpType
	Path  string
	Cause error
}

func (e *InvalidOperationError) Error() string {
	return fmt.Sprintf("operation %d (%s %s) failed: %v", e.Index, e.Op, e.Path, e.Cause)
}

func (e *InvalidOperationError) Unwrap() error {
	return e.Cause
}
