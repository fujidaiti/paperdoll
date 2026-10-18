//go:build integration

package itest

import (
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
	"github.com/google/go-cmp/cmp"
)

func TestFeed_SearchFeeds_Page(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("www.nasa.gov", "/news", "./testdata/feed/page_with_feed.html")
	testenv.StubHTTP("www.nasa.gov", "/news-release/feed/", "./testdata/feed/nasa_news_release.xml")
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")

	tests := []struct {
		name, query string
		want        feedValue
	}{
		{
			name:  "page with an alternate feed",
			query: "http://www.nasa.gov/news",
			want: feedValue{
				URL:         "http://www.nasa.gov/news-release/feed/",
				SiteURL:     "https://www.nasa.gov",
				Title:       "NASA",
				Description: "Official National Aeronautics and Space Administration Website",
			},
		},
		{
			name:  "page without a feed",
			query: "http://blog.example.com/",
			want: feedValue{
				URL:     "http://blog.example.com/",
				SiteURL: "http://blog.example.com/",
				Title:   "Example Blog",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
			got, err := s.SearchFeeds(t.Context(), tt.query)
			if err != nil {
				t.Fatalf("got %v, want a nil error", err)
			}
			if len(got) != 1 {
				t.Fatalf("got %d results, want exactly 1", len(got))
			}
			if d := cmp.Diff(tt.want, newFeedValue(feed.Feed{FeedAttrs: got[0]})); d != "" {
				t.Errorf("returned feed mismatch:\n%s", d)
			}
		})
	}
}

func TestFeed_Matchers(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	testenv.StubHTTP("www.nasa.gov", "/news-release/feed/", "./testdata/feed/nasa_news_release.xml")
	uid := provisionDefaultTestAccount(t, time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))

	page, err := s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/")))
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if page.SiteURL == nil || page.SiteURL.String() != page.URL.String() {
		t.Errorf("site URL = %v, want the page URL %s", page.SiteURL, page.URL.String())
	}
	if page.HasMatchers {
		t.Error("HasMatchers = true for a new feed, want false")
	}
	again, err := s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/")))
	if err != nil {
		t.Fatalf("Subscribe again: %v", err)
	}
	if again.ID != page.ID {
		t.Errorf("subscribing again registered feed %d, want %d", again.ID, page.ID)
	}

	doc, matchers, err := s.Matchers(t.Context(), page.ID)
	if err != nil {
		t.Fatalf("Matchers: %v", err)
	}
	if doc.Title != "Example Blog" || doc.Root == nil {
		t.Errorf("page = %+v, want the simplified page", doc)
	}
	if matchers != "" {
		t.Errorf("matchers = %q, want none", matchers)
	}

	raw := `[{"post":["box","box"],"fields":{"title":[{"shape":"h2+link","nth":0}]},"exclude":[]}]`
	if err := s.SaveMatchers(t.Context(), page.ID, raw); err != nil {
		t.Fatalf("SaveMatchers: %v", err)
	}
	if _, matchers, err = s.Matchers(t.Context(), page.ID); err != nil {
		t.Fatalf("Matchers after saving: %v", err)
	}
	var got, want any
	_ = json.Unmarshal([]byte(matchers), &got)
	_ = json.Unmarshal([]byte(raw), &want)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("saved matchers mismatch:\n%s", d)
	}
	again, err = s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/")))
	if err != nil {
		t.Fatalf("Subscribe after saving: %v", err)
	}
	if !again.HasMatchers {
		t.Error("HasMatchers = false after saving, want true")
	}

	for _, invalid := range []string{`[]`, `{}`, `[{"post":[]}]`, `not json`} {
		if err := s.SaveMatchers(t.Context(), page.ID, invalid); !errors.Is(err, feed.ErrInvalidMatchers) {
			t.Errorf("SaveMatchers(%s) = %v, want ErrInvalidMatchers", invalid, err)
		}
	}

	rss, err := s.Subscribe(t.Context(), uid, *must(url.Parse("http://www.nasa.gov/news-release/feed/")))
	if err != nil {
		t.Fatalf("Subscribe to RSS: %v", err)
	}
	if err := s.SaveMatchers(t.Context(), rss.ID, raw); !errors.Is(err, feed.ErrNotUnstructured) {
		t.Errorf("SaveMatchers on an RSS feed = %v, want ErrNotUnstructured", err)
	}
	if _, _, err := s.Matchers(t.Context(), rss.ID); !errors.Is(err, feed.ErrNotUnstructured) {
		t.Errorf("Matchers on an RSS feed = %v, want ErrNotUnstructured", err)
	}
	if err := s.SaveMatchers(t.Context(), 1_000_000, raw); !errors.Is(err, feed.ErrFeedNotFound) {
		t.Errorf("SaveMatchers on a missing feed = %v, want ErrFeedNotFound", err)
	}
}
