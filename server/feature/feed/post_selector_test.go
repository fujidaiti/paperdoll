package feed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
)

// TestExtractPosts checks that ExtractPosts finds the posts in
// testdata/extract/<case>.posts.json on the page testdata/extract/<case>.html,
// with one small page for each case.
func TestExtractPosts(t *testing.T) {
	tests := []struct {
		name  string
		rules ExtractionRules
	}{
		{
			// Each post has every field that it has a value of, and leaves out the
			// others, such as the thumbnail of the second post. The body joins the
			// texts of its node.
			name: "fields",
			rules: ExtractionRules{Selectors: []PostSelector{{
				Box:       []Step{{Tag: "article", Cls: []string{"post"}}},
				Title:     []Step{{Tag: "h2"}},
				Link:      []Step{{Tag: "h2"}},
				Body:      []Step{{Tag: "div", Cls: []string{"summary"}}},
				Thumbnail: []Step{{Tag: "img"}},
				PubDate:   []Step{{Tag: "time"}},
			}}},
		},
		{
			// A node without the title or the link is not a post.
			name: "required-fields",
			rules: ExtractionRules{Selectors: []PostSelector{{
				Box:   []Step{{Tag: "article", Cls: []string{"post"}}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "h2"}},
			}}},
		},
		{
			// A post is left out when the host and the path of its link are in
			// ExcludeLinks. The query does not matter.
			name: "exclude",
			rules: ExtractionRules{
				Selectors: []PostSelector{{
					Box:   []Step{{Tag: "article", Cls: []string{"post"}}},
					Title: []Step{{Tag: "h2"}},
					Link:  []Step{{Tag: "h2"}},
				}},
				ExcludeLinks: []string{"example.com/newsletters"},
			},
		},
		{
			// No post is found in the subtree of a node that ExcludeNodes selects,
			// such as the menu in the first section.
			name: "exclude-nodes",
			rules: ExtractionRules{
				Selectors: []PostSelector{{
					Box:   []Step{{Tag: "article", Cls: []string{"post"}}},
					Title: []Step{{Tag: "h2"}},
					Link:  []Step{{Tag: "h2"}},
				}},
				ExcludeNodes: [][]Step{{{Tag: "section", Nth: new(0)}}},
			},
		},
		{
			// A post that contains an excluded node is still a post, but its fields
			// are not resolved in the excluded node: the body skips the paragraphs
			// of the section.
			name: "exclude-nodes-in-post",
			rules: ExtractionRules{
				Selectors: []PostSelector{{
					Box:   []Step{{Tag: "article", Cls: []string{"post"}}},
					Title: []Step{{Tag: "h2"}},
					Link:  []Step{{Tag: "h2"}},
					Body:  []Step{{Tag: "p"}},
				}},
				ExcludeNodes: [][]Step{{{Tag: "article", Nth: new(0)}, {Tag: "section", Nth: new(0)}}},
			},
		},
		{
			// A node that contains posts is a list of posts rather than a post, even
			// when it has a title and a link of its own.
			name: "list",
			rules: ExtractionRules{Selectors: []PostSelector{{
				Box:   []Step{{Tag: "div", Cls: []string{"card"}}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "h2"}},
			}}},
		},
		{
			// A node that Box selects inside a post and that has no title and link
			// does not make the post a list.
			name: "inner-node",
			rules: ExtractionRules{Selectors: []PostSelector{{
				Box:   []Step{{Tag: "div", Cls: []string{"card"}}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "h2"}},
			}}},
		},
		{
			// Each step of Box after the first selects children, so that the
			// article inside the related list is not a post.
			name: "children",
			rules: ExtractionRules{Selectors: []PostSelector{{
				Box:   []Step{{Tag: "section", Cls: []string{"latest"}}, {Tag: "article"}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "h2"}},
			}}},
		},
		{
			// A step with Nth selects only the node at that index.
			name: "nth",
			rules: ExtractionRules{Selectors: []PostSelector{{
				Box:   []Step{{Tag: "article", Cls: []string{"post"}}},
				Title: []Step{{Tag: "p", Nth: new(1)}},
				Link:  []Step{{Tag: "p", Nth: new(1)}},
			}}},
		},
		{
			// A field takes the first node that its path selects and that it
			// accepts: the body skips the paragraph that has only an image.
			name: "accepted-node",
			rules: ExtractionRules{Selectors: []PostSelector{{
				Box:       []Step{{Tag: "article", Cls: []string{"post"}}},
				Title:     []Step{{Tag: "h2"}},
				Link:      []Step{{Tag: "h2"}},
				Body:      []Step{{Tag: "p"}},
				Thumbnail: []Step{{Tag: "img"}},
			}}},
		},
		{
			// An empty path selects the post node itself.
			name: "post-node",
			rules: ExtractionRules{Selectors: []PostSelector{{
				Box:   []Step{{Tag: "a", Cls: []string{"card"}}},
				Title: []Step{{Tag: "h3"}},
				Link:  []Step{},
				Body:  []Step{{Tag: "p"}},
			}}},
		},
		{
			// The posts of all selectors are in document order.
			name: "selectors",
			rules: ExtractionRules{Selectors: []PostSelector{
				{
					Box:   []Step{{Tag: "article", Cls: []string{"post"}}},
					Title: []Step{{Tag: "h2"}},
					Link:  []Step{{Tag: "h2"}},
				},
				{
					Box:   []Step{{Tag: "div", Cls: []string{"featured"}}},
					Title: []Step{{Tag: "h3"}},
					Link:  []Step{{Tag: "h3"}},
				},
			}},
		},
		{
			// Posts with the same link are merged into the first one in document
			// order, which keeps its values of the fields that both have, even when
			// its selector comes later.
			name: "same-link-first-wins",
			rules: ExtractionRules{Selectors: []PostSelector{
				{
					Box:   []Step{{Tag: "div", Cls: []string{"card"}}},
					Title: []Step{{Tag: "h2"}},
					Link:  []Step{{Tag: "h2"}},
				},
				{
					Box:   []Step{{Tag: "div", Cls: []string{"teaser"}}},
					Title: []Step{{Tag: "h3"}},
					Link:  []Step{{Tag: "h3"}},
				},
			}},
		},
		{
			// The first post of a link takes the fields that it does not have from
			// the later posts with the same link.
			name: "same-link-missing-fields",
			rules: ExtractionRules{Selectors: []PostSelector{
				{
					Box:       []Step{{Tag: "div", Cls: []string{"teaser"}}},
					Title:     []Step{{Tag: "h3"}},
					Link:      []Step{{Tag: "h3"}},
					Thumbnail: []Step{{Tag: "img"}},
				},
				{
					Box:   []Step{{Tag: "div", Cls: []string{"card"}}},
					Title: []Step{{Tag: "h3"}},
					Link:  []Step{{Tag: "h3"}},
					Body:  []Step{{Tag: "p"}},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata/extract", tt.name+".html"))
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
			var want []ExtractedPost
			readJSON(t, filepath.Join("testdata/extract", tt.name+".posts.json"), &want)
			if diff := cmp.Diff(want, ExtractPosts(doc, tt.rules)); diff != "" {
				t.Errorf("posts differ (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPostSelectorJSON(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		m := PostSelector{
			Box:     []Step{{Tag: "a", Cls: []string{"card"}}},
			Title:   []Step{{Tag: "h3"}},
			Link:    []Step{},
			PubDate: []Step{{Tag: "time", Nth: new(0)}},
		}

		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"box":[{"tag":"a","cls":["card"]}],` +
			`"title":[{"tag":"h3"}],"link":[],"pubDate":[{"tag":"time","nth":0}]}`
		if string(data) != want {
			t.Errorf("JSON is %s, want %s", data, want)
		}

		// The empty path of the link stays empty instead of becoming nil, which
		// would mean that the selector has no link.
		var got PostSelector
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(m, got); diff != "" {
			t.Errorf("selector differs (-want +got):\n%s", diff)
		}
	})

	errors := []struct{ name, json string }{
		{"unknown field", `{"box":[{"tag":"article"}],"title":[],"link":[],"author":[]}`},
		{"no title", `{"box":[{"tag":"article"}],"link":[]}`},
		{"no link", `{"box":[{"tag":"article"}],"title":[]}`},
	}
	for _, tt := range errors {
		t.Run(tt.name, func(t *testing.T) {
			var m PostSelector
			if err := json.Unmarshal([]byte(tt.json), &m); err == nil {
				t.Errorf("no error for %s", tt.json)
			}
		})
	}
}

func TestExtractionRulesJSON(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		rules := ExtractionRules{
			Selectors: []PostSelector{{
				Box:   []Step{{Tag: "article"}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "a"}},
			}},
			ExcludeLinks: []string{"example.com/newsletters"},
			ExcludeNodes: [][]Step{{{Tag: "nav", Nth: new(0)}}},
		}

		data, err := json.Marshal(rules)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"selectors":[{"box":[{"tag":"article"}],"title":[{"tag":"h2"}],"link":[{"tag":"a"}]}],` +
			`"excludeLinks":["example.com/newsletters"],` +
			`"excludeNodes":[[{"tag":"nav","nth":0}]]}`
		if string(data) != want {
			t.Errorf("JSON is %s, want %s", data, want)
		}

		var got ExtractionRules
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(rules, got); diff != "" {
			t.Errorf("rules differ (-want +got):\n%s", diff)
		}
	})

	// Rules without selectors and exclusions keep the selectors as an array, and
	// leave out the empty lists.
	t.Run("empty", func(t *testing.T) {
		data, err := json.Marshal(ExtractionRules{Selectors: []PostSelector{}})
		if err != nil {
			t.Fatal(err)
		}
		if want := `{"selectors":[]}`; string(data) != want {
			t.Errorf("JSON is %s, want %s", data, want)
		}
	})
}

// TestBodyText checks bodyText against the nodes of testdata/body-text.html,
// one node for each case, with the name of the case as its class.
func TestBodyText(t *testing.T) {
	raw, err := os.ReadFile("testdata/body-text.html")
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
	nodes := slices.Collect(pageBody(doc).ChildNodes())

	tests := []struct{ name, want string }{
		{"value", "The summary of a post."},
		// The texts are joined in document order, at any depth.
		{"values", "The first line. The second line."},
		{"nested", "The first line. The second line. The third line."},
		// No space is put before closing punctuation or after an opening bracket.
		{"punctuation", "Tools, Kiro (beta)"},
		// A link keeps only its text.
		{"link", "The summary of a post. Read more"},
		// Images and times are left out, even with an alt text or a timestamp.
		{"image", "The summary of a post."},
		{"time", "The summary of a post."},
		{"only-image", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := slices.IndexFunc(nodes, func(n *html.Node) bool {
				return slices.Contains(classesOf(n), tt.name)
			})
			if i < 0 {
				t.Fatalf("no node with the class %q", tt.name)
			}
			if got := bodyText(nodes[i]); got != tt.want {
				t.Errorf("bodyText is %q, want %q", got, tt.want)
			}
		})
	}
}
