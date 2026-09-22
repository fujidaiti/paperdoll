package main

import (
	"context"
	"database/sql"
	"net/url"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
)

var stubServerAddr = must(url.Parse("http://127.0.0.1:8081"))

// seedFeedSuit_BbcNews subscribes the test account to BBC News through the
// real [feed.Service.Subscribe] (fetching the fixture via the stub HTTP
// server, same as the API server does) rather than inserting feeds/
// feed_subscriptions rows directly, so the seeded state matches what a real
// subscription flow produces. A single entry is then inserted directly since
// there's no public service method for entries outside the background
// polling job.
func seedFeedSuit_BbcNews(ctx context.Context, db *sql.DB) error {
	if _, err := provisionTestAccount(ctx, db); err != nil {
		return err
	}

	var uid int
	if err := db.QueryRowContext(ctx, `
		SELECT id FROM users WHERE email = $1
	`, testAccountEmail).Scan(&uid); err != nil {
		return err
	}

	testenv.StubHTTP("feeds.bbci.co.uk", "/news/rss.xml", "./testdata/bbc_news_rss.xml")

	svc := feed.NewService(db, scraper.NewService(stubServerAddr))
	f, err := svc.Subscribe(ctx, uid, *must(url.Parse("http://feeds.bbci.co.uk/news/rss.xml")), nil)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO feed_entries (dedup_key, feed_id, url, title, description, content, snapshot_at, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "https://www.bbc.co.uk/news/articles/cj03r59z73po",
		f.ID,
		"https://www.bbc.co.uk/news/articles/cj03r59z73po",
		"US signs landmark nuclear deal with Saudi Arabia",
		nil,
		`<article><p>The US Department of Energy says the "peaceful" co-operation agreement `+
			`will give US firms great access to the Saudi nuclear energy programme.</p></article>`,
		mustTimeUTC("2026-07-23 00:00:00"),
		mustTimeUTC("2026-07-23 00:00:00"),
	)
	return err
}

// seedFeedSuit_NasaCandidate only provisions the test account and stubs the
// NASA feed document; the actual subscribe happens through the live UI-driven
// search-and-subscribe flow during the test, via the session's own API
// server (wired to the same stub server as its outbound HTTP proxy).
func seedFeedSuit_NasaCandidate(ctx context.Context, db *sql.DB) error {
	if _, err := provisionTestAccount(ctx, db); err != nil {
		return err
	}
	testenv.StubHTTP("www.nasa.gov", "/news-release/feed/", "./testdata/nasa_news_release.xml")
	return nil
}
