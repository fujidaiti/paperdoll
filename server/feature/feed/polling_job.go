package feed

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/newspaper"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mmcdole/gofeed"
)

// FeedRecord is the minimal reference to a feeds row a [Job] needs to poll it.
type FeedRecord struct {
	ID    int
	URL   string
	Title string
}

type entryRecord struct {
	id          int
	dedupKey    string
	feedId      int
	url         string
	title       string
	description sql.NullString
	publishedAt sql.NullTime
}

// Job polls a single feed for new entries, and downloads the content of the
// new entries published within Interval.
type Job struct {
	DB       *sql.DB
	Feed     FeedRecord
	Interval newspaper.EditorialInterval
	ScrpSvc  *scraper.Service
}

func (j *Job) Timeout() time.Duration {
	return 30 * time.Second
}

func (j *Job) Do(ctx context.Context) error {
	feed, db := j.Feed, j.DB
	fmt.Println("Fetching feed: ", feed.URL)

	link, err := url.Parse(feed.URL)
	if err != nil {
		return err
	}
	res, err := j.ScrpSvc.Fetch(ctx, *link)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("couldn't fetch feed (id=%d)", feed.ID)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	fp := gofeed.NewParser()
	raw, err := fp.Parse(res.Body)
	if err != nil {
		return err
	}
	if len(raw.Items) == 0 {
		return fmt.Errorf("feed %d has no items", feed.ID)
	}

	fmt.Printf("Got %d entries from %s\n", len(raw.Items), feed.URL)

	newEntries, err := saveEntries(ctx, db, feed.ID, raw.Items)
	if err != nil {
		return err
	}
	if len(newEntries) == 0 {
		fmt.Printf("No new entry from %s, skipping.\n", feed.URL)
		return nil
	}

	// TODO(#129): Mark the entry as queued in DB to avoid duplicate jobs for the same entry
	for _, e := range newEntries {
		if e.publishedAt.Valid && j.Interval.Contains(e.publishedAt.Time) {
			fmt.Printf("Fetching content for %s\n", e.url)
			st := time.Now()
			err := fetchContent(ctx, j.ScrpSvc, e, db)
			if err != nil {
				fmt.Print(err)
			}
			// TODO(#13): Respect Crawl-delay in robot.txt
			if dt := time.Since(st); dt < time.Second {
				time.Sleep(time.Second - dt)
			}
		}
	}

	return nil
}

func CollectJobs(
	ctx context.Context,
	db *sql.DB,
	scrpSvc *scraper.Service,
	now time.Time,
) ([]Job, error) {
	ei, err := newspaper.FindEditorialInterval(ctx, db, now)
	if err != nil {
		fmt.Print(err)
		// Fail-soft: we don't exit here.
	}
	// TODO(#110): Create an index for next_poll_at column
	rows, err := db.QueryContext(ctx, `
		SELECT id, url, title
		FROM feeds;
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	var jobs []Job
	for rows.Next() {
		feed := FeedRecord{}
		err := rows.Scan(&feed.ID, &feed.URL, &feed.Title)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, Job{db, feed, ei, scrpSvc})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

// saveEntries stores the feed items as entries of the feed and returns only
// the ones that were not stored yet. Items that are already stored, identified
// by their dedup key, are left unchanged.
func saveEntries(ctx context.Context, db *sql.DB, feedID int, items []*gofeed.Item) ([]entryRecord, error) {
	if len(items) == 0 {
		return nil, nil
	}

	// TODO(#128): Batch insertions if the feed is too large
	ncols := 7
	vals := make([]string, 0, len(items))
	args := make([]any, 0, len(items)*ncols)
	snapshotAt := time.Now()
	for i, item := range items {
		j := i * ncols
		vals = append(
			vals,
			fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", j+1, j+2, j+3, j+4, j+5, j+6, j+7),
		)
		e := normalizeEntry(item, feedID)
		args = append(
			args, e.dedupKey, e.feedId, e.url, e.title, e.description, snapshotAt, e.publishedAt,
		)
	}
	sql := fmt.Sprintf(`
		INSERT INTO feed_entries (dedup_key, feed_id, url, title, description, snapshot_at, published_at)
		VALUES %s
		ON CONFLICT (dedup_key) DO NOTHING
		RETURNING id, dedup_key, feed_id, url, title, description, published_at;
	`, strings.Join(vals, ","))
	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	var newEntries []entryRecord
	for rows.Next() {
		e := entryRecord{}
		err := rows.Scan(
			&e.id,
			&e.dedupKey,
			&e.feedId,
			&e.url,
			&e.title,
			&e.description,
			&e.publishedAt,
		)
		if err != nil {
			// TODO(#127): Make this case fail-soft instead of exiting.
			return nil, err
		}
		newEntries = append(newEntries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return newEntries, nil
}

func normalizeEntry(entry *gofeed.Item, feedId int) entryRecord {
	e := entryRecord{
		feedId: feedId,
		// TODO(#126): Cleanup the link
		url:   entry.Link,
		title: entry.Title,
	}

	// TODO(#126): Sanitize the description
	if desc := entry.Description; len(desc) > 0 {
		e.description.String = desc
		e.description.Valid = true
	}
	if date := entry.PublishedParsed; date != nil {
		e.publishedAt.Time = *date
		e.publishedAt.Valid = true
	}
	if guid := entry.GUID; len(guid) > 0 {
		e.dedupKey = guid
	} else {
		// TODO(#126): Synthesize a guid for this entry
		e.dedupKey = entry.Link
	}

	return e
}

func fetchContent(ctx context.Context, scrp *scraper.Service, entry entryRecord, db *sql.DB) error {
	entryUrl, err := url.Parse(entry.url)
	if err != nil {
		return err
	}

	res, err := scrp.Fetch(ctx, *entryUrl)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP GET failed with status code %d", res.StatusCode)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// The entry's title and description come from the feed, so the page's
	// metadata is ignored.
	page, err := scraper.Extract(res.Body, *entryUrl)
	if err != nil {
		return err
	}
	if page.Content == "" {
		return fmt.Errorf("failed to extract content")
	}

	_, err = db.ExecContext(ctx, `
		UPDATE feed_entries
		SET content = $1, snapshot_at = $2
		WHERE id = $3;
	`, page.Content, time.Now(), entry.id)
	if err != nil {
		return fmt.Errorf("failed to fetch content")
	}

	return nil
}
