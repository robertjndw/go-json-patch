package jsonpatch

import (
	"encoding/json"
	"testing"
)

func mustParseJSON(s string) interface{} {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic(err)
	}
	return v
}

func TestParsePointer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantStr string
	}{
		{"empty string (root)", "", false, ""},
		{"single key", "/foo", false, "/foo"},
		{"nested path", "/a/b/c", false, "/a/b/c"},
		{"array index", "/foo/0", false, "/foo/0"},
		{"tilde escape ~0", "/~0", false, "/~0"},
		{"slash escape ~1", "/~1", false, "/~1"},
		{"combined escape", "/~01", false, "/~01"},
		{"missing leading slash", "foo", true, ""},
		{"invalid escape ~2", "/~2", true, ""},
		{"dangling tilde at end", "/foo~", true, ""},
		{"invalid escape ~a", "/~a", true, ""},
		{"tilde at end of multi-token", "/foo/bar~", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParsePointer(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePointer(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && p.String() != tt.wantStr {
				t.Errorf("ParsePointer(%q).String() = %q, want %q", tt.input, p.String(), tt.wantStr)
			}
		})
	}
}

func TestPointerEscaping(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"tilde", "~", "~0"},
		{"slash", "/", "~1"},
		{"tilde-one literal", "~1", "~01"},
		{"no escape needed", "foo", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapePointerToken(tt.token)
			if got != tt.want {
				t.Errorf("escapePointerToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
			back := unescapePointerToken(got)
			if back != tt.token {
				t.Errorf("round-trip failed: %q -> %q -> %q", tt.token, got, back)
			}
		})
	}
}

func TestPointerEvaluate(t *testing.T) {
	doc := mustParseJSON(`{
		"foo": ["bar", "baz"],
		"": 0,
		"a/b": 1,
		"c%d": 2,
		"e^f": 3,
		"g|h": 4,
		" ": 7,
		"m~n": 8
	}`)

	tests := []struct {
		pointer string
		want    interface{}
	}{
		{"/foo", []interface{}{"bar", "baz"}},
		{"/foo/0", "bar"},
		{"/", float64(0)},
		{"/a~1b", float64(1)},
		{"/m~0n", float64(8)},
	}

	for _, tt := range tests {
		t.Run(tt.pointer, func(t *testing.T) {
			p, err := ParsePointer(tt.pointer)
			if err != nil {
				t.Fatal(err)
			}
			got, err := p.Evaluate(doc)
			if err != nil {
				t.Fatal(err)
			}
			if !jsonEqual(got, tt.want) {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.pointer, got, tt.want)
			}
		})
	}
}

func TestPointerIsPrefixOf(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"/a/b", "/a/b/c", true},
		{"/a", "/a/b", true},
		{"/a/b", "/a/b", false},
		{"/a/b/c", "/a/b", false},
		{"", "/a", true},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_prefix_of_"+tt.b, func(t *testing.T) {
			pa, _ := ParsePointer(tt.a)
			pb, _ := ParsePointer(tt.b)
			if got := pa.IsPrefixOf(pb); got != tt.want {
				t.Errorf("(%q).IsPrefixOf(%q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
