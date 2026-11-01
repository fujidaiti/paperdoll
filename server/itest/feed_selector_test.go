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
	testenv.StubHTTP("engineering.example.com", "/", "./testdata/feed/page_with_body_feed.html")
	testenv.StubHTTP("engineering.example.com", "/feed", "./testdata/feed/engineering_feed.xml")
	testenv.StubHTTP("other.example.com", "/feed", "./testdata/feed/nasa_news_release.xml")
	testenv.StubHTTP("engineering.example.com", "/posts/first/feed", "./testdata/feed/nasa_news_release.xml")

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
			name:  "page linking a feed from its body",
			query: "http://engineering.example.com/",
			want: feedValue{
				URL:         "http://engineering.example.com/feed",
				SiteURL:     "https://engineering.example.com/",
				Title:       "Example Engineering",
				Description: "Posts of Example Engineering",
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

func TestFeed_Selectors(t *testing.T) {
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
	if page.HasSelectors {
		t.Error("HasSelectors = true for a new feed, want false")
	}
	again, err := s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/")))
	if err != nil {
		t.Fatalf("Subscribe again: %v", err)
	}
	if again.ID != page.ID {
		t.Errorf("subscribing again registered feed %d, want %d", again.ID, page.ID)
	}

	tree, selectors, err := s.Selectors(t.Context(), page.ID)
	if err != nil {
		t.Fatalf("Selectors: %v", err)
	}
	if tree.Title != "Example Blog" || tree.Root == nil {
		t.Errorf("page = %+v, want the semantic tree", tree)
	}
	if selectors != "" {
		t.Errorf("selectors = %q, want none", selectors)
	}

	raw := `[{"post":[{"tag":"article","cls":["post"]}],"fields":{"title":[{"tag":"h2"}],"link":[{"tag":"a"}]}}]`
	if err := s.SaveSelectors(t.Context(), page.ID, raw); err != nil {
		t.Fatalf("SaveSelectors: %v", err)
	}
	if _, selectors, err = s.Selectors(t.Context(), page.ID); err != nil {
		t.Fatalf("Selectors after saving: %v", err)
	}
	var got, want any
	_ = json.Unmarshal([]byte(selectors), &got)
	_ = json.Unmarshal([]byte(raw), &want)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("saved selectors mismatch:\n%s", d)
	}
	again, err = s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/")))
	if err != nil {
		t.Fatalf("Subscribe after saving: %v", err)
	}
	if !again.HasSelectors {
		t.Error("HasSelectors = false after saving, want true")
	}

	for _, invalid := range []string{
		`[]`,
		`{}`,
		`[{"post":[]}]`,
		`not json`,
		`[{"post":["box","box"],"fields":{"title":[{"shape":"h2+link","nth":0}]},"exclude":[]}]`,
	} {
		if err := s.SaveSelectors(t.Context(), page.ID, invalid); !errors.Is(err, feed.ErrInvalidSelectors) {
			t.Errorf("SaveSelectors(%s) = %v, want ErrInvalidSelectors", invalid, err)
		}
	}

	rss, err := s.Subscribe(t.Context(), uid, *must(url.Parse("http://www.nasa.gov/news-release/feed/")))
	if err != nil {
		t.Fatalf("Subscribe to RSS: %v", err)
	}
	if err := s.SaveSelectors(t.Context(), rss.ID, raw); !errors.Is(err, feed.ErrNotUnstructured) {
		t.Errorf("SaveSelectors on an RSS feed = %v, want ErrNotUnstructured", err)
	}
	if _, _, err := s.Selectors(t.Context(), rss.ID); !errors.Is(err, feed.ErrNotUnstructured) {
		t.Errorf("Selectors on an RSS feed = %v, want ErrNotUnstructured", err)
	}
	if err := s.SaveSelectors(t.Context(), 1_000_000, raw); !errors.Is(err, feed.ErrFeedNotFound) {
		t.Errorf("SaveSelectors on a missing feed = %v, want ErrFeedNotFound", err)
	}
}
