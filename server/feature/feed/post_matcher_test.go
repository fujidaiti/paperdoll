package feed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var postFields = []PostField{PostTitle, PostBody, PostLink, PostThumbnail, PostPubDate}

// fixturePage is a test page in testdata/source with its fixture: the posts
// that a correct matcher finds on it (see testdata/README.md).
type fixturePage struct {
	name  string
	tree  *SemanticTree
	posts []ExtractedPost
}

func loadFixturePages(t *testing.T) []fixturePage {
	t.Helper()
	files, err := filepath.Glob("testdata/fixture/*.fixture.json")
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
		raw, err := os.ReadFile(filepath.Join("testdata/source", name+".html"))
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
// each test page keeps every value of the posts in its fixture, so that a
// matcher can find them in the tree.
func TestBuildSemanticTreeKeepsFixtureValues(t *testing.T) {
	for _, page := range loadFixturePages(t) {
		t.Run(page.name, func(t *testing.T) {
			// The values that a matcher can take from the nodes, by field. A
			// field is mapped only to nodes that it accepts, as the client's
			// PostField.accepts tells.
			values := map[PostField]map[string]bool{}
			for _, f := range postFields {
				values[f] = map[string]bool{}
			}
			add := func(f PostField, n *SemanticTreeNode) {
				if v, ok := f.valueOf(n); ok {
					values[f][v] = true
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
					visit(c)
				}
			}
			visit(page.tree.Root)
			for i, post := range page.posts {
				for f := range post {
					if values[f] == nil {
						t.Errorf("post %d: unknown field %q", i, f)
					}
				}
				for _, f := range postFields {
					if want, ok := post[f]; ok && !values[f][want] {
						t.Errorf("post %d: %s %q is missing", i, f, want)
					}
				}
			}
		})
	}
}

// TestExtractPostsFindsFixturePosts checks that the matchers of each test page
// (testdata/matcher) extract exactly the posts in its fixture.
func TestExtractPostsFindsFixturePosts(t *testing.T) {
	for _, page := range loadFixturePages(t) {
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
