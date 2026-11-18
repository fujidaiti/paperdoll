package feed

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
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

// generateWebpageRules returns the rules that the marks of the fixture of the
// test page name generate on its first version.
func generateWebpageRules(t *testing.T, name string) ExtractionRules {
	t.Helper()
	_, index, err := parseEditorPage(simplifyWebpage(t, name, ""))
	if err != nil {
		t.Fatal(err)
	}
	var marks Marks
	readJSON(t, filepath.Join("testdata/webpage", name, "marks.fixture.json"), &marks)
	posts, excluded, excludedNodes, err := index.resolveMarks(marks)
	if err != nil {
		t.Fatal(err)
	}
	return index.generateRules(posts, excluded, excludedNodes)
}

// TestGenerateRulesGolden checks the rules that the marks of each test page
// generate against its golden file.
func TestGenerateRulesGolden(t *testing.T) {
	for _, name := range webpages(t) {
		t.Run(name, func(t *testing.T) {
			got, err := json.Marshal(generateWebpageRules(t, name))
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, filepath.Join("testdata/webpage", name, "rules.golden.json"), got)
		})
	}
}

// TestGeneratedRulesFindFixturePosts checks that the rules generated from the
// marks of each test page find exactly the posts of its fixture, by node ID and
// value, on the page that they are built on and on its later version. A found
// post has the fields of the first selector that finds its node, so the posts
// that the rules extract, which take their fields from every selector and merge
// by link, are checked against their own fixture.
func TestGeneratedRulesFindFixturePosts(t *testing.T) {
	for _, name := range webpages(t) {
		rules := generateWebpageRules(t, name)
		for _, version := range []string{"", "later"} {
			t.Run(filepath.Join(name, version), func(t *testing.T) {
				doc, index, err := parseEditorPage(simplifyWebpage(t, name, version))
				if err != nil {
					t.Fatal(err)
				}
				dir := filepath.Join("testdata/webpage", name, version)
				state := index.editorState(doc, rules)
				var want []FoundPost
				readJSON(t, filepath.Join(dir, "posts.fixture.json"), &want)
				if d := cmp.Diff(want, state.FoundPosts); d != "" {
					t.Errorf("found posts differ (-want +got):\n%s", d)
				}
				var preview []ExtractedPost
				readJSON(t, filepath.Join(dir, "preview.fixture.json"), &preview)
				if d := cmp.Diff(preview, state.Preview); d != "" {
					t.Errorf("extracted posts differ (-want +got):\n%s", d)
				}
			})
		}
	}
}

func TestPostClasses(t *testing.T) {
	for _, tt := range []struct{ cls, want []string }{
		{[]string{"post", "other"}, []string{"post"}},
		{[]string{"type-post", "other"}, []string{"type-post"}},
		{[]string{"post-card", "other"}, []string{"post-card"}},
		{[]string{"masonry-card", "other"}, []string{"masonry-card"}},
		{[]string{"card__inner", "other"}, []string{"card__inner"}},
		{[]string{"storyCard__linked", "other"}, []string{"storyCard__linked"}},
		{[]string{"IndexCardStyled", "other"}, []string{"IndexCardStyled"}},
		{[]string{"AdPost", "other"}, []string{"AdPost"}},
		{[]string{"adpost", "other"}, []string{"adpost", "other"}},
		{[]string{"compost", "other"}, []string{"compost", "other"}},
		{[]string{"greencard", "other"}, []string{"greencard", "other"}},
		{[]string{"postcard", "other"}, []string{"postcard", "other"}},
		{[]string{"cardinal", "other"}, []string{"cardinal", "other"}},
		{[]string{"post-9818", "other"}, []string{"other"}},
		{[]string{"category-security", "other"}, []string{"other"}},
		{[]string{"tag-whatsapp", "other"}, []string{"other"}},
		{[]string{"has-post-thumbnail", "other"}, []string{"other"}},
		{[]string{"post", "type-post", "hentry", "post-9818", "category-security"}, []string{"post", "type-post"}},
		{[]string{"post-9818", "category-security"}, []string{}},
	} {
		if got := postClasses(tt.cls); !slices.Equal(got, tt.want) {
			t.Errorf("postClasses(%q) = %q, want %q", tt.cls, got, tt.want)
		}
	}
}
