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
func (s *Service) Subscribe(ctx context.Context, uid user.UserID, fu url.URL) (Feed, error) {
	// TODO: Check if the f already exists first to avoid making an unnecessary request
	// TODO: Validate and cleanup the url (check schema, remove tracking params, etc.)
	// TODO: Save fetched entries to DB
	var f Feed
	if a, err := s.fetchFeed(ctx, fu); err != nil {
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
	a, err := s.fetchFeed(ctx, *u)
	if err != nil {
		return nil, err
	}
	return []FeedAttrs{a}, nil
}

func (s *Service) fetchFeed(ctx context.Context, fu url.URL) (FeedAttrs, error) {
	res, err := s.scraper.Fetch(ctx, fu)
	if err != nil {
		return FeedAttrs{}, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		return FeedAttrs{}, fmt.Errorf("failed to fetch feed: %s", fu.String())
	}
	// TODO: Limit body size
	raw, err := gofeed.NewParser().Parse(res.Body)
	if err != nil {
		return FeedAttrs{}, err
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
	return a, nil
}
