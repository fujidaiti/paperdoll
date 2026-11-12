package feed

import (
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
)

// TestPositionPath checks that the path of a node by its position selects only
// that node, when its siblings have the same tag, on
// testdata/position-path.html, whose target node has the class "target".
func TestPositionPath(t *testing.T) {
	raw, err := os.ReadFile("testdata/position-path.html")
	if err != nil {
		t.Fatal(err)
	}
	page, err := SimplifyPage(raw, "https://example.com/")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	index := newPageIndex(pageBody(doc))
	i := slices.IndexFunc(index.nodes, func(n *html.Node) bool {
		return slices.Contains(classesOf(n), "target")
	})
	if i < 0 {
		t.Fatalf("no node with the class \"target\": %s", page)
	}
	target := index.nodes[i]

	path := index.positionPath(target)

	want := []Step{{Tag: "div", Nth: new(0)}, {Tag: "nav", Nth: new(1)}}
	if d := cmp.Diff(want, path); d != "" {
		t.Errorf("path differs (-want +got):\n%s", d)
	}
	rules := ExtractionRules{ExcludeNodes: [][]Step{path}}
	if got := slices.Collect(maps.Keys(rules.excludedNodes(index.body))); !slices.Equal(got, []*html.Node{target}) {
		t.Errorf("the path selects %d nodes, want only the target", len(got))
	}
}
