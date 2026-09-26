package feed

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// The document below is the smallest page that produces one group: two items
// of the same shape, each carrying a link, a heading, a date, a paragraph and
// an image. Everything above the items, up to <body>, also forms a group of
// one item, and every one of those is removed by the subset rule, because the
// first link of its single item is the first link of the group below it.
const twoPostPage = `<html><body><main><ul>
	<li><a href="/p/1"><h2>First post</h2></a>
	    <time datetime="2026-01-02T10:00:00Z">two days ago</time>
	    <p>The first post of the year.</p>
	    <img src="/one.png" alt="One"></li>
	<li><a href="/p/2"><h2>Second post</h2></a>
	    <time datetime="2026-01-03T10:00:00Z">yesterday</time>
	    <p>The second post of the year.</p>
	    <img src="/two.png" alt="Two"></li>
</ul></main></body></html>`

func TestEnumeratePostGroups(t *testing.T) {
	groups := enumerate(t, twoPostPage, "https://example.test/blog")
	if len(groups) != 1 {
		for _, g := range groups {
			t.Logf("group %q", g.Selector)
		}
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	g := groups[0]
	if want := "html > body > main > ul > li"; g.Selector != want {
		t.Errorf("got key %q, want %q", g.Selector, want)
	}
	if len(g.Posts) != 2 {
		t.Fatalf("got %d items, want 2", len(g.Posts))
	}

	first := g.Posts[0]
	if first.Node == nil || first.Node.Data != "li" {
		t.Errorf("got node %v, want the <li> element", first.Node)
	}
	checkAttrs(t, "links", first.Links, []Attribute{
		{Selector: "a", Value: "https://example.test/p/1"},
	})
	// The datetime attribute is reported instead of the text of the <time>
	// element, and the <a> is not reported, because its heading holds the
	// text.
	checkAttrs(t, "texts", first.Texts, []Attribute{
		{Selector: "a > h2", Value: "First post"},
		{Selector: "time", Value: "2026-01-02T10:00:00Z"},
		{Selector: "p", Value: "The first post of the year."},
	})
	checkAttrs(t, "images", first.Images, []Attribute{
		{Selector: "img", Value: "https://example.test/one.png", Alt: "One"},
	})

	// The second item writes the same keys as the first, which is the whole
	// point of the key format: one saved key reads every item of the group.
	second := g.Posts[1]
	for i, a := range second.Texts {
		if a.Selector != first.Texts[i].Selector {
			t.Errorf("text %d: the two items write different keys, %q and %q",
				i, first.Texts[i].Selector, a.Selector)
		}
	}
}

// A container can hold two shapes of the same tag, for example a featured card
// and a plain one, and the key is built from tag names only. The shape number
// is what keeps the two keys apart.
func TestEnumeratePostGroups_ShapeNumber(t *testing.T) {
	page := `<html><body><main><ul>
		<li><a href="/p/1"><h2>First post</h2></a></li>
		<li><a href="/p/2"><h2>Second post</h2></a></li>
		<li><a href="/q/1">A plain row</a><span>x</span></li>
		<li><a href="/q/2">Another plain row</a><span>y</span></li>
	</ul></main></body></html>`

	groups := enumerate(t, page, "https://example.test/blog")
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
	if want := "html > body > main > ul > li"; groups[0].Selector != want {
		t.Errorf("got key %q, want %q", groups[0].Selector, want)
	}
	if want := "html > body > main > ul > li:nth-shape(2)"; groups[1].Selector != want {
		t.Errorf("got key %q, want %q", groups[1].Selector, want)
	}
}

func TestExtractPosts(t *testing.T) {
	const root = "html > body > main > ul > li"
	page := "https://example.test/blog"
	set := Selectors{Root: root, Link: "a", Title: "a > h2", Image: "img", Timestamp: "time"}

	posts := extract(t, twoPostPage, page, []Selectors{set})
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want 2", len(posts))
	}
	p := posts[0]
	if want := "https://example.test/p/1"; p.URL.String() != want {
		t.Errorf("got url %q, want %q", p.URL.String(), want)
	}
	if want := "First post"; p.Title != want {
		t.Errorf("got title %q, want %q", p.Title, want)
	}
	if want := "2026-01-02T10:00:00Z"; p.Timestamp != want {
		t.Errorf("got timestamp %q, want %q", p.Timestamp, want)
	}
	if p.ImageURL == nil || p.ImageURL.String() != "https://example.test/one.png" {
		t.Errorf("got image %v, want https://example.test/one.png", p.ImageURL)
	}
	// The user picked no key for the description, which is not the same as a
	// key the items do not carry.
	if p.Description != "" {
		t.Errorf("got description %q, want none", p.Description)
	}

	// Two sets over the same items report each post once.
	posts = extract(t, twoPostPage, page, []Selectors{set, set})
	if len(posts) != 2 {
		t.Errorf("got %d posts from the same set twice, want 2", len(posts))
	}

	// An optional key that no item carries leaves the field empty rather than
	// failing.
	posts = extract(t, twoPostPage, page, []Selectors{{Root: root, Link: "a", Title: "div > h4"}})
	if len(posts) != 2 || posts[0].Title != "" {
		t.Errorf("got %d posts with title %q, want 2 posts with no title", len(posts), posts[0].Title)
	}

	// A root that names no group, and a link that no item carries, are both
	// refused: saving either would create a feed that stays empty forever.
	for _, s := range []Selectors{
		{Root: "html > body > main > ol > li", Link: "a"},
		{Root: root, Link: "div > a"},
	} {
		_, err := ExtractPosts(strings.NewReader(twoPostPage), mustParse(t, page), []Selectors{s})
		if !errors.Is(err, ErrSelectors) {
			t.Errorf("extracting with %+v: got error %v, want ErrSelectors", s, err)
		}
	}
}

// An item of a group is free not to carry the link key the user picked, for
// example because the page writes a self link where the others write the post
// link. Such an item is skipped instead of being reported without a URL.
func TestExtractPosts_ItemWithoutTheLinkKey(t *testing.T) {
	page := `<html><body><main><ul>
		<li><a href="/p/1"><h2>First post</h2></a><a href="/tags/one">tag</a></li>
		<li><a href="/blog"><h2>Second post</h2></a><a href="/tags/two">tag</a></li>
	</ul></main></body></html>`

	posts := extract(t, page, "https://example.test/blog", []Selectors{{
		Root: "html > body > main > ul > li",
		Link: "a:nth-of-type(1)",
	}})
	if len(posts) != 1 {
		t.Fatalf("got %d posts, want 1", len(posts))
	}
	if want := "https://example.test/p/1"; posts[0].URL.String() != want {
		t.Errorf("got url %q, want %q", posts[0].URL.String(), want)
	}
}

// The keys a user answers on come from one enumeration and are read back by
// another, so the two runs have to write the same strings over a real page.
func TestExtractPosts_SavedPage(t *testing.T) {
	const name, page = "claude.com-blog", "https://claude.com/blog"
	groups := enumerateFile(t, name+".html", page)
	if len(groups) == 0 {
		t.Fatal("the saved page produced no group")
	}
	// The largest group of the page, read through the key of the first link
	// of its first item.
	best := 0
	for i, g := range groups {
		if len(g.Posts) > len(groups[best].Posts) {
			best = i
		}
	}
	g := groups[best]
	set := Selectors{Root: g.Selector, Link: g.Posts[0].Links[0].Selector}
	if len(g.Posts[0].Texts) > 0 {
		set.Title = g.Posts[0].Texts[0].Selector
	}

	f, err := os.Open(filepath.Join("testdata", name+".html"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	posts, err := ExtractPosts(f, mustParse(t, page), []Selectors{set})
	if err != nil {
		t.Fatal(err)
	}

	var want []string
	for _, p := range g.Posts {
		if v, ok := valueOf(p.Links, set.Link); ok {
			want = append(want, v)
		}
	}
	if len(posts) != len(want) {
		t.Fatalf("got %d posts, want %d", len(posts), len(want))
	}
	for i, p := range posts {
		if p.URL.String() != want[i] {
			t.Errorf("post %d: got %q, want %q", i, p.URL.String(), want[i])
		}
	}
	if set.Title != "" && posts[0].Title == "" {
		t.Errorf("the title key %q read nothing back", set.Title)
	}
}

func enumerate(t *testing.T, page, pageURL string) []Group {
	t.Helper()
	groups, err := EnumeratePostGroups(strings.NewReader(page), mustParse(t, pageURL))
	if err != nil {
		t.Fatal(err)
	}
	return groups
}

func extract(t *testing.T, page, pageURL string, sets []Selectors) []Post {
	t.Helper()
	posts, err := ExtractPosts(strings.NewReader(page), mustParse(t, pageURL), sets)
	if err != nil {
		t.Fatal(err)
	}
	return posts
}

func mustParse(t *testing.T, raw string) url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return *u
}

func checkAttrs(t *testing.T, kind string, got, want []Attribute) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("got %d %s, want %d: %+v", len(got), kind, len(want), got)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s %d: got %+v, want %+v", kind, i, got[i], want[i])
		}
	}
}

// TestSanitize checks the cleaning step against a fixture per saved page. The
// fixtures were produced by sanitize itself, so they are a record of what the
// cleaning rules remove today rather than a hand written ideal: a change in
// those rules shows up here as a diff of the cleaned HTML, which is easier to
// read than the shift it causes in the accuracy metrics.
func TestSanitize(t *testing.T) {
	pages, err := filepath.Glob(filepath.Join("testdata", "*.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("no saved page found in testdata")
	}

	for _, page := range pages {
		name := strings.TrimSuffix(filepath.Base(page), ".html")
		t.Run(name, func(t *testing.T) {
			f, err := os.Open(page)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()
			doc, err := sanitize(f)
			if err != nil {
				t.Fatal(err)
			}
			var b strings.Builder
			if err := html.Render(&b, doc); err != nil {
				t.Fatal(err)
			}

			fixture := filepath.Join("testdata", "sanitize", name+".fixture.html")
			want, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}

			got, wantStr := normalize(t, b.String()), normalize(t, string(want))
			if got == wantStr {
				return
			}
			// The documents are hundreds of kilobytes, so only the place where
			// they start to differ is printed.
			at := 0
			for at < len(got) && at < len(wantStr) && got[at] == wantStr[at] {
				at++
			}
			const context = 200
			from := max(at-context, 0)
			t.Errorf("the cleaned page differs from %s at byte %d:\n  got  %s\n  want %s",
				fixture, at,
				got[from:min(at+context, len(got))],
				wantStr[from:min(at+context, len(wantStr))])
		})
	}
}

// normalize renders an HTML string in a form that does not depend on how the
// file is laid out: it drops comments, collapses the whitespace of every text
// node and of every attribute value, and removes the text nodes left empty.
// Both sides of the comparison pass through it, so a fixture reformatted with
// prettier still matches, while the element tree and the attribute values
// themselves are compared as they are. Attribute values are normalized because
// prettier breaks a long class list or a long srcset over several lines.
// Parsing here also puts both sides through the same wrapping rules, which
// matters because cleanup removes <head> and a parser adds an empty one back.
func normalize(t *testing.T, page string) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		var next *html.Node
		for c := n.FirstChild; c != nil; c = next {
			next = c.NextSibling
			if c.Type == html.CommentNode {
				n.RemoveChild(c)
				continue
			}
			if c.Type == html.TextNode {
				c.Data = strings.Join(strings.Fields(c.Data), " ")
				if c.Data == "" {
					n.RemoveChild(c)
				}
				continue
			}
			for i, a := range c.Attr {
				c.Attr[i].Val = strings.Join(strings.Fields(a.Val), " ")
			}
			walk(c)
		}
	}
	walk(doc)
	var b strings.Builder
	if err := html.Render(&b, doc); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
