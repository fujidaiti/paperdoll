package feed

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
)

var update = flag.Bool("update", false, "update the golden files in testdata/simplified and testdata/selector")

// checkGolden compares got, an HTML page or compact JSON, with the golden file
// at path, or writes got to it with -update. A JSON file is written without
// formatting, which Prettier does on commit, and is compared without the
// whitespace between JSON tokens. On a difference, both are split after each
// ">" or "}", so that the diff shows the parts that differ.
func checkGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if filepath.Ext(path) == ".json" {
		var b bytes.Buffer
		if err := json.Compact(&b, want); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		want = b.Bytes()
	}
	split := regexp.MustCompile(`[>}]`)
	lines := func(b []byte) []string {
		return strings.SplitAfter(split.ReplaceAllString(string(b), "$0\n"), "\n")
	}
	if diff := cmp.Diff(lines(want), lines(got)); diff != "" {
		t.Errorf("%s differs (-want +got):\n%s", path, diff)
	}
}

// TestSimplifyPageGolden checks that the simplified page of each test page in
// testdata/source is the page in testdata/simplified. It runs before
// TestPerfectSelectorsGolden, which reads the simplified pages.
func TestSimplifyPageGolden(t *testing.T) {
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
			page, err := SimplifyPage(raw, "https://"+strings.ReplaceAll(name, "-", "/"))
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, filepath.Join("testdata/simplified", name+".html"), []byte(page))
		})
	}
}

// TestPerfectSelectorsGolden checks that the selectors built on the simplified
// page of each test page from its fixture are the selectors in
// testdata/selector. A selector file has the JSON that the database stores for
// the page.
func TestPerfectSelectorsGolden(t *testing.T) {
	files, err := filepath.Glob("testdata/fixture/*.fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".fixture.json")
		t.Run(name, func(t *testing.T) {
			var posts []ExtractedPost
			readJSON(t, f, &posts)
			raw, err := os.ReadFile(filepath.Join("testdata/simplified", name+".html"))
			if err != nil {
				t.Fatal(err)
			}
			page, err := html.Parse(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			selectors, marks, err := perfectSelectors(page, posts)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%d marks for %d posts", marks, len(posts))
			got, err := json.Marshal(selectors)
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, filepath.Join("testdata/selector", name+".json"), got)
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
