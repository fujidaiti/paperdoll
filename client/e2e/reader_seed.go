package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
)

// seedReaderSuit_Article saves a feed entry without server content to the
// reading list. The entry's URL points at a stubbed article page, so the
// reader can only show the article by extracting it from the page itself.
//
// The rule is registered under the address the emulator reaches the host at,
// because the app's WebView loads the page directly.
func seedReaderSuit_Article(ctx context.Context, db *sql.DB) error {
	token, err := provisionTestAccount(ctx, db)
	if err != nil {
		return err
	}

	svc := &user.Service{DB: db, Now: time.Now}
	uid, err := svc.VerifyAuthToken(ctx, token.Encode())
	if err != nil {
		return err
	}

	var feedID int
	err = db.QueryRowContext(ctx, `
		INSERT INTO feeds (url, site_url, icon_url, title, description)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url
		RETURNING id;
	`, "http://10.0.2.2:8081/feed", "http://10.0.2.2:8081", nil, "Coastal Notes",
		"Notes from the coast.").Scan(&feedID)
	if err != nil {
		return err
	}

	var entryID int
	err = db.QueryRowContext(ctx, `
		INSERT INTO feed_entries (dedup_key, feed_id, url, title, description, snapshot_at, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "http://10.0.2.2:8081/articles/tide-pools", feedID, "http://10.0.2.2:8081/articles/tide-pools",
		"How Tide Pools Survive the Summer", "What lives in a pool that dries out.",
		mustTimeUTC("2026-06-30 12:34:30"), mustTimeUTC("2026-07-01 15:30:12")).Scan(&entryID)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO reading_list_items (kind, feed_entry_id, title, description, saved_at, user_id)
		VALUES ('feed_entry', $1, $2, $3, $4, $5)
	`, entryID, "How Tide Pools Survive the Summer", "What lives in a pool that dries out.",
		mustTimeUTC("2026-07-01 15:31:00"), uid)
	if err != nil {
		return err
	}

	testenv.StubHTTP("10.0.2.2:8081", "/articles/tide-pools", "./testdata/reader_article.html")
	return nil
}
