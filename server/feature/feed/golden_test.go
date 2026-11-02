package feed

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
)

var update = flag.Bool("update", false, "update the golden files in testdata/simplified and testdata/selector")

// checkGolden compares got, an HTML page or compact JSON, with the golden file
// at path, or writes got to it with -update. A file is written without
// formatting, which Prettier does on commit, and is compared without the
// whitespace that Prettier adds (see parseGoldenPage for an HTML page). On a
// difference, both are split after each ">" or "}", so that the diff shows the
// parts that differ.
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
	switch filepath.Ext(path) {
	case ".json":
		var b bytes.Buffer
		if err := json.Compact(&b, want); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		want = b.Bytes()
	case ".html":
		doc, err := parseGoldenPage(want)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		var b bytes.Buffer
		if err := html.Render(&b, doc); err != nil {
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

// parseGoldenPage parses a simplified page in testdata/simplified. Prettier
// formats the pages, which adds whitespace between elements and inside texts.
// SimplifyPage never writes a text of only whitespace and normalizes the
// whitespace of every text, so doing the same after parsing gives back the
// page that it wrote.
func parseGoldenPage(raw []byte) (*html.Node, error) {
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			if c.Type == html.TextNode {
				if c.Data = normalizeSpace(c.Data); c.Data == "" {
					n.RemoveChild(c)
				}
			} else {
				visit(c)
			}
			c = next
		}
	}
	visit(doc)
	return doc, nil
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
			page, err := parseGoldenPage(raw)
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

// TestLabelFoundPostsGolden checks that the selectors generated from the posts
// that the golden selectors of each test page find (see labelFoundPosts) find
// the same posts with the same values, as when a user starts an editor of the
// feed and commits it without changes. A post that another selector also finds
// is extracted anyway, so the post nodes are compared too.
func TestLabelFoundPostsGolden(t *testing.T) {
	files, err := filepath.Glob("testdata/selector/*.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		t.Run(name, func(t *testing.T) {
			var golden []PostSelector
			readJSON(t, f, &golden)
			raw, err := os.ReadFile(filepath.Join("testdata/simplified", name+".html"))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := parseGoldenPage(raw)
			if err != nil {
				t.Fatal(err)
			}
			index := newPageIndex(pageBody(doc))
			excluded := map[string]bool{}
			for _, m := range golden {
				for _, l := range m.Exclude {
					excluded[l] = true
				}
			}

			selectors := index.generateSelectors(index.labelFoundPosts(golden), excluded)

			postNodes := func(selectors []PostSelector) []int {
				var ids []int
				for _, m := range selectors {
					for _, n := range m.selectPosts(index.body) {
						if !slices.Contains(ids, index.index[n]) {
							ids = append(ids, index.index[n])
						}
					}
				}
				slices.Sort(ids)
				return ids
			}
			if d := cmp.Diff(postNodes(golden), postNodes(selectors)); d != "" {
				t.Errorf("post nodes differ (-golden +got):\n%s", d)
			}
			if d := cmp.Diff(ExtractPosts(doc, golden), ExtractPosts(doc, selectors)); d != "" {
				t.Errorf("extracted posts differ (-golden +got):\n%s", d)
			}
		})
	}
}
