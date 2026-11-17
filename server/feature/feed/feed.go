package feed

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/mmcdole/gofeed"
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

// Subscribe creates a new subscription to a web feed for the user, from its URL.
// Feeds are identified by the URL and this operation is idempotent; subscribing
// to the same feed (URL) twice has no additional effect.
//
// When the feed is new to the server, its current entries are stored too, so
// that they are available before the first polling run.
func (s *Service) Subscribe(ctx context.Context, uid user.UserID, fu url.URL) (Feed, error) {
	// TODO(#123): Check if the f already exists first to avoid making an unnecessary request
	// TODO(#115): Validate and cleanup the url (check schema, remove tracking params, etc.)
	var f Feed
	var items []*gofeed.Item
	if a, is, err := s.fetchFeed(ctx, fu); err != nil {
		fmt.Println(err)
		return Feed{}, err
	} else {
		f, items = Feed{FeedAttrs: a}, is
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

	// xmax is 0 only for a row inserted by this statement, not for an existing
	// row touched by the ON CONFLICT update.
	var created bool
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO feeds (url, site_url, icon_url, title, description)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url
		RETURNING id, xmax = 0;
	`, f.URL.String(), su, iu, f.Title, desc).Scan(&f.ID, &created)
	if err != nil {
		return Feed{}, err
	}
	// Entries of an existing feed are left to the polling job. If they were
	// stored here, the polling job would no longer see them as new, and would
	// not draft stories from them for the existing subscribers.
	if created {
		if _, err := saveEntries(ctx, s.DB, f.ID, items); err != nil {
			return Feed{}, err
		}
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
	// TODO(#125): Accept arbitrary keywards as a query
	// TODO(#115): Validate and cleanup the url (check schema, remove tracking params, etc.)
	u, err := url.Parse(query)
	if err != nil {
		return []FeedAttrs{}, nil
	}
	a, _, err := s.fetchFeed(ctx, *u)
	if err != nil {
		return nil, err
	}
	return []FeedAttrs{a}, nil
}

// fetchFeed downloads and parses the feed, returning its attributes and its
// current items.
func (s *Service) fetchFeed(ctx context.Context, fu url.URL) (FeedAttrs, []*gofeed.Item, error) {
	res, err := s.scraper.Fetch(ctx, fu)
	if err != nil {
		return FeedAttrs{}, nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		return FeedAttrs{}, nil, fmt.Errorf("failed to fetch feed: %s", fu.String())
	}
	// TODO(#113): Limit body size
	raw, err := gofeed.NewParser().Parse(res.Body)
	if err != nil {
		return FeedAttrs{}, nil, err
	}

	a := FeedAttrs{URL: fu, Title: raw.Title}
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
	return a, raw.Items, nil
}
