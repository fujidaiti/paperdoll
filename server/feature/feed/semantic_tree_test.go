package feed

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSemanticTree renders the semantic tree of every saved page into
// testdata/semantic/, one JSON file per page, and writes the file on every
// run. The files are not an expected result to compare against: the tree is an
// experiment, and the point of writing them is to read them and to see in a
// diff what a change to the folding rule does to every page at once.
func TestSemanticTree(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no fixture found in testdata")
	}
	out := filepath.Join("testdata", "semantic")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".fixture.json")
		t.Run(name, func(t *testing.T) {
			page := readFixture(t, file).URL
			u, err := url.Parse(page)
			if err != nil {
				t.Fatal(err)
			}
			f, err := os.Open(filepath.Join("testdata", name+".html"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()

			root, err := BuildSemanticTree(f, *u)
			if err != nil {
				t.Fatal(err)
			}
			var b bytes.Buffer
			enc := json.NewEncoder(&b)
			enc.SetIndent("", "  ")
			// A text of the page may hold "<", ">" or "&", and an encoder
			// writes them as escape sequences unless this is turned off.
			enc.SetEscapeHTML(false)
			if err := enc.Encode(root); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(out, name+".json"), b.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s: %d nodes, depth %d", name, countNodes(root), depth(root))
		})
	}
}

func countNodes(n *SemanticNode) int {
	if n == nil {
		return 0
	}
	c := 1
	for _, k := range n.Children {
		c += countNodes(k)
	}
	return c
}

func depth(n *SemanticNode) int {
	if n == nil {
		return 0
	}
	d := 0
	for _, k := range n.Children {
		if x := depth(k); x > d {
			d = x
		}
	}
	return d + 1
}
