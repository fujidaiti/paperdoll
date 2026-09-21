package feed

import (
	"net/url"
	"strings"
	"testing"
)

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
