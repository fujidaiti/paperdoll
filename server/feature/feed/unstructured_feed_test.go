package feed

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// The pages in testdata/ were saved with curl, without running any JavaScript,
// so each file is what a plain HTTP client receives. The counts below are what
// the method currently finds on them, measured page by page and described in
// docs/html-feed-detection.md. They are not all correct results: bbc.com mixes
// 19 section links into its 109 links, and the two pages that produce nothing
// are client rendered. The numbers are here so that a change to the method
// shows which pages it moves, and in which direction.
var savedPages = map[string]struct {
	url   string
	lists int
	posts int
}{
	"deepmind.google-research-publications.html": {"https://deepmind.google/research/publications/", 1, 30},
	"claude.com-blog.html":                       {"https://claude.com/blog", 2, 23},
	"developers.openai.com-blog.html":            {"https://developers.openai.com/blog", 1, 29},
	"anthropic.com-news.html":                    {"https://www.anthropic.com/news", 2, 12},
	"cursor.com-blog.html":                       {"https://cursor.com/blog", 4, 24},
	"deepmind.google-blog.html":                  {"https://deepmind.google/blog/", 2, 24},
	"paulgraham.com-articles.html":               {"https://www.paulgraham.com/articles.html", 1, 235},
	"developer.apple.com-news.html":              {"https://developer.apple.com/news/", 1, 108},
	// Its post items carry no link at all, so it holds nothing a feed can be
	// built from.
	"security.apple.com-blog.html": {"https://security.apple.com/blog/", 0, 0},
	// Client rendered: the saved file holds no post.
	"apple.com-newsroom.html":                    {"https://www.apple.com/newsroom/", 0, 0},
	"github.blog-ai-and-ml.html":                 {"https://github.blog/ai-and-ml/", 2, 19},
	"github.blog.html":                           {"https://github.blog/", 6, 27},
	"bbc.com.html":                               {"https://www.bbc.com/", 19, 109},
	"ycombinator.com-blog.html":                  {"https://www.ycombinator.com/blog", 5, 24},
	"ycombinator.com-blog-tag-essay.html":        {"https://www.ycombinator.com/blog/tag/essay", 2, 10},
	"blog.google.html":                           {"https://blog.google/", 2, 7},
	"blog.google-feed.html":                      {"https://blog.google/feed/", 0, 0},
	"aws.amazon.com-jp-blogs-news.html":          {"https://aws.amazon.com/jp/blogs/news/", 2, 11},
	"flutter.dev-blog.html":                      {"https://flutter.dev/blog", 1, 284},
	"go.dev-blog.html":                           {"https://go.dev/blog/", 2, 11},
	"reddit.com-r-golang.html":                   {"https://www.reddit.com/r/golang/", 0, 0},
	"qiita.com.html":                             {"https://qiita.com/", 2, 22},
	"daily.bandcamp.com-features.html":           {"https://daily.bandcamp.com/features", 1, 30},
	"daily.bandcamp.com-album-of-the-day.html":   {"https://daily.bandcamp.com/album-of-the-day", 1, 30},
	"diggersfactory.com-vinyl-shop-new-ins.html": {"https://www.diggersfactory.com/vinyl-shop/293/new-ins", 5, 35},
}

func TestDetectPostLists_SavedPages(t *testing.T) {
	for file, want := range savedPages {
		t.Run(file, func(t *testing.T) {
			lists, doc := detectFile(t, file, want.url)
			if len(lists) != want.lists {
				t.Errorf("got %d lists, want %d", len(lists), want.lists)
			}
			posts, seen := 0, map[string]bool{}
			for i, l := range lists {
				if l.ID != i+1 {
					t.Errorf("list %d has ID %d, want %d", i, l.ID, i+1)
				}
				if i > 0 && l.Score > lists[i-1].Score {
					t.Errorf("list %d scores %.1f, above the list before it (%.1f)", i, l.Score, lists[i-1].Score)
				}
				if len(l.Posts) < 1 {
					t.Errorf("list %d holds no post", l.ID)
				}
				// The same post can be rendered twice on one page, as a grid
				// and as a list. It must be reported once.
				for _, p := range l.Posts {
					posts++
					if !p.URL.IsAbs() {
						t.Errorf("post URL %q is not absolute", p.URL.String())
					}
					if seen[p.URL.String()] {
						t.Errorf("post URL %q is reported twice", p.URL.String())
					}
					seen[p.URL.String()] = true
					if p.Node == nil {
						t.Errorf("post %q carries no node", p.URL.String())
					}
				}
				// The selector has to find the posts again, so that a caller
				// can re-read this list from a later copy of the page.
				matched := map[*html.Node]bool{}
				goquery.NewDocumentFromNode(doc).Find(l.Selector).Each(func(_ int, s *goquery.Selection) {
					matched[s.Nodes[0]] = true
				})
				for _, p := range l.Posts {
					if !matched[p.Node] {
						t.Errorf("selector %q does not match the post %q", l.Selector, p.URL.String())
						break
					}
				}
			}
			if posts != want.posts {
				t.Errorf("got %d posts, want %d", posts, want.posts)
			}
		})
	}
}

// The attributes are best effort, so they are measured on the pages that
// publish them rather than required everywhere.
func TestDetectPostLists_PostAttributes(t *testing.T) {
	lists, _ := detectFile(t, "go.dev-blog.html", "https://go.dev/blog/")
	var got *Post
	for _, l := range lists {
		for i, p := range l.Posts {
			if strings.HasSuffix(p.URL.Path, "/blog/size-specialized-allocations") {
				got = &l.Posts[i]
			}
		}
	}
	if got == nil {
		t.Fatal("the post size-specialized-allocations was not detected")
	}
	if want := "Size-Specialized Memory Allocation"; got.Title != want {
		t.Errorf("got title %q, want %q", got.Title, want)
	}
	if want := "16 September 2026"; got.Timestamp != want {
		t.Errorf("got timestamp %q, want %q", got.Timestamp, want)
	}
}

func TestDetectPostLists_Attributes(t *testing.T) {
	page := `<html><body><main><ul>
		<li><a href="/p/1"><img src="/one.png"><h2>First post</h2></a>
		    <time datetime="2026-01-02T10:00:00Z">two days ago</time>
		    <p>The first post of the year, with enough text to look like one.</p></li>
		<li><a href="/p/2"><h2>Second post</h2></a>
		    <span>Jan 3, 2026</span>
		    <p>The second post of the year, with enough text to look like one.</p></li>
		<li><a href="/p/3"><h2>Third post</h2></a>
		    <p>The third post of the year, with enough text to look like one.</p></li>
	</ul></main></body></html>`

	u, err := url.Parse("https://example.test/blog")
	if err != nil {
		t.Fatal(err)
	}
	lists, err := DetectPostLists(strings.NewReader(page), *u)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Fatalf("got %d lists, want 1", len(lists))
	}
	l := lists[0]
	if l.ID != 1 {
		t.Errorf("got ID %d, want 1", l.ID)
	}
	// The three items differ in shape, because only the first one carries an
	// image and only the first two carry a date, so they are detected as the
	// group of all children of the <ul> rather than as one shape. The
	// selector for that group ends in "*".
	if want := "html > body > main > ul > *"; l.Selector != want {
		t.Errorf("got selector %q, want %q", l.Selector, want)
	}
	if len(l.Posts) != 3 {
		t.Fatalf("got %d posts, want 3", len(l.Posts))
	}

	first := l.Posts[0]
	if want := "https://example.test/p/1"; first.URL.String() != want {
		t.Errorf("got URL %q, want %q", first.URL.String(), want)
	}
	if want := "First post"; first.Title != want {
		t.Errorf("got title %q, want %q", first.Title, want)
	}
	// The datetime attribute wins over the text of the <time> element.
	if want := "2026-01-02T10:00:00Z"; first.Timestamp != want {
		t.Errorf("got timestamp %q, want %q", first.Timestamp, want)
	}
	if first.ImageURL == nil || first.ImageURL.String() != "https://example.test/one.png" {
		t.Errorf("got image %v, want https://example.test/one.png", first.ImageURL)
	}
	if first.Node == nil || first.Node.Data != "li" {
		t.Errorf("got node %v, want the <li> element", first.Node)
	}

	// The date written in the text is read when there is no <time> element,
	// and it is kept as the page writes it.
	if second, want := l.Posts[1], "Jan 3, 2026"; second.Timestamp != want {
		t.Errorf("got timestamp %q, want %q", second.Timestamp, want)
	}
	// A post without a date or an image is still a post.
	if third := l.Posts[2]; third.Timestamp != "" || third.ImageURL != nil {
		t.Errorf("got timestamp %q and image %v, want both empty", third.Timestamp, third.ImageURL)
	}
}

// Site navigation, the page banner and the page footer hold many links that
// repeat on every page of the site, and they are the main source of groups that
// look like a post list but are not.
func TestDetectPostLists_IgnoresNavigationAndFooter(t *testing.T) {
	page := `<html><body>
		<nav><ul><li><a href="/a">About us and the team</a></li>
		         <li><a href="/b">Careers at the company</a></li>
		         <li><a href="/c">Contact the support team</a></li></ul></nav>
		<footer><ul><li><a href="/x">Terms of service and conditions</a></li>
		            <li><a href="/y">Privacy policy of the company</a></li>
		            <li><a href="/z">Cookie settings for this site</a></li></ul></footer>
	</body></html>`

	u, err := url.Parse("https://example.test/")
	if err != nil {
		t.Fatal(err)
	}
	lists, err := DetectPostLists(strings.NewReader(page), *u)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 0 {
		t.Errorf("got %d lists, want none", len(lists))
	}
}

func detectFile(t *testing.T, file, page string) ([]PostList, *html.Node) {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", file))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	u, err := url.Parse(page)
	if err != nil {
		t.Fatal(err)
	}
	// The document is parsed twice: DetectPostLists cleans up the tree it
	// parses, and the selectors are checked against that same tree, which the
	// returned nodes belong to. Reading it back from the posts is enough.
	lists, err := DetectPostLists(f, *u)
	if err != nil {
		t.Fatal(err)
	}
	var root *html.Node
	if len(lists) > 0 {
		for n := lists[0].Posts[0].Node; n != nil; n = n.Parent {
			root = n
		}
	}
	return lists, root
}
