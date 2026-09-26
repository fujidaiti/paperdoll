package feed

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

type Feed struct {
	ID int
	FeedAttrs
}

type FeedAttrs struct {
	URL         url.URL
	SiteURL     *url.URL
	IconURL     *url.URL
	Title       string
	Description *string
}

// Candidate is one result of a search: a feed the user can subscribe to,
// together with the post groups of the page when the URL turned out to be an
// HTML page that publishes no feed. Groups is empty for a real feed, and the
// client then subscribes with the URL alone.
type Candidate struct {
	FeedAttrs
	Groups []Group
}

// Subscribe creates a new subscription to a web feed for the user, from its URL.
// Feeds are identified by the URL and this operation is idempotent; subscribing
// to the same feed (URL) twice has no additional effect.
//
// sets is empty for an RSS/Atom feed and for an HTML page that links to one. For
// a page that publishes no feed it holds the keys the user picked on the search
// screen, one set per group they ticked. The page is read once before anything
// is saved, and a set that produces no post fails with ErrSelectors, because
// such a subscription would stay empty forever.
func (s *Service) Subscribe(ctx context.Context, uid user.UserID, fu url.URL, sets []Selectors) (Feed, error) {
	// TODO: Check if the f already exists first to avoid making an unnecessary request
	// TODO: Validate and cleanup the url (check schema, remove tracking params, etc.)
	src, err := s.discover(ctx, fu)
	if err != nil {
		fmt.Println(err)
		return Feed{}, err
	}

	var posts []Post
	if src.feed != nil {
		// The URL is a feed, or a page that links to one, so the selectors
		// describe something the server no longer needs and are ignored.
		sets = nil
	} else {
		if len(sets) == 0 {
			return Feed{}, fmt.Errorf("%w: %s publishes no feed and no selectors were sent", ErrSelectors, fu.String())
		}
		posts, err = ExtractPosts(bytes.NewReader(src.body), src.url, sets)
		if err != nil {
			return Feed{}, err
		}
	}

	f := Feed{FeedAttrs: src.attrs()}
	var su, iu, desc sql.NullString
	if u := f.IconURL; u != nil {
		iu = sql.NullString{String: u.String(), Valid: true}
	}
	if u := f.SiteURL; u != nil {
		su = sql.NullString{String: u.String(), Valid: true}
	}
	if d := f.Description; d != nil {
		desc = sql.NullString{String: *d, Valid: true}
	}

	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO feeds (url, site_url, icon_url, title, description)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url
		RETURNING id;
	`, f.URL.String(), su, iu, f.Title, desc).Scan(&f.ID)
	if err != nil {
		return Feed{}, err
	}
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO feed_subscriptions (user_id, feed_id) VALUES ($1, $2)
		ON CONFLICT (user_id, feed_id) DO NOTHING
	`, uid, f.ID)
	if err != nil {
		return Feed{}, err
	}
	if len(sets) == 0 {
		return f, nil
	}

	// The selectors belong to the feed, not to the subscription, because the
	// entries are stored per feed. A second user who subscribes to the same
	// page with different keys gets the feed as the first user described it;
	// their own keys were validated above and are then dropped.
	if err := saveSelectors(ctx, s.DB, f.ID, sets); err != nil {
		return Feed{}, err
	}
	// The posts are stored right away, so that the timeline answers without
	// waiting for the first poll.
	if _, err := insertEntries(ctx, s.DB, entriesFromPosts(f.ID, posts, time.Now())); err != nil {
		return Feed{}, err
	}
	return f, nil
}

// SearchFeeds searches subscriptable feeds by the given query.
func (s *Service) SearchFeeds(ctx context.Context, query string) ([]Candidate, error) {
	// TODO: Accept arbitrary keywards as a query
	// TODO: Validate and cleanup the url (check schema, remove tracking params, etc.)
	u, err := url.Parse(query)
	if err != nil {
		return []Candidate{}, nil
	}
	src, err := s.discover(ctx, *u)
	if err != nil {
		return nil, err
	}
	c := Candidate{FeedAttrs: src.attrs()}
	if src.feed == nil {
		c.Groups, err = EnumeratePostGroups(bytes.NewReader(src.body), src.url)
		if err != nil {
			return nil, err
		}
	}
	return []Candidate{c}, nil
}

// source is what a URL turned out to be: a feed, or an HTML page that
// publishes none.
type source struct {
	// url is where the body was fetched from. It is the requested URL, except
	// when the page linked to a feed, in which case it is the feed's own URL.
	url url.URL
	// feed is the parsed feed, and body the HTML page. Exactly one of the two
	// is set.
	feed *gofeed.Feed
	body []byte
}

// discover fetches a URL and decides what it is, which is the same decision at
// search time and at subscribe time, so that the two requests agree.
//
//  1. Fetch the URL.
//  2. If it parses as a feed, it is a feed.
//  3. Otherwise look for a feed in the head of the page and fetch that.
//  4. Otherwise it is a plain page.
func (s *Service) discover(ctx context.Context, fu url.URL) (source, error) {
	body, err := s.fetch(ctx, fu)
	if err != nil {
		return source{}, err
	}
	if f, err := gofeed.NewParser().Parse(bytes.NewReader(body)); err == nil {
		return source{url: fu, feed: f}, nil
	}
	if href, ok := feedLink(body, fu); ok {
		if b, err := s.fetch(ctx, href); err == nil {
			if f, err := gofeed.NewParser().Parse(bytes.NewReader(b)); err == nil {
				return source{url: href, feed: f}, nil
			}
		}
		// A head that names a feed the server cannot read is not an error:
		// the page itself is still there to enumerate.
	}
	return source{url: fu, body: body}, nil
}

func (s *Service) fetch(ctx context.Context, fu url.URL) ([]byte, error) {
	res, err := s.scraper.Fetch(ctx, fu)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: status %d", fu.String(), res.StatusCode)
	}
	// TODO: Limit body size
	return io.ReadAll(res.Body)
}

// feedLink returns the feed a page links to in its head, resolved against the
// page URL:
//
//	<link rel="alternate" type="application/rss+xml" href="/feed.xml">
func feedLink(body []byte, page url.URL) (url.URL, bool) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return url.URL{}, false
	}
	var found *url.URL
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "link" &&
			strings.EqualFold(attr(n, "rel"), "alternate") {
			t := strings.ToLower(attr(n, "type"))
			if t == "application/rss+xml" || t == "application/atom+xml" {
				href := attr(n, "href")
				// Some pages write the feed:// scheme, which no HTTP client
				// can fetch. developer.apple.com does this.
				if after, ok := strings.CutPrefix(href, "feed://"); ok {
					href = "https://" + after
				}
				if u, err := page.Parse(href); err == nil {
					found = u
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if found == nil {
		return url.URL{}, false
	}
	return *found, true
}

// attrs describes the feed a subscription would be made to. For a real feed
// the attributes come from the feed document. For a plain page there is no
// such document, so the page is the feed: its title is the <title> of the
// document and its site is the page itself.
func (s source) attrs() FeedAttrs {
	if s.feed == nil {
		site := s.url
		return FeedAttrs{URL: s.url, SiteURL: &site, Title: pageTitle(s.body)}
	}
	raw := s.feed
	a := FeedAttrs{URL: s.url, Title: raw.Title}
	if raw.Link != "" {
		if u, err := url.Parse(raw.Link); err == nil {
			a.SiteURL = u
		}
	}
	if raw.Image != nil && raw.Image.URL != "" {
		if u, err := url.Parse(raw.Image.URL); err == nil {
			a.IconURL = u
		}
	}
	if raw.Description != "" {
		a.Description = &raw.Description
	}
	return a
}

func pageTitle(body []byte) string {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return ""
	}
	var title string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if title != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "title" {
			title = strings.TrimSpace(text(n))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return title
}

// saveSelectors writes the key sets of a page feed. They are read back with
// ORDER BY id, which is the order the request listed them in, and that order
// decides which set wins when two of them produce the same post URL.
func saveSelectors(ctx context.Context, db *sql.DB, feedID int, sets []Selectors) error {
	for _, s := range sets {
		_, err := db.ExecContext(ctx, `
			INSERT INTO feed_post_selectors (feed_id, root, link, title, description, image, published_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7);
		`, feedID, s.Root, s.Link,
			nullString(s.Title), nullString(s.Description),
			nullString(s.Image), nullString(s.Timestamp))
		if err != nil {
			return err
		}
	}
	return nil
}

// loadSelectors returns the key sets of a feed. An empty result means the feed
// is an RSS/Atom feed rather than a page feed.
func loadSelectors(ctx context.Context, db *sql.DB, feedID int) ([]Selectors, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT root, link, title, description, image, published_at
		FROM feed_post_selectors
		WHERE feed_id = $1
		ORDER BY id;
	`, feedID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	var out []Selectors
	for rows.Next() {
		var s Selectors
		var title, desc, image, published sql.NullString
		if err := rows.Scan(&s.Root, &s.Link, &title, &desc, &image, &published); err != nil {
			return nil, err
		}
		s.Title, s.Description = title.String, desc.String
		s.Image, s.Timestamp = image.String, published.String
		out = append(out, s)
	}
	return out, rows.Err()
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
