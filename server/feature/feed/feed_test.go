package feed

import (
	"testing"
)

// A page that publishes a feed names it in its head, and the flow has to end
// up on that feed rather than on the page's own post groups.
func TestFeedLink(t *testing.T) {
	page := mustParse(t, "https://developer.apple.com/news/")

	tests := []struct {
		name, head, want string
	}{
		{
			name: "an absolute rss link",
			head: `<link rel="alternate" type="application/rss+xml" href="https://example.test/feed.xml">`,
			want: "https://example.test/feed.xml",
		},
		{
			name: "a relative atom link",
			head: `<link rel="alternate" type="application/atom+xml" href="/news/atom.xml">`,
			want: "https://developer.apple.com/news/atom.xml",
		},
		{
			// developer.apple.com writes this scheme, which no HTTP client can
			// fetch.
			name: "the feed scheme",
			head: `<link rel="alternate" type="application/rss+xml" href="feed://developer.apple.com/news/rss/news.rss">`,
			want: "https://developer.apple.com/news/rss/news.rss",
		},
		{
			name: "a link to something that is not a feed",
			head: `<link rel="alternate" type="application/json" href="/news/feed.json">`,
			want: "",
		},
		{
			name: "no link at all",
			head: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte("<html><head>" + tt.head + "</head><body><p>News</p></body></html>")
			got, ok := feedLink(body, page)
			if tt.want == "" {
				if ok {
					t.Errorf("got %q, want no feed", got.String())
				}
				return
			}
			if !ok {
				t.Fatalf("got no feed, want %q", tt.want)
			}
			if got.String() != tt.want {
				t.Errorf("got %q, want %q", got.String(), tt.want)
			}
		})
	}
}

// The attributes of a page feed come from the page itself, because there is no
// feed document to read them from.
func TestSourceAttrs_Page(t *testing.T) {
	page := mustParse(t, "https://example.test/blog")
	body := []byte(`<html><head><title>  The Example  Blog </title></head><body><p>Posts</p></body></html>`)

	a := source{url: page, body: body}.attrs()
	if want := "The Example Blog"; a.Title != want {
		t.Errorf("got title %q, want %q", a.Title, want)
	}
	if a.SiteURL == nil || a.SiteURL.String() != page.String() {
		t.Errorf("got site url %v, want %s", a.SiteURL, page.String())
	}
	if a.URL != page {
		t.Errorf("got url %v, want %v", a.URL, page)
	}
	if a.Description != nil {
		t.Errorf("got description %q, want none", *a.Description)
	}
}
