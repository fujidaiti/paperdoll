package feed

import (
	"cmp"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
// post has its own subtree: every smallest node that holds all values of a
// post must neither be nor contain nor be inside such a node of another post,
// since posts do not nest in a page. A matcher can only find posts as such
// nodes. The later versions of the pages in testdata/robustness are checked
// too.
func TestBuildSemanticTreeKeepsFixtureValues(t *testing.T) {
	for _, set := range []struct{ name, source, fixture string }{
		{"source", "testdata/source", "testdata/fixture"},
		{"robustness", "testdata/robustness", "testdata/fixture/robustness"},
	} {
		t.Run(set.name, func(t *testing.T) {
			for _, page := range loadFixturePages(t, set.source, set.fixture) {
				t.Run(page.name, func(t *testing.T) {
					// The nodes that hold each value that a matcher can take from
					// them, by field. A field is mapped only to nodes that it
					// accepts, as the client's PostField.accepts tells.
					holders := map[PostField]map[string][]*SemanticTreeNode{}
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
						switch n.Kind {
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

					// The smallest nodes that hold all values of each post: the nodes
					// whose subtree holds every value, without a child that does.
					smallest := make([][]*SemanticTreeNode, len(page.posts))
					for i, post := range page.posts {
						for f := range post {
							if holders[f] == nil {
								t.Errorf("post %d: unknown field %q", i, f)
							}
						}
						var count map[*SemanticTreeNode]int
						fields := 0
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
							if count == nil {
								count = map[*SemanticTreeNode]int{}
							}
							for n := range marked {
								count[n]++
							}
							fields++
						}
						for n, c := range count {
							if c != fields {
								continue
							}
							if !slices.ContainsFunc(
								n.Children,
								func(c *SemanticTreeNode) bool { return count[c] == fields },
							) {
								smallest[i] = append(smallest[i], n)
							}
						}
					}

					// The posts whose smallest nodes are at each node.
					owners := map[*SemanticTreeNode][]int{}
					for i, nodes := range smallest {
						for _, n := range nodes {
							owners[n] = append(owners[n], i)
						}
					}
					// Walking up from each smallest node finds every smallest node
					// that is it or contains it, so the pairs where one contains the
					// other are found from the inner one.
					overlaps := map[[2]int]bool{}
					for i, nodes := range smallest {
						for _, n := range nodes {
							for a := n; a != nil; a = parent[a] {
								for _, j := range owners[a] {
									if j != i {
										overlaps[[2]int{min(i, j), max(i, j)}] = true
									}
								}
							}
						}
					}
					for _, pair := range slices.SortedFunc(maps.Keys(overlaps), func(a, b [2]int) int {
						return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1]))
					}) {
						i, j := pair[0], pair[1]
						t.Errorf(
							"posts %d (%s) and %d (%s) overlap",
							i,
							page.posts[i][PostLink],
							j,
							page.posts[j][PostLink],
						)
					}
				})
			}
		})
	}
}

// TestExtractPostsFindsFixturePosts checks that the matchers of each test page
// (testdata/matcher) extract exactly the posts in its fixture.
func TestExtractPostsFindsFixturePosts(t *testing.T) {
	checkExtractedPosts(t, loadFixturePages(t, "testdata/source", "testdata/fixture"))
}

// TestExtractPostsFindsRobustnessFixturePosts checks that the matchers of each
// test page, which are built on testdata/source, still extract exactly the
// posts of the later version of the page in testdata/robustness.
func TestExtractPostsFindsRobustnessFixturePosts(t *testing.T) {
	checkExtractedPosts(t, loadFixturePages(t, "testdata/robustness", "testdata/fixture/robustness"))
}

// checkExtractedPosts checks that the matchers in testdata/matcher extract
// exactly the fixture posts from each of pages.
func checkExtractedPosts(t *testing.T, pages []fixturePage) {
	t.Helper()
	for _, page := range pages {
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
}
