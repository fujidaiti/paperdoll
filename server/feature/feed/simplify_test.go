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

var update = flag.Bool("update", false, "update the golden files in testdata/webpage")

// webpages returns the names of the test pages in testdata/webpage.
func webpages(t *testing.T) []string {
	t.Helper()
	dirs, err := filepath.Glob("testdata/webpage/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) == 0 {
		t.Fatal("no test pages")
	}
	names := make([]string, 0, len(dirs))
	for _, d := range dirs {
		names = append(names, filepath.Base(d))
	}
	return names
}

// simplifyWebpage returns the simplified page of the test page name. version
// is "" for the page as first fetched, and "later" for its later version.
func simplifyWebpage(t *testing.T, name, version string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata/webpage", name, version, "source.html"))
	if err != nil {
		t.Fatal(err)
	}
	// The test URL of a page is its directory name with "-" read as "/".
	page, err := SimplifyPage(raw, "http://"+strings.ReplaceAll(name, "-", "/"))
	if err != nil {
		t.Fatal(err)
	}
	return page
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

// parseGoldenPage parses a simplified page that was written by hand or
// formatted by Prettier, which adds whitespace between elements and inside
// texts. SimplifyPage never writes a text of only whitespace and normalizes the
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

// renderGoldenPage returns the simplified page raw in a form that does not
// depend on its formatting. The parser reorders the attributes of some
// elements, so a page is compared as it renders after parsing.
func renderGoldenPage(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := parseGoldenPage(raw)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := html.Render(&b, doc); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// checkGolden compares got, an HTML page or compact JSON, with the golden file
// at path, or writes got to it with -update. A file is written without
// formatting, which Prettier does on commit, and is compared without the
// whitespace that Prettier adds. On a difference, both are split after each
// ">" or "}", so that the diff shows the parts that differ.
func checkGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	switch filepath.Ext(path) {
	case ".json":
		var b bytes.Buffer
		if err := json.Compact(&b, want); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		want = b.Bytes()
	case ".html":
		want, got = renderGoldenPage(t, want), renderGoldenPage(t, got)
	}
	split := regexp.MustCompile(`[>}]`)
	lines := func(b []byte) []string {
		return strings.SplitAfter(split.ReplaceAllString(string(b), "$0\n"), "\n")
	}
	if diff := cmp.Diff(lines(want), lines(got)); diff != "" {
		t.Errorf("%s differs (-want +got):\n%s", path, diff)
	}
}

// TestSimplifyPageGolden checks the simplified page of each version of each
// test page against its golden file, and that the page still holds the posts
// of its fixture: the box of a post is an element below the body, and the node
// at the ID of each field has the value of that field in the fixture. The
// fixture is checked even with -update, since no test rewrites a fixture.
func TestSimplifyPageGolden(t *testing.T) {
	for _, name := range webpages(t) {
		for _, version := range []string{"", "later"} {
			t.Run(filepath.Join(name, version), func(t *testing.T) {
				dir := filepath.Join("testdata/webpage", name, version)
				page := simplifyWebpage(t, name, version)
				checkGolden(t, filepath.Join(dir, "simplified.golden.html"), []byte(page))

				_, index, err := parseEditorPage(page)
				if err != nil {
					t.Fatal(err)
				}
				var posts []FoundPost
				readJSON(t, filepath.Join(dir, "posts.fixture.json"), &posts)
				for _, p := range posts {
					box := p.IDs["box"]
					if box <= 0 || box >= len(index.nodes) || index.nodes[box].Type != html.ElementNode {
						t.Errorf("post %d: the box is not an element below the body", box)
						continue
					}
					for f, want := range p.Values {
						id, ok := p.IDs[string(f)]
						if !ok {
							t.Errorf("post %d: the %s has a value but no node", box, f)
							continue
						}
						if id < 0 || id >= len(index.nodes) {
							t.Errorf("post %d: the %s is the node %d, which is not on the page", box, f, id)
							continue
						}
						if got, _ := f.valueOf(index.nodes[id]); got != want {
							t.Errorf("post %d: the %s at node %d is %q, want %q", box, f, id, got, want)
						}
					}
				}
			})
		}
	}
}
