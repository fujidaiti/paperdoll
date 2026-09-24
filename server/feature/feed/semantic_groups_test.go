package feed

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// groupAcceptance is what one page is required to produce when its posts are
// read out of the semantic tree. Groups is a ceiling, because it is the number
// of rows the user has to read; Found and Extra are a floor and a ceiling on
// the links offered.
type groupAcceptance struct {
	// Groups is how many groups the page may offer.
	Groups int
	// Found is how many fixture posts the offered links have to reach, and
	// Extra is how many offered links may point at something else.
	Found, Extra int
}

// TestSemanticGroups reads the posts of every saved page out of the semantic
// tree and checks them against the hand written fixtures.
//
// Run it with PAPERDOLL_DUMP=1 to print the measured table instead of only
// checking it, which is how the values below are written.
func TestSemanticGroups(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no fixture found in testdata")
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, strings.TrimSuffix(filepath.Base(f), ".fixture.json"))
	}
	sort.Strings(names)

	dump := os.Getenv("PAPERDOLL_DUMP") != ""
	var groups, found, extra, posts int
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			want := readFixture(t, filepath.Join("testdata", name+".fixture.json"))
			if len(want.Posts) == 0 {
				t.Skip("the fixture holds no post")
			}
			u, err := url.Parse(want.URL)
			if err != nil {
				t.Fatal(err)
			}
			f, err := os.Open(filepath.Join("testdata", name+".html"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()
			root, err := BuildSemanticTree(f, *u)
			if err != nil {
				t.Fatal(err)
			}

			wanted := map[string]bool{}
			for _, p := range want.Posts {
				wanted[postKey(p.URL)] = true
			}
			offered := map[string]bool{}
			selected := SelectPostGroups(root, 0.8)
			for _, g := range selected {
				for _, l := range g.Links {
					if l != "" {
						offered[postKey(l)] = true
					}
				}
			}
			hit, miss := 0, 0
			for k := range offered {
				if wanted[k] {
					hit++
				} else {
					miss++
				}
			}

			groups += len(selected)
			found += hit
			extra += miss
			posts += len(wanted)

			if dump {
				fmt.Printf("\t%q: {%d, %d, %d},\n", name, len(selected), hit, miss)
				return
			}
			a, ok := groupAccepted[name]
			if !ok {
				t.Fatal("no acceptance values for this page. Run the test " +
					"with PAPERDOLL_DUMP=1 and add the line it prints to " +
					"the table at the end of semantic_groups_test.go.")
			}
			if len(selected) > a.Groups {
				t.Errorf("groups: %d, want at most %d", len(selected), a.Groups)
			}
			if hit < a.Found {
				t.Errorf("posts found: %d of %d, want at least %d", hit, len(wanted), a.Found)
			}
			if miss > a.Extra {
				t.Errorf("links offered that are not posts: %d, want at most %d", miss, a.Extra)
			}
		})
	}
	if posts > 0 {
		t.Logf("%d groups, %d of %d posts found, %d other links offered "+
			"(precision %.3f, recall %.3f)",
			groups, found, posts, extra,
			float64(found)/float64(found+extra), float64(found)/float64(posts))
	}
}

// postKey is the form used to compare a link with a fixture URL. The query and
// the fragment are kept, because developer.apple.com identifies a post by ?id=
// and some pages identify one by an anchor.
func postKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	key := strings.ToLower(u.Host) + strings.TrimSuffix(u.Path, "/")
	if u.RawQuery != "" {
		key += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		key += "#" + u.Fragment
	}
	return key
}

// groupAccepted is what every page produces today, not a target. Lowering the
// group counts and raising the posts found one page at a time is how an
// improvement becomes visible.
var groupAccepted = map[string]groupAcceptance{
	"anthropic.com-news":                    {3, 13, 0},
	"aws.amazon.com-jp-blogs-news":          {1, 10, 1},
	"bbc.com":                               {19, 91, 0},
	"blog.google":                           {1, 5, 0},
	"claude.com-blog":                       {4, 23, 0},
	"cursor.com-blog":                       {10, 24, 0},
	"daily.bandcamp.com-album-of-the-day":   {2, 30, 7},
	"daily.bandcamp.com-features":           {2, 30, 8},
	"deepmind.google-blog":                  {3, 25, 0},
	"deepmind.google-research-publications": {1, 30, 0},
	"developer.apple.com-news":              {1, 100, 7},
	"developers.openai.com-blog":            {1, 29, 0},
	"diggersfactory.com-vinyl-shop-new-ins": {1, 9, 0},
	"flutter.dev-blog":                      {1, 284, 0},
	"github.blog":                           {5, 21, 0},
	"github.blog-ai-and-ml":                 {2, 19, 0},
	"go.dev-blog":                           {1, 10, 1},
	"paulgraham.com-articles":               {2, 235, 1},
	"qiita.com":                             {4, 30, 6},
	"ycombinator.com-blog":                  {4, 9, 0},
	"ycombinator.com-blog-tag-essay":        {4, 10, 0},
}

// The two paths are benchmarked side by side on the two largest saved pages,
// because reading the posts out of the tree replaces the enumeration and has
// to stay in the same range of cost.

func benchmarkPage(b *testing.B, name, page string) ([]byte, url.URL) {
	b.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name+".html"))
	if err != nil {
		b.Fatal(err)
	}
	u, err := url.Parse(page)
	if err != nil {
		b.Fatal(err)
	}
	return data, *u
}

func BenchmarkSelectPostGroups(b *testing.B) {
	for _, p := range []struct{ name, url string }{
		{"developers.openai.com-blog", "https://developers.openai.com/blog"},
		{"developer.apple.com-news", "https://developer.apple.com/news/"},
	} {
		data, u := benchmarkPage(b, p.name, p.url)
		b.Run(p.name, func(b *testing.B) {
			for b.Loop() {
				root, err := BuildSemanticTree(bytes.NewReader(data), u)
				if err != nil {
					b.Fatal(err)
				}
				SelectPostGroups(root, 0.8)
			}
		})
	}
}

func BenchmarkEnumeratePostGroups(b *testing.B) {
	for _, p := range []struct{ name, url string }{
		{"developers.openai.com-blog", "https://developers.openai.com/blog"},
		{"developer.apple.com-news", "https://developer.apple.com/news/"},
	} {
		data, u := benchmarkPage(b, p.name, p.url)
		b.Run(p.name, func(b *testing.B) {
			for b.Loop() {
				if _, err := EnumeratePostGroups(bytes.NewReader(data), u); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
