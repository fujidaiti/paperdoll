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
	doc   *PageDocument
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
		doc, err := SimplifyPage(raw, "https://"+strings.ReplaceAll(name, "-", "/"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		pages = append(pages, fixturePage{name, doc, posts})
	}
	return pages
}

// TestSimplifyPageKeepsFixtureValues checks that simplifying each test page
// keeps every value of the posts in its fixture, so that a matcher can find
// them in the tree.
func TestSimplifyPageKeepsFixtureValues(t *testing.T) {
	for _, page := range loadFixturePages(t) {
		t.Run(page.name, func(t *testing.T) {
			// The values that a matcher can take from the nodes, by field. A
			// field is mapped only to nodes that it accepts, as the client's
			// PostField.accepts tells.
			values := map[PostField]map[string]bool{}
			for _, f := range postFields {
				values[f] = map[string]bool{}
			}
			add := func(f PostField, n *PageNode) {
				if v, ok := f.valueOf(n); ok {
					values[f][v] = true
				}
			}
			var visit func(n *PageNode)
			visit = func(n *PageNode) {
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
			visit(page.doc.Root)
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

// TestExtractPostsFindsFixturePosts checks that the matchers built by hand on
// each test page (testdata/matcher) extract the posts in its fixture.
func TestExtractPostsFindsFixturePosts(t *testing.T) {
	for _, page := range loadFixturePages(t) {
		t.Run(page.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata/matcher", page.name+".matcher.json"))
			if err != nil {
				t.Fatal(err)
			}
			var file struct {
				Matchers []PostMatcher `json:"matchers"`
			}
			// A page that has no usable matcher has an empty file or list.
			if s := strings.TrimSpace(string(data)); s == "" || s == "[]" {
				t.Skip("no matchers")
			}
			if err := json.Unmarshal(data, &file); err != nil {
				t.Fatal(err)
			}
			got := ExtractPosts(page.doc, file.Matchers)
			for i := range max(len(got), len(page.posts)) {
				switch {
				case i >= len(got):
					t.Errorf("post %d is missing: %v", i, page.posts[i])
				case i >= len(page.posts):
					t.Errorf("post %d is extra: %v", i, got[i])
				default:
					for _, f := range postFields {
						if got[i][f] != page.posts[i][f] {
							t.Errorf("post %d: %s is %q, want %q", i, f, got[i][f], page.posts[i][f])
						}
					}
				}
			}
		})
	}
}
