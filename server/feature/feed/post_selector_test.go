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

// postFields are the fields of a post, with the body last, since building the
// body of a node is slow.
var postFields = []PostField{PostTitle, PostLink, PostThumbnail, PostPubDate, PostBody}

// fixturePage is a test page with its fixture: the posts that a correct
// selector finds on it (see testdata/README.md).
type fixturePage struct {
	name string
	// doc is the simplified page parsed by html.Parse.
	doc   *html.Node
	posts []ExtractedPost
}

// fixtureSets are the sets of test pages with their fixtures: the pages that
// the selectors are built on, and the later versions of them.
var fixtureSets = []struct{ name, source, fixture string }{
	{"source", "testdata/source", "testdata/fixture"},
	{"robustness", "testdata/robustness", "testdata/fixture/robustness"},
}

// loadFixturePages loads the pages in sourceDir that have a fixture in
// fixtureDir.
func loadFixturePages(t *testing.T, sourceDir, fixtureDir string) []fixturePage {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(fixtureDir, "*.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no fixtures")
	}
	var pages []fixturePage
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".fixture.json")
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var posts []ExtractedPost
		if err := json.Unmarshal(data, &posts); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		raw, err := os.ReadFile(filepath.Join(sourceDir, name+".html"))
		if err != nil {
			t.Fatal(err)
		}
		page, err := SimplifyPage(raw, "https://"+strings.ReplaceAll(name, "-", "/"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		doc, err := html.Parse(strings.NewReader(page))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		pages = append(pages, fixturePage{name, doc, posts})
	}
	return pages
}

// TestSimplifyPageKeepsFixtureValues checks that the simplified page of each
// test page keeps every value of the posts in its fixture, and that each
// post has its own subtree: a smallest node that holds all values of the post
// and neither is nor contains nor is inside such a node of another post, since
// posts do not nest in a page. A selector can only find posts as such nodes.
// The later versions of the pages in testdata/robustness are checked too.
func TestSimplifyPageKeepsFixtureValues(t *testing.T) {
	// holds reports whether the subtree of n has a node that f
	// accepts and whose value of f is want.
	var holds func(n *html.Node, f PostField, want string) bool
	holds = func(n *html.Node, f PostField, want string) bool {
		if v, ok := f.valueOf(n); ok && v == want && f.accepts(n) {
			return true
		}
		for c := range childrenOf(n) {
			if holds(c, f, want) {
				return true
			}
		}
		return false
	}

	for _, set := range fixtureSets {
		t.Run(set.name, func(t *testing.T) {
			for _, page := range loadFixturePages(t, set.source, set.fixture) {
				t.Run(page.name, func(t *testing.T) {
					// claim makes each node that holds all values of a post, with
					// children before their parent, the node of that post unless it
					// is or contains the node of another post, and reports whether n
					// is or contains such a node. A post that gets no node overlaps
					// another post.
					found := make([]bool, len(page.posts))
					var claim func(n *html.Node) bool
					claim = func(n *html.Node) bool {
						taken := false
						for c := range childrenOf(n) {
							taken = claim(c) || taken
						}
						if taken {
							return true
						}
					posts:
						for i, post := range page.posts {
							if found[i] {
								continue
							}
							for _, f := range postFields {
								if want, ok := post[f]; ok && !holds(n, f, want) {
									continue posts
								}
							}
							found[i] = true
							return true
						}
						return false
					}

					claim(pageBody(page.doc))
					for i, ok := range found {
						if !ok {
							t.Errorf("post %d (%s) has no node of its own", i, page.posts[i][PostLink])
						}
					}
				})
			}
		})
	}
}

// TestExtractPostsFindsFixturePosts checks that the selectors of each test page
// (testdata/selector) extract exactly the posts in its fixture. The selectors
// are built on testdata/source, and must still extract exactly the posts of
// the later version of each page in testdata/robustness.
func TestExtractPostsFindsFixturePosts(t *testing.T) {
	for _, set := range fixtureSets {
		t.Run(set.name, func(t *testing.T) {
			for _, page := range loadFixturePages(t, set.source, set.fixture) {
				t.Run(page.name, func(t *testing.T) {
					var selectors []PostSelector
					readJSON(t, filepath.Join("testdata/selector", page.name+".json"), &selectors)
					// The posts are compared by link, since a fixture does not always
					// list them in the order of the page.
					want := map[string]ExtractedPost{}
					for _, p := range page.posts {
						want[p[PostLink]] = p
					}
					got := map[string]ExtractedPost{}
					for _, p := range ExtractPosts(page.doc, selectors) {
						got[p[PostLink]] = p
						if w, ok := want[p[PostLink]]; !ok {
							t.Errorf("extra post: %v", p)
						} else if !equalPosts(p, w) {
							t.Errorf("post is %v, want %v", p, w)
						}
					}
					for _, p := range page.posts {
						if _, ok := got[p[PostLink]]; !ok {
							t.Errorf("missing post: %v", p)
						}
					}
				})
			}
		})
	}
}

// TestExtractPosts checks that ExtractPosts finds the posts in
// testdata/extract/<case>.posts.json on the page testdata/extract/<case>.html,
// with one small page for each case.
func TestExtractPosts(t *testing.T) {
	tests := []struct {
		name      string
		selectors []PostSelector
	}{
		{
			// Each post has every field that it has a value of, and leaves out the
			// others, such as the thumbnail of the second post. The body joins the
			// texts of its node.
			name: "fields",
			selectors: []PostSelector{{
				Post:      []Step{{Tag: "article", Cls: []string{"post"}}},
				Title:     []Step{{Tag: "h2"}},
				Link:      []Step{{Tag: "h2"}},
				Body:      []Step{{Tag: "div", Cls: []string{"summary"}}},
				Thumbnail: []Step{{Tag: "img"}},
				PubDate:   []Step{{Tag: "time"}},
			}},
		},
		{
			// A node without the title or the link is not a post.
			name: "required-fields",
			selectors: []PostSelector{{
				Post:  []Step{{Tag: "article", Cls: []string{"post"}}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "h2"}},
			}},
		},
		{
			// A post is left out when the host and the path of its link match a
			// pattern in Exclude. The query does not matter.
			name: "exclude",
			selectors: []PostSelector{{
				Post:    []Step{{Tag: "article", Cls: []string{"post"}}},
				Title:   []Step{{Tag: "h2"}},
				Link:    []Step{{Tag: "h2"}},
				Exclude: []string{"example.com/newsletters"},
			}},
		},
		{
			// A node that contains posts is a list of posts rather than a post, even
			// when it has a title and a link of its own.
			name: "list",
			selectors: []PostSelector{{
				Post:  []Step{{Tag: "div", Cls: []string{"card"}}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "h2"}},
			}},
		},
		{
			// A node that Post selects inside a post and that has no title and link
			// does not make the post a list.
			name: "inner-node",
			selectors: []PostSelector{{
				Post:  []Step{{Tag: "div", Cls: []string{"card"}}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "h2"}},
			}},
		},
		{
			// Each step of Post after the first selects children, so that the
			// article inside the related list is not a post.
			name: "children",
			selectors: []PostSelector{{
				Post:  []Step{{Tag: "section", Cls: []string{"latest"}}, {Tag: "article"}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "h2"}},
			}},
		},
		{
			// A step with Nth selects only the node at that index.
			name: "nth",
			selectors: []PostSelector{{
				Post:  []Step{{Tag: "article", Cls: []string{"post"}}},
				Title: []Step{{Tag: "p", Nth: new(1)}},
				Link:  []Step{{Tag: "p", Nth: new(1)}},
			}},
		},
		{
			// A field takes the first node that its path selects and that it
			// accepts: the body skips the paragraph that has only an image.
			name: "accepted-node",
			selectors: []PostSelector{{
				Post:      []Step{{Tag: "article", Cls: []string{"post"}}},
				Title:     []Step{{Tag: "h2"}},
				Link:      []Step{{Tag: "h2"}},
				Body:      []Step{{Tag: "p"}},
				Thumbnail: []Step{{Tag: "img"}},
			}},
		},
		{
			// An empty path selects the post node itself.
			name: "post-node",
			selectors: []PostSelector{{
				Post:  []Step{{Tag: "a", Cls: []string{"card"}}},
				Title: []Step{{Tag: "h3"}},
				Link:  []Step{},
				Body:  []Step{{Tag: "p"}},
			}},
		},
		{
			// The posts of all selectors are in document order.
			name: "selectors",
			selectors: []PostSelector{
				{
					Post:  []Step{{Tag: "article", Cls: []string{"post"}}},
					Title: []Step{{Tag: "h2"}},
					Link:  []Step{{Tag: "h2"}},
				},
				{
					Post:  []Step{{Tag: "div", Cls: []string{"featured"}}},
					Title: []Step{{Tag: "h3"}},
					Link:  []Step{{Tag: "h3"}},
				},
			},
		},
		{
			// Posts with the same link are merged into the first one in document
			// order, which keeps its values of the fields that both have, even when
			// its selector comes later.
			name: "same-link-first-wins",
			selectors: []PostSelector{
				{
					Post:  []Step{{Tag: "div", Cls: []string{"card"}}},
					Title: []Step{{Tag: "h2"}},
					Link:  []Step{{Tag: "h2"}},
				},
				{
					Post:  []Step{{Tag: "div", Cls: []string{"teaser"}}},
					Title: []Step{{Tag: "h3"}},
					Link:  []Step{{Tag: "h3"}},
				},
			},
		},
		{
			// The first post of a link takes the fields that it does not have from
			// the later posts with the same link.
			name: "same-link-missing-fields",
			selectors: []PostSelector{
				{
					Post:      []Step{{Tag: "div", Cls: []string{"teaser"}}},
					Title:     []Step{{Tag: "h3"}},
					Link:      []Step{{Tag: "h3"}},
					Thumbnail: []Step{{Tag: "img"}},
				},
				{
					Post:  []Step{{Tag: "div", Cls: []string{"card"}}},
					Title: []Step{{Tag: "h3"}},
					Link:  []Step{{Tag: "h3"}},
					Body:  []Step{{Tag: "p"}},
				},
			},
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
			if diff := cmp.Diff(want, ExtractPosts(doc, tt.selectors)); diff != "" {
				t.Errorf("posts differ (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPostSelectorJSON(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		m := PostSelector{
			Post:    []Step{{Tag: "a", Cls: []string{"card"}}},
			Title:   []Step{{Tag: "h3"}},
			Link:    []Step{},
			PubDate: []Step{{Tag: "time", Nth: new(0)}},
			Exclude: []string{"example.com/newsletters"},
		}

		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"post":[{"tag":"a","cls":["card"]}],` +
			`"fields":{"link":[],"pubDate":[{"tag":"time","nth":0}],"title":[{"tag":"h3"}]},` +
			`"exclude":["example.com/newsletters"]}`
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
		{"unknown field", `{"post":[{"tag":"article"}],"fields":{"title":[],"link":[],"author":[]}}`},
		{"no title", `{"post":[{"tag":"article"}],"fields":{"link":[]}}`},
		{"no link", `{"post":[{"tag":"article"}],"fields":{"title":[]}}`},
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

func TestGlobRegexp(t *testing.T) {
	tests := []struct {
		pattern, hostPath string
		want              bool
	}{
		{"example.com/blog", "example.com/blog", true},
		{"example.com/blog", "example.com/blog/first", false},
		{"example.com", "exampleXcom", false},
		{"example.com/*", "example.com/blog", true},
		{"example.com/*", "example.com/blog/first", false},
		{"example.com/**", "example.com/blog/first", true},
		{"example.com/blog-?", "example.com/blog-1", true},
		{"example.com/blog?", "example.com/blog/", false},
	}
	for _, tt := range tests {
		if got := globRegexp(tt.pattern).MatchString(tt.hostPath); got != tt.want {
			t.Errorf("globRegexp(%q) matches %q: %v, want %v", tt.pattern, tt.hostPath, got, tt.want)
		}
	}
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
