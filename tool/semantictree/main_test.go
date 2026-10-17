package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestKeepsFixtureValues checks that simplifying each test page keeps every
// value of the posts in its fixture, so that a matcher can find them in the
// tree.
func TestKeepsFixtureValues(t *testing.T) {
	const testdata = "../../server/feature/feed/testdata"
	fixtures, err := filepath.Glob(filepath.Join(testdata, "fixture", "*.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures")
	}
	for _, f := range fixtures {
		name := strings.TrimSuffix(filepath.Base(f), ".fixture.json")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			var posts []map[string]string
			if err := json.Unmarshal(data, &posts); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(testdata, "source", name+".html"))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := simplify(raw, "https://"+strings.ReplaceAll(name, "-", "/"))
			if err != nil {
				t.Fatal(err)
			}
			// The values that a matcher can take from each node, by field, as
			// the client's extractPosts takes them.
			fields := map[string]map[string]bool{
				"title": {}, "body": {}, "link": {}, "thumbnail": {}, "pubDate": {},
			}
			var visit func(n *Node)
			visit = func(n *Node) {
				if n.Link != "" {
					fields["link"][n.Link] = true
				}
				switch n.Kind {
				case "img":
					fields["thumbnail"][n.Link] = true
				case "time":
					fields["pubDate"][n.Val] = true
				default:
					if n.isValue() && n.Val != "" {
						fields["title"][n.Val] = true
					}
					fields["body"][bodyMarkup(n)] = true
				}
				for _, c := range n.Children {
					visit(c)
				}
			}
			visit(doc.Root)
			for i, post := range posts {
				for field := range post {
					if fields[field] == nil {
						t.Errorf("post %d: unknown field %q", i, field)
					}
				}
				for _, field := range []string{"title", "body", "link", "thumbnail", "pubDate"} {
					if want, ok := post[field]; ok && !fields[field][want] {
						t.Errorf("post %d: %s %q is missing", i, field, want)
					}
				}
			}
		})
	}
}

// bodyMarkup is the body of a post at n, as the client's bodyMarkup in
// body_markup.dart writes it: one block per value, separated by a blank line.
func bodyMarkup(n *Node) string {
	var blocks []string
	var visit func(n *Node)
	visit = func(n *Node) {
		if !n.isValue() {
			for _, c := range n.Children {
				visit(c)
			}
			return
		}
		if n.Kind == "img" {
			blocks = append(blocks, "!["+n.Val+"]("+n.Link+")")
			return
		}
		text := n.Val
		if n.Link != "" {
			if text == "" {
				text = n.Link
			}
			text = "[" + text + "](" + n.Link + ")"
		}
		if text == "" {
			return
		}
		if headingTags[n.Kind] {
			text = strings.Repeat("#", int(n.Kind[1]-'0')) + " " + text
		}
		blocks = append(blocks, text)
	}
	visit(n)
	return strings.Join(blocks, "\n\n")
}
