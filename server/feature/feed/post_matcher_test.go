package feed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// postFields are the fields of a post, with the body last, since building the
// body of a node is slow.
var postFields = []PostField{PostTitle, PostLink, PostThumbnail, PostPubDate, PostBody}

// fixturePage is a test page with its fixture: the posts that a correct
// matcher finds on it (see testdata/README.md).
type fixturePage struct {
	name  string
	tree  *SemanticTree
	posts []ExtractedPost
}

// fixtureSets are the sets of test pages with their fixtures: the pages that
// the matchers are built on, and the later versions of them.
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
		tree, err := BuildSemanticTree(raw, "https://"+strings.ReplaceAll(name, "-", "/"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		pages = append(pages, fixturePage{name, tree, posts})
	}
	return pages
}

// TestBuildSemanticTreeKeepsFixtureValues checks that the semantic tree of
// each test page keeps every value of the posts in its fixture, and that each
// post has its own subtree: a smallest node that holds all values of the post
// and neither is nor contains nor is inside such a node of another post, since
// posts do not nest in a page. A matcher can only find posts as such nodes.
// The later versions of the pages in testdata/robustness are checked too.
func TestBuildSemanticTreeKeepsFixtureValues(t *testing.T) {
	for _, set := range fixtureSets {
		t.Run(set.name, func(t *testing.T) {
			for _, page := range loadFixturePages(t, set.source, set.fixture) {
				t.Run(page.name, func(t *testing.T) {
					for i, post := range page.posts {
						for f := range post {
							if !slices.Contains(postFields, f) {
								t.Errorf("post %d: unknown field %q", i, f)
							}
						}
					}

					// holds reports whether the subtree of n has a node that f
					// accepts and whose value of f is want.
					var holds func(n *SemanticTreeNode, f PostField, want string) bool
					holds = func(n *SemanticTreeNode, f PostField, want string) bool {
						if v, ok := f.valueOf(n); ok && v == want && f.accepts(n) {
							return true
						}
						return slices.ContainsFunc(
							n.Children,
							func(c *SemanticTreeNode) bool { return holds(c, f, want) },
						)
					}
					// claim makes each node that holds all values of a post, with
					// children before their parent, the node of that post unless it
					// is or contains the node of another post, and reports whether n
					// is or contains such a node. A post that gets no node overlaps
					// another post.
					found := make([]bool, len(page.posts))
					var claim func(n *SemanticTreeNode) bool
					claim = func(n *SemanticTreeNode) bool {
						taken := false
						for _, c := range n.Children {
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
					claim(page.tree.Root)
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

// TestExtractPostsFindsFixturePosts checks that the matchers of each test page
// (testdata/matcher) extract exactly the posts in its fixture. The matchers
// are built on testdata/source, and must still extract exactly the posts of
// the later version of each page in testdata/robustness.
func TestExtractPostsFindsFixturePosts(t *testing.T) {
	for _, set := range fixtureSets {
		t.Run(set.name, func(t *testing.T) {
			for _, page := range loadFixturePages(t, set.source, set.fixture) {
				t.Run(page.name, func(t *testing.T) {
					var matchers []PostMatcher
					readJSON(t, filepath.Join("testdata/matcher", page.name+".json"), &matchers)
					// The posts are compared by link, since a fixture does not always
					// list them in the order of the page.
					want := map[string]ExtractedPost{}
					for _, p := range page.posts {
						want[p[PostLink]] = p
					}
					got := map[string]ExtractedPost{}
					for _, p := range ExtractPosts(page.tree, matchers) {
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
