package feed

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "update the golden files in testdata/tree and testdata/matcher")

// checkGolden compares got with the golden file at path, or writes got to it
// with -update. The file is written without formatting, which Prettier does
// on commit, and is compared without the whitespace between JSON tokens.
func checkGolden(t *testing.T, path string, got any) {
	t.Helper()
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	var want bytes.Buffer
	if err := json.Compact(&want, raw); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if diff := cmp.Diff(data, want.Bytes(), nil); diff != "" {
		t.Errorf("generated JSON differs from golden:\n%s", diff)
	}
}

// TestSemanticTreeGolden checks that simplifying each test page in
// testdata/source gives its semantic tree in testdata/tree. It runs before
// TestPerfectMatchersGolden, which reads the trees.
func TestSemanticTreeGolden(t *testing.T) {
	files, err := filepath.Glob("testdata/source/*.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".html")
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := SimplifyPage(raw, "https://"+strings.ReplaceAll(name, "-", "/"))
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, filepath.Join("testdata/tree", name+".json"), doc)
		})
	}
}

// TestPerfectMatchersGolden checks that the matchers built on the semantic
// tree of each test page from its fixture are the matchers in
// testdata/matcher. A matcher file has the JSON that the database stores for
// the page.
func TestPerfectMatchersGolden(t *testing.T) {
	files, err := filepath.Glob("testdata/fixture/*.fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".fixture.json")
		t.Run(name, func(t *testing.T) {
			var posts []ExtractedPost
			readJSON(t, f, &posts)
			var doc PageDocument
			readJSON(t, filepath.Join("testdata/tree", name+".json"), &doc)
			matchers, err := perfectMatchers(&doc, posts)
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, filepath.Join("testdata/matcher", name+".json"), matchers)
		})
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
