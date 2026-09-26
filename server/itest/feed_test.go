//go:build integration

package itest

import (
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
	"github.com/google/go-cmp/cmp"
)

type feedSubscriptionRecord struct {
	FeedID int
	UserID user.UserID
}

type feedRecord struct {
	ID          int
	URL         string
	SiteURL     sql.NullString
	IconURL     sql.NullString
	Title       string
	Description sql.NullString
}

// feedValue is a cmp-friendly view of a [feed.Feed]'s attributes where every
// URL and the optional description are flattened to strings ("" for nil).
//
// TODO: add a custom comparable URL type and use it throughout the codebase
type feedValue struct {
	URL         string
	SiteURL     string
	IconURL     string
	Title       string
	Description string
}

func newFeedValue(f feed.Feed) feedValue {
	urlString := func(u *url.URL) string {
		if u == nil {
			return ""
		}
		return u.String()
	}
	v := feedValue{
		URL:     f.URL.String(),
		SiteURL: urlString(f.SiteURL),
		IconURL: urlString(f.IconURL),
		Title:   f.Title,
	}
	if f.Description != nil {
		v.Description = *f.Description
	}
	return v
}

// nullString maps "" to a NULL, mirroring how [feed.Service.Subscribe] stores
// a nil pointer.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func TestFeed_Subscribe(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	tests := []struct {
		name, host, path, fixture, feedURL string
		want                               feed.Feed
	}{
		{
			name:    "wikipedia recent changes",
			host:    "en.wikipedia.org",
			path:    "/w/api.php",
			fixture: "./testdata/feed/wikipedia_recent_changes.xml",
			feedURL: "http://en.wikipedia.org/w/api.php?limit=50&action=feedrecentchanges&feedformat=rss",
			want: feed.Feed{FeedAttrs: feed.FeedAttrs{
				URL:         *must(url.Parse("http://en.wikipedia.org/w/api.php?limit=50&action=feedrecentchanges&feedformat=rss")),
				SiteURL:     must(url.Parse("https://en.wikipedia.org/wiki/Special:RecentChanges")),
				IconURL:     nil, // no <image> in the feed
				Title:       "Wikipedia  - Recent changes [en]",
				Description: new("Track the most recent changes to the wiki in this feed."),
			}},
		},
		{
			name:    "nasa news releases",
			host:    "www.nasa.gov",
			path:    "/news-release/feed/",
			fixture: "./testdata/feed/nasa_news_release.xml",
			feedURL: "http://www.nasa.gov/news-release/feed/",
			want: feed.Feed{FeedAttrs: feed.FeedAttrs{
				URL:         *must(url.Parse("http://www.nasa.gov/news-release/feed/")),
				SiteURL:     must(url.Parse("https://www.nasa.gov")),
				IconURL:     nil, // no <image> in the feed
				Title:       "NASA",
				Description: new("Official National Aeronautics and Space Administration Website"),
			}},
		},
		{
			name:    "stackoverflow recent questions",
			host:    "stackoverflow.com",
			path:    "/feeds",
			fixture: "./testdata/feed/stackoverflow_feeds.xml",
			feedURL: "http://stackoverflow.com/feeds",
			want: feed.Feed{FeedAttrs: feed.FeedAttrs{
				URL:         *must(url.Parse("http://stackoverflow.com/feeds")),
				SiteURL:     must(url.Parse("https://stackoverflow.com/questions")),
				IconURL:     nil, // no <icon>/<logo> in the feed
				Title:       "Recent Questions - Stack Overflow",
				Description: new("most recent 30 from stackoverflow.com"),
			}},
		},
		{
			name:    "bbc news front page",
			host:    "feeds.bbci.co.uk",
			path:    "/news/rss.xml",
			fixture: "./testdata/feed/bbc_news_rss.xml",
			feedURL: "http://feeds.bbci.co.uk/news/rss.xml",
			want: feed.Feed{FeedAttrs: feed.FeedAttrs{
				URL:         *must(url.Parse("http://feeds.bbci.co.uk/news/rss.xml")),
				SiteURL:     must(url.Parse("https://www.bbc.co.uk/news")),
				IconURL:     must(url.Parse("https://news.bbcimg.co.uk/nol/shared/img/bbc_news_120x60.gif")),
				Title:       "BBC News",
				Description: new("BBC News - News Front Page"),
			}},
		},
	}

	// Seed a user
	uid := provisionDefaultTestAccount(t, mustTimeUTC("2026-07-01 13:30:00"))

	for _, tt := range tests {
		testenv.StubHTTP(tt.host, tt.path, tt.fixture)
		t.Run(tt.name, func(t *testing.T) {
			feedURL := must(url.Parse(tt.feedURL))
			s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
			fd, err := s.Subscribe(t.Context(), uid, *feedURL, nil)
			if err != nil {
				t.Fatalf("got %q, want a nil error", err)
			}

			if fd.ID == 0 {
				t.Errorf("feed ID must be assigned, got %d", fd.ID)
			}
			wantFeed := newFeedValue(tt.want)
			if d := cmp.Diff(wantFeed, newFeedValue(fd)); d != "" {
				t.Errorf("returned feed mismatch:\n%s", d)
			}

			var got feedRecord
			scanRowOrFatal(t, `
				SELECT id, url, site_url, icon_url, title, description FROM feeds WHERE url = $1
			`, []any{feedURL.String()}, &got.ID, &got.URL, &got.SiteURL, &got.IconURL, &got.Title, &got.Description)
			want := feedRecord{
				ID:          fd.ID,
				URL:         wantFeed.URL,
				SiteURL:     nullString(wantFeed.SiteURL),
				IconURL:     nullString(wantFeed.IconURL),
				Title:       wantFeed.Title,
				Description: nullString(wantFeed.Description),
			}
			if d := cmp.Diff(got, want); d != "" {
				t.Errorf("stored feed record mismatch:\n%s", d)
			}

			var got2 feedSubscriptionRecord
			scanRowOrFatal(t, `
				SELECT feed_id, user_id
				FROM feed_subscriptions ORDER BY created_at DESC LIMIT 1
			`, []any{}, &got2.FeedID, &got2.UserID)
			want2 := feedSubscriptionRecord{
				FeedID: fd.ID,
				UserID: uid,
			}
			if d := cmp.Diff(got2, want2); d != "" {
				t.Errorf("stored subscription record mismatch:\n%s", d)
			}
		})
	}
}

func TestFeed_SearchFeeds(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	tests := []struct {
		name, host, path, fixture, query string
		want                             feed.FeedAttrs
	}{
		{
			name:    "wikipedia recent changes",
			host:    "en.wikipedia.org",
			path:    "/w/api.php",
			fixture: "./testdata/feed/wikipedia_recent_changes.xml",
			query:   "http://en.wikipedia.org/w/api.php?limit=50&action=feedrecentchanges&feedformat=rss",
			want: feed.FeedAttrs{
				URL:         *must(url.Parse("http://en.wikipedia.org/w/api.php?limit=50&action=feedrecentchanges&feedformat=rss")),
				SiteURL:     must(url.Parse("https://en.wikipedia.org/wiki/Special:RecentChanges")),
				IconURL:     nil, // no <image> in the feed
				Title:       "Wikipedia  - Recent changes [en]",
				Description: new("Track the most recent changes to the wiki in this feed."),
			},
		},
		{
			name:    "bbc news front page",
			host:    "feeds.bbci.co.uk",
			path:    "/news/rss.xml",
			fixture: "./testdata/feed/bbc_news_rss.xml",
			query:   "http://feeds.bbci.co.uk/news/rss.xml",
			want: feed.FeedAttrs{
				URL:         *must(url.Parse("http://feeds.bbci.co.uk/news/rss.xml")),
				SiteURL:     must(url.Parse("https://www.bbc.co.uk/news")),
				IconURL:     must(url.Parse("https://news.bbcimg.co.uk/nol/shared/img/bbc_news_120x60.gif")),
				Title:       "BBC News",
				Description: new("BBC News - News Front Page"),
			},
		},
	}

	for _, tt := range tests {
		testenv.StubHTTP(tt.host, tt.path, tt.fixture)
		t.Run(tt.name, func(t *testing.T) {
			s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
			got, err := s.SearchFeeds(t.Context(), tt.query)
			if err != nil {
				t.Fatalf("got %v, want a nil error", err)
			}

			if len(got) != 1 {
				t.Fatalf("got %d results, want exactly 1", len(got))
			}
			wantValue := newFeedValue(feed.Feed{FeedAttrs: tt.want})
			if d := cmp.Diff(wantValue, newFeedValue(feed.Feed{FeedAttrs: got[0].FeedAttrs})); d != "" {
				t.Errorf("returned feed mismatch:\n%s", d)
			}
			// A real feed is subscribed to with its URL alone, so the client
			// must not be sent a group screen for it.
			if len(got[0].Groups) != 0 {
				t.Errorf("got %d post groups, want none for a feed URL", len(got[0].Groups))
			}
		})
	}
}

// The three tests below cover the page flow: a URL that is neither a feed nor
// a page that links to one. The page is served from the stub server, so it has
// to be fetched over http.
const (
	pageHost = "blog.example.test"
	pageURL  = "http://" + pageHost + "/"
)

// searchPage searches the stubbed page and returns the candidate.
func searchPage(t *testing.T, fixture string) feed.Candidate {
	t.Helper()
	testenv.StubHTTP(pageHost, "/", fixture)
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	got, err := s.SearchFeeds(t.Context(), pageURL)
	if err != nil {
		t.Fatalf("got %v, want a nil error", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d results, want exactly 1", len(got))
	}
	return got[0]
}

// postSet builds the keys a user would send back after ticking the group that
// holds the posts and picking one row for each attribute.
func postSet(t *testing.T, c feed.Candidate) feed.Selectors {
	t.Helper()
	var g feed.Group
	for _, x := range c.Groups {
		if len(x.Posts) > len(g.Posts) {
			g = x
		}
	}
	if len(g.Posts) == 0 {
		t.Fatal("the page produced no group with items")
	}
	set := feed.Selectors{Root: g.Selector, Link: g.Posts[0].Links[0].Selector}
	for _, a := range g.Posts[0].Texts {
		switch {
		case set.Title == "":
			set.Title = a.Selector
		case set.Timestamp == "":
			set.Timestamp = a.Selector
		case set.Description == "":
			set.Description = a.Selector
		}
	}
	if len(g.Posts[0].Images) > 0 {
		set.Image = g.Posts[0].Images[0].Selector
	}
	return set
}

func TestFeed_SearchFeeds_Page(t *testing.T) {
	t.Cleanup(testenv.TearDown)

	t.Run("a page that publishes no feed answers with its groups", func(t *testing.T) {
		c := searchPage(t, "./testdata/feed/page_blog.html")
		if want := "The Example Blog"; c.Title != want {
			t.Errorf("got title %q, want %q", c.Title, want)
		}
		if c.SiteURL == nil || c.SiteURL.String() != pageURL {
			t.Errorf("got site url %v, want %s", c.SiteURL, pageURL)
		}
		if len(c.Groups) == 0 {
			t.Fatal("got no post group, want at least one")
		}
		set := postSet(t, c)
		if !strings.HasSuffix(set.Root, "li") {
			t.Errorf("got the group key %q, want one that ends in the item tag", set.Root)
		}
	})

	t.Run("a page that links to a feed answers with the feed", func(t *testing.T) {
		testenv.StubHTTP("www.nasa.gov", "/news-release/feed/", "./testdata/feed/nasa_news_release.xml")
		c := searchPage(t, "./testdata/feed/page_with_feed_link.html")
		if want := "NASA"; c.Title != want {
			t.Errorf("got title %q, want %q", c.Title, want)
		}
		if want := "http://www.nasa.gov/news-release/feed/"; c.URL.String() != want {
			t.Errorf("got url %q, want %q", c.URL.String(), want)
		}
		if len(c.Groups) != 0 {
			t.Errorf("got %d post groups, want none for a page that links to a feed", len(c.Groups))
		}
	})
}

func TestFeed_Subscribe_Page(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	uid := provisionDefaultTestAccount(t, mustTimeUTC("2026-07-01 13:30:00"))
	c := searchPage(t, "./testdata/feed/page_blog.html")
	set := postSet(t, c)
	svc := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))

	fd, err := svc.Subscribe(t.Context(), uid, *must(url.Parse(pageURL)), []feed.Selectors{set})
	if err != nil {
		t.Fatalf("got %q, want a nil error", err)
	}
	if fd.ID == 0 {
		t.Fatal("feed ID must be assigned")
	}

	var stored feed.Selectors
	scanRowOrFatal(t, `
		SELECT root, link, title, description, image, published_at
		FROM feed_post_selectors WHERE feed_id = $1 ORDER BY id
	`, []any{fd.ID}, &stored.Root, &stored.Link, &stored.Title, &stored.Description, &stored.Image, &stored.Timestamp)
	if d := cmp.Diff(set, stored); d != "" {
		t.Errorf("stored selector record mismatch:\n%s", d)
	}

	// The posts are stored as entries right away, so the timeline answers
	// before the first poll runs.
	got := scanRowsOrFatal(t, `
		SELECT url, title FROM feed_entries WHERE feed_id = $1 ORDER BY url
	`, []any{fd.ID}, func(r *sql.Rows, dest *[2]string) error {
		return r.Scan(&dest[0], &dest[1])
	})
	want := [][2]string{
		{"http://blog.example.test/posts/first", "The first post"},
		{"http://blog.example.test/posts/second", "The second post"},
		{"http://blog.example.test/posts/third", "The third post"},
	}
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("stored entries mismatch:\n%s", d)
	}
}

func TestFeed_Subscribe_PageWithBrokenSelectors(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	uid := provisionDefaultTestAccount(t, mustTimeUTC("2026-07-01 13:30:00"))
	c := searchPage(t, "./testdata/feed/page_blog.html")
	good := postSet(t, c)

	tests := []struct {
		name string
		sets []feed.Selectors
	}{
		{"no selectors at all", nil},
		{"a root that names no group", []feed.Selectors{{Root: "html > body > main > ol > li", Link: good.Link}}},
		{"a link that no item carries", []feed.Selectors{{Root: good.Root, Link: "div > a"}}},
	}
	svc := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Subscribe(t.Context(), uid, *must(url.Parse(pageURL)), tt.sets)
			if !errors.Is(err, feed.ErrSelectors) {
				t.Fatalf("got error %v, want ErrSelectors", err)
			}
			n := scanValOrFatal[int](t, "SELECT count(*) FROM feeds WHERE url = $1", pageURL)
			if n != 0 {
				t.Errorf("the feed was written although the request was refused")
			}
		})
	}
}
