package feed

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// update rewrites the fixture files from the current result instead of
// comparing against them. Run it after a deliberate change to the method:
//
//	go test ./server/feature/feed/ -run TestDetectPostLists_SavedPages -update
//
// Read the diff before committing it. The fixtures are not all correct
// results: bbc.com mixes section links into its posts, and the pages that are
// rendered by JavaScript hold no post at all. They record what the method finds
// today, so that a change shows which pages it moves and in which direction.
var update = flag.Bool("update", false, "rewrite the fixture files in testdata/")

// fixture is the expected result for one saved page. It lives next to the page
// as <page>.fixture.json. url is the address the page was fetched from, which
// is needed to resolve the links, and note says what is known to be wrong with
// the result. Both are written by hand and kept when the fixture is rewritten.
type fixture struct {
	URL   string        `json:"url"`
	Note  string        `json:"note,omitempty"`
	Lists []fixtureList `json:"lists"`
}

type fixtureList struct {
	ID       int           `json:"id"`
	Selector string        `json:"selector"`
	Score    float64       `json:"score"`
	Posts    []fixturePost `json:"posts"`
}

type fixturePost struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Timestamp string `json:"timestamp,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
}

// The pages in testdata/ were saved with curl, without running any JavaScript,
// so each file is what a plain HTTP client receives. Every attribute of every
// post is compared against the fixture of the page, so that a change to the
// method shows the exact posts, titles, dates and images it adds, drops or
// rewrites.
func TestDetectPostLists_SavedPages(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no fixture found in testdata/")
	}
	sort.Strings(paths)

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			want := readFixture(t, path)
			page := strings.TrimSuffix(filepath.Base(path), ".fixture.json") + ".html"
			lists, doc := detectFile(t, page, want.URL)
			got := fixture{URL: want.URL, Note: want.Note, Lists: []fixtureList{}}
			for _, l := range lists {
				fl := fixtureList{ID: l.ID, Selector: l.Selector, Score: l.Score, Posts: []fixturePost{}}
				for _, p := range l.Posts {
					fp := fixturePost{URL: p.URL.String(), Title: p.Title, Timestamp: p.Timestamp}
					if p.ImageURL != nil {
						fp.ImageURL = p.ImageURL.String()
					}
					fl.Posts = append(fl.Posts, fp)
				}
				got.Lists = append(got.Lists, fl)
			}

			if *update {
				writeFixture(t, path, got)
				return
			}

			checkStructure(t, lists, doc)
			if len(got.Lists) != len(want.Lists) {
				t.Errorf("got %d lists, want %d", len(got.Lists), len(want.Lists))
			}
			for i := range got.Lists {
				if i >= len(want.Lists) {
					t.Errorf("list %d is not in the fixture: selector %q, %d posts",
						i+1, got.Lists[i].Selector, len(got.Lists[i].Posts))
					continue
				}
				comparePosts(t, got.Lists[i], want.Lists[i])
			}
		})
	}
}

// comparePosts reports every difference between one detected list and the list
// the fixture expects, post by post, so that a failure names the posts that
// changed rather than only the number of them.
func comparePosts(t *testing.T, got, want fixtureList) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("list %d: got ID %d, want %d", want.ID, got.ID, want.ID)
	}
	if got.Selector != want.Selector {
		t.Errorf("list %d: got selector %q, want %q", want.ID, got.Selector, want.Selector)
	}
	if got.Score != want.Score {
		t.Errorf("list %d: got score %v, want %v", want.ID, got.Score, want.Score)
	}
	if len(got.Posts) != len(want.Posts) {
		t.Errorf("list %d: got %d posts, want %d", want.ID, len(got.Posts), len(want.Posts))
	}
	for i := 0; i < len(got.Posts) && i < len(want.Posts); i++ {
		g, w := got.Posts[i], want.Posts[i]
		if g.URL != w.URL {
			t.Errorf("list %d post %d: got URL %q, want %q", want.ID, i, g.URL, w.URL)
		}
		if g.Title != w.Title {
			t.Errorf("list %d post %d (%s): got title %q, want %q", want.ID, i, w.URL, g.Title, w.Title)
		}
		if g.Timestamp != w.Timestamp {
			t.Errorf("list %d post %d (%s): got timestamp %q, want %q", want.ID, i, w.URL, g.Timestamp, w.Timestamp)
		}
		if g.ImageURL != w.ImageURL {
			t.Errorf("list %d post %d (%s): got image %q, want %q", want.ID, i, w.URL, g.ImageURL, w.ImageURL)
		}
	}
	for i := len(got.Posts); i < len(want.Posts); i++ {
		t.Errorf("list %d post %d (%s) was not detected", want.ID, i, want.Posts[i].URL)
	}
	for i := len(want.Posts); i < len(got.Posts); i++ {
		t.Errorf("list %d post %d (%s) is not in the fixture", want.ID, i, got.Posts[i].URL)
	}
}

// checkStructure verifies what holds for every page, whatever its fixture says:
// the lists are numbered and ordered, every post carries an absolute URL and
// the node it was read from, no URL is reported twice, and the selector of a
// list finds its posts again in the document.
func checkStructure(t *testing.T, lists []PostList, doc *html.Node) {
	t.Helper()
	seen := map[string]bool{}
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
		for _, p := range l.Posts {
			if !p.URL.IsAbs() {
				t.Errorf("post URL %q is not absolute", p.URL.String())
			}
			// The same post can be rendered twice on one page, as a grid and
			// as a list. It must be reported once.
			if seen[p.URL.String()] {
				t.Errorf("post URL %q is reported twice", p.URL.String())
			}
			seen[p.URL.String()] = true
			if p.Node == nil {
				t.Errorf("post %q carries no node", p.URL.String())
			}
		}
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

func readFixture(t *testing.T, path string) fixture {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f fixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if f.URL == "" {
		t.Fatalf("%s carries no url", path)
	}
	return f
}

func writeFixture(t *testing.T, path string, f fixture) {
	t.Helper()
	// The default encoder escapes "<", ">" and "&", which would make every
	// selector in the file unreadable.
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(f); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
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
