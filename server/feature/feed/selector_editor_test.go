package feed

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// TestNodeIDs checks that the node IDs of each golden page survive rendering
// and parsing again: the node at each ID is the node at that position in the
// page without IDs, including the texts, which carry no data-id.
func TestNodeIDs(t *testing.T) {
	files, err := filepath.Glob("testdata/simplified/*.html")
	if err != nil {
		t.Fatal(err)
	}
	texts := 0
	for _, f := range files {
		t.Run(strings.TrimSuffix(filepath.Base(f), ".html"), func(t *testing.T) {
			raw, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := parseGoldenPage(raw)
			if err != nil {
				t.Fatal(err)
			}
			var b bytes.Buffer
			if err := html.Render(&b, doc); err != nil {
				t.Fatal(err)
			}
			want := newPageIndex(pageBody(doc))

			page, err := withNodeIDs(b.String())
			if err != nil {
				t.Fatal(err)
			}
			_, got, err := parseEditorPage(page)
			if err != nil {
				t.Fatal(err)
			}

			if len(got.nodes) != len(want.nodes) {
				t.Fatalf("got %d nodes, want %d", len(got.nodes), len(want.nodes))
			}
			for i, n := range got.nodes {
				w := want.nodes[i]
				if n.Type != w.Type || n.Data != w.Data {
					t.Fatalf("node %d is %q, want %q", i, n.Data, w.Data)
				}
				if n.Type == html.TextNode {
					texts++
				}
			}
		})
	}
	if texts == 0 {
		t.Error("no page has a text node")
	}
}

// TestParseEditorPageRejectsWrongIDs checks that a page whose data-id of an
// element is not its position is rejected.
func TestParseEditorPageRejectsWrongIDs(t *testing.T) {
	raw, err := os.ReadFile("testdata/extract/list.html")
	if err != nil {
		t.Fatal(err)
	}
	page, err := withNodeIDs(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := parseEditorPage(page); err != nil {
		t.Fatalf("the page with node IDs: %v", err)
	}
	// The body is always node 0.
	wrong := strings.Replace(page, `<body data-id="0">`, `<body data-id="1">`, 1)
	if wrong == page {
		t.Fatalf("the page has no body with the ID 0: %s", page)
	}
	if _, _, err := parseEditorPage(wrong); err == nil {
		t.Error("got no error for a wrong data-id")
	}
}
