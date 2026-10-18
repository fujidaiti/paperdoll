package feed

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

type Feed struct {
	ID int
	FeedAttrs
	// HasMatchers reports whether the post matchers of an unstructured feed,
	// a web page without an RSS feed whose URL and site URL are the page's URL,
	// have been saved.
	HasMatchers bool
}

type FeedAttrs struct {
	URL         url.URL
	SiteURL     *url.URL
	IconURL     *url.URL
	Title       string
	Description *string
}

// Subscribe creates a new subscription to a web feed for the user, from its URL
// or the URL of a page (see discover). Feeds are identified by the URL and this
// operation is idempotent; subscribing to the same feed (URL) twice has no
// additional effect.
func (s *Service) Subscribe(ctx context.Context, uid user.UserID, fu url.URL) (Feed, error) {
	// TODO: Check if the f already exists first to avoid making an unnecessary request
	// TODO: Validate and cleanup the url (check schema, remove tracking params, etc.)
	// TODO: Save fetched entries to DB
	var f Feed
	if a, err := s.discover(ctx, fu); err != nil {
		fmt.Println(err)
		return Feed{}, err
	} else {
		f = Feed{FeedAttrs: a}
	}
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

	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO feeds (url, site_url, icon_url, title, description)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url
		RETURNING id, matchers IS NOT NULL;
	`, f.URL.String(), su, iu, f.Title, desc).Scan(&f.ID, &f.HasMatchers)
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
	return f, nil
}

// Unsubscribe drops the user's subscription to the feed. The feeds row is
// shared across all users, so only the join row is deleted and the feed itself
// stays intact. Reports false with no error if the user was not subscribed,
// true otherwise.
func (s *Service) Unsubscribe(ctx context.Context, uid user.UserID, feedID int) (bool, error) {
	res, err := s.DB.ExecContext(ctx, `
		DELETE FROM feed_subscriptions
		WHERE user_id = $1 AND feed_id = $2;
	`, uid, feedID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n != 0, nil
}

// SearchFeeds searches subscriptable feeds by the given query.
func (s *Service) SearchFeeds(ctx context.Context, query string) ([]FeedAttrs, error) {
	// TODO: Accept arbitrary keywards as a query
	// TODO: Validate and cleanup the url (check schema, remove tracking params, etc.)
	u, err := url.Parse(query)
	if err != nil {
		return []FeedAttrs{}, nil
	}
	a, err := s.discover(ctx, *u)
	if err != nil {
		return nil, err
	}
	return []FeedAttrs{a}, nil
}

// discover returns the feed at u. When u is an HTML page, the feed is the
// first RSS, Atom or JSON feed that the page links as an alternate in its
// head, or else the first feed that the page links from its body at one of
// rootFeedPaths, or else the page itself as an unstructured feed.
func (s *Service) discover(ctx context.Context, u url.URL) (FeedAttrs, error) {
	raw, pageURL, err := s.fetch(ctx, u)
	if err != nil {
		return FeedAttrs{}, err
	}
	if f, err := gofeed.NewParser().Parse(bytes.NewReader(raw)); err == nil {
		return feedAttrs(u, f), nil
	}
	root, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return FeedAttrs{}, err
	}
	var head *html.Node
	for n := range root.Descendants() {
		if n.Type == html.ElementNode && n.Data == "head" {
			head = n
			break
		}
	}
	var title string
	var alternate *url.URL
	if head != nil {
		for n := range head.Descendants() {
			if n.Type != html.ElementNode {
				continue
			}
			switch n.Data {
			case "title":
				if title == "" && n.FirstChild != nil {
					title = normalizeSpace(n.FirstChild.Data)
				}
			case "link":
				rels := strings.Fields(strings.ToLower(attr(n, "rel")))
				if alternate != nil || !slices.Contains(rels, "alternate") {
					continue
				}
				switch strings.ToLower(attr(n, "type")) {
				case "application/rss+xml", "application/atom+xml", "application/feed+json":
					if h, err := pageURL.Parse(attr(n, "href")); err == nil {
						alternate = h
					}
				}
			}
		}
	}
	if alternate != nil {
		raw, _, err := s.fetch(ctx, *alternate)
		if err != nil {
			return FeedAttrs{}, err
		}
		f, err := gofeed.NewParser().Parse(bytes.NewReader(raw))
		if err != nil {
			return FeedAttrs{}, err
		}
		return feedAttrs(*alternate, f), nil
	}
	// Some pages link their feed only from the body, such as a feed icon in
	// the footer. Only a link to one of the usual feed paths at the root of
	// the same host is taken, so that a feed of another site or a comment
	// feed is not taken for the page's feed.
	seen := map[string]bool{}
	for n := range root.Descendants() {
		if n.Type != html.ElementNode || n.Data != "a" {
			continue
		}
		h, err := pageURL.Parse(attr(n, "href"))
		if err != nil || h.Host != pageURL.Host ||
			!slices.Contains(rootFeedPaths, strings.TrimSuffix(h.Path, "/")) ||
			seen[h.String()] {
			continue
		}
		seen[h.String()] = true
		raw, _, err := s.fetch(ctx, *h)
		if err != nil {
			continue
		}
		if f, err := gofeed.NewParser().Parse(bytes.NewReader(raw)); err == nil {
			return feedAttrs(*h, f), nil
		}
	}
	if title == "" {
		title = pageURL.Host
	}
	site := *pageURL
	return FeedAttrs{URL: *pageURL, SiteURL: &site, Title: title}, nil
}

// rootFeedPaths are the paths at which sites commonly serve their feed, such
// as /feed of WordPress, /rss of Ghost and /index.xml of Hugo.
var rootFeedPaths = []string{
	"/feed", "/rss", "/atom", "/feed.xml", "/rss.xml", "/atom.xml", "/index.xml",
}

// fetch returns the body of the resource at u and its URL after redirects.
func (s *Service) fetch(ctx context.Context, u url.URL) ([]byte, *url.URL, error) {
	res, err := s.scraper.Fetch(ctx, u)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("failed to fetch %s: %s", u.String(), res.Status)
	}
	// TODO: Limit body size
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	return raw, res.Request.URL, nil
}

func feedAttrs(u url.URL, raw *gofeed.Feed) FeedAttrs {
	a := FeedAttrs{URL: u, Title: raw.Title}
	// A feed whose site link is the feed itself is given no site URL, since a
	// site URL equal to the URL marks an unstructured feed.
	if raw.Link != "" && raw.Link != u.String() {
		if su, err := url.Parse(raw.Link); err == nil {
			a.SiteURL = su
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

var (
	ErrFeedNotFound    = errors.New("feed not found")
	ErrNotUnstructured = errors.New("feed is not unstructured")
	ErrInvalidMatchers = errors.New("invalid matchers")
)

// Matchers returns the semantic tree of the page of the unstructured feed id,
// on which its post matchers are edited, and the matchers saved for it, which
// are empty when none are saved.
func (s *Service) Matchers(ctx context.Context, id int) (*SemanticTree, string, error) {
	var fu string
	var matchers sql.NullString
	var unstructured bool
	err := s.DB.QueryRowContext(ctx, `
		SELECT url, matchers, (url = site_url) IS TRUE FROM feeds WHERE id = $1;
	`, id).Scan(&fu, &matchers, &unstructured)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrFeedNotFound
	} else if err != nil {
		return nil, "", err
	}
	if !unstructured {
		return nil, "", ErrNotUnstructured
	}
	u, err := url.Parse(fu)
	if err != nil {
		return nil, "", err
	}
	raw, _, err := s.fetch(ctx, *u)
	if err != nil {
		return nil, "", err
	}
	tree, err := BuildSemanticTree(raw, fu)
	if err != nil {
		return nil, "", err
	}
	return tree, matchers.String, nil
}

// SaveMatchers replaces the post matchers of the unstructured feed id with
// raw, a JSON array of PostMatcher. The matchers are shared by all
// subscribers, as the feed is, so any user can replace them.
func (s *Service) SaveMatchers(ctx context.Context, id int, raw string) error {
	var ms []PostMatcher
	if err := json.Unmarshal([]byte(raw), &ms); err != nil || len(ms) == 0 {
		return ErrInvalidMatchers
	}
	for _, m := range ms {
		if len(m.Post) == 0 {
			return ErrInvalidMatchers
		}
	}
	res, err := s.DB.ExecContext(ctx, `
		UPDATE feeds SET matchers = $1 WHERE id = $2 AND url = site_url;
	`, raw, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n > 0 {
		return nil
	}
	var exists bool
	err = s.DB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM feeds WHERE id = $1);`, id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrFeedNotFound
	}
	return ErrNotUnstructured
}
