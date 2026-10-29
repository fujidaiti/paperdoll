package feed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var postFields = []PostField{PostTitle, PostBody, PostLink, PostThumbnail, PostPubDate}

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
					// The nodes that hold each value that a matcher can take from
					// them, by field. A field is mapped only to nodes that it
					// accepts, as the client's PostField.accepts tells.
					holders := make(map[PostField]map[string][]*SemanticTreeNode)
					for _, f := range postFields {
						holders[f] = map[string][]*SemanticTreeNode{}
					}
					parent := map[*SemanticTreeNode]*SemanticTreeNode{}
					add := func(f PostField, n *SemanticTreeNode) {
						if v, ok := f.valueOf(n); ok {
							holders[f][v] = append(holders[f][v], n)
						}
					}
					var visit func(n *SemanticTreeNode)
					visit = func(n *SemanticTreeNode) {
						switch n.Tag {
						case "img":
							add(PostThumbnail, n)
						case "time":
							add(PostPubDate, n)
						default:
							if n.isValue() {
								add(PostTitle, n)
							}
							add(PostBody, n)
							add(PostLink, n)
						}
						for _, c := range n.Children {
							parent[c] = n
							visit(c)
						}
					}
					visit(page.tree.Root)

					// The number of values of each post that the subtree of each node
					// holds, and the number of values of each post.
					counts := make([]map[*SemanticTreeNode]int, len(page.posts))
					fields := make([]int, len(page.posts))
					for i, post := range page.posts {
						for f := range post {
							if holders[f] == nil {
								t.Errorf("post %d: unknown field %q", i, f)
							}
						}
						counts[i] = map[*SemanticTreeNode]int{}
						for _, f := range postFields {
							want, ok := post[f]
							if !ok {
								continue
							}
							if len(holders[f][want]) == 0 {
								t.Errorf("post %d: %s %q is missing", i, f, want)
							}
							// Each node is counted once per field, however many of
							// its descendants hold the value.
							marked := map[*SemanticTreeNode]bool{}
							for _, n := range holders[f][want] {
								for ; n != nil && !marked[n]; n = parent[n] {
									marked[n] = true
								}
							}
							for n := range marked {
								counts[i][n]++
							}
							fields[i]++
						}
					}

					// Each node that holds all values of a post, with children
					// before their parent, is the node of that post unless it is or
					// contains the node of another post. A post that gets no node
					// overlaps another post.
					found := make([]bool, len(page.posts))
					taken := map[*SemanticTreeNode]bool{}
					var claim func(n *SemanticTreeNode)
					claim = func(n *SemanticTreeNode) {
						for _, c := range n.Children {
							claim(c)
						}
						if taken[n] {
							return
						}
						for i := range page.posts {
							if !found[i] && counts[i][n] == fields[i] {
								found[i] = true
								for a := n; a != nil && !taken[a]; a = parent[a] {
									taken[a] = true
								}
								return
							}
						}
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
