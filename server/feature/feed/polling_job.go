package feed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"codeberg.org/readeck/go-readability/v2"
	"github.com/fujidaiti/paperdoll/server/feature/newspaper"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/feature/user"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/microcosm-cc/bluemonday"
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

// Job polls a single feed for new entries and drafts stories for its subscribers.
type Job struct {
	DB           *sql.DB
	Feed         FeedRecord
	Interval     newspaper.EditorialInterval
	NewspaperSvc *newspaper.Service
	ScrpSvc      *scraper.Service
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

	// TODO: For an unstructured feed, extract the entries from the page with
	//       its matchers instead of parsing it as a feed (see CollectJobs).
	fp := gofeed.NewParser()
	raw, err := fp.Parse(res.Body)
	if err != nil {
		return err
	}
	if len(raw.Items) == 0 {
		return fmt.Errorf("feed %d has no items", feed.ID)
	}

	fmt.Printf("Got %d entries from %s\n", len(raw.Items), feed.URL)

	// TODO: Batch insertions if the feed is too large
	ncols := 7
	vals := make([]string, 0, len(raw.Items))
	args := make([]any, 0, len(raw.Items)*ncols)
	snapshotAt := time.Now()
	for i, item := range raw.Items {
		j := i * ncols
		vals = append(
			vals,
			fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", j+1, j+2, j+3, j+4, j+5, j+6, j+7),
		)
		e := normalizeEntry(item, feed.ID)
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
		return err
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
			// TODO: Make this case fail-soft instead of exiting.
			return err
		}
		newEntries = append(newEntries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(newEntries) == 0 {
		fmt.Printf("No new entry from %s, skipping.\n", feed.URL)
		return nil
	}

	var entriesToBeStories []entryRecord
	// TODO: Mark the entry as queued in DB to avoid duplicate jobs for the same entry
	for _, e := range newEntries {
		if e.publishedAt.Valid && j.Interval.Contains(e.publishedAt.Time) {
			entriesToBeStories = append(entriesToBeStories, e)
			fmt.Printf("Fetching content for %s\n", e.url)
			st := time.Now()
			err := fetchContent(ctx, j.ScrpSvc, e, db)
			if err != nil {
				fmt.Print(err)
			}
			// TODO: Respect Crawl-delay in robot.txt
			if dt := time.Since(st); dt < time.Second {
				time.Sleep(time.Second - dt)
			}
		}
	}

	err = writeStories(ctx, j.DB, j.NewspaperSvc, feed, entriesToBeStories)
	if err != nil {
		fmt.Printf("Something went wrong while writing stories: %s\n", err)
	}

	return nil
}

func CollectJobs(
	ctx context.Context,
	db *sql.DB,
	newspaperSvc *newspaper.Service,
	scrpSvc *scraper.Service,
	now time.Time,
) ([]Job, error) {
	ei, err := newspaper.FindEditorialInterval(ctx, db, now)
	if err != nil {
		fmt.Print(err)
		// Fail-soft: we don't exit here.
	}
	// TODO: Create an index for next_poll_at column
	// TODO: Poll unstructured feeds (url = site_url) that have matchers: fetch
	//       the page, run SimplifyPage and ExtractPosts, and save the posts as
	//       entries. They are skipped for now, as their pages are not feeds.
	rows, err := db.QueryContext(ctx, `
		SELECT id, url, title
		FROM feeds
		WHERE site_url IS DISTINCT FROM url;
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
		jobs = append(jobs, Job{db, feed, ei, newspaperSvc, scrpSvc})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func normalizeEntry(entry *gofeed.Item, feedId int) entryRecord {
	e := entryRecord{
		feedId: feedId,
		// TODO: Cleanup the link
		url:   entry.Link,
		title: entry.Title,
	}

	// TODO: Sanitize the description
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
		// TODO: Synthesize a guid for this entry
		e.dedupKey = entry.Link
	}

	return e
}

func writeStories(ctx context.Context, db *sql.DB, svc *newspaper.Service, f FeedRecord, es []entryRecord) error {
	rows, err := db.QueryContext(ctx, `
		SELECT u.id
		FROM users u
		JOIN feed_subscriptions s
		ON s.user_id = u.id AND s.feed_id = $1
	`, f.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch subscribers for feed ID=%d: %w", f.ID, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	var subscribers []user.UserID
	for rows.Next() {
		var uid user.UserID
		if err := rows.Scan(&uid); err != nil {
			return fmt.Errorf("failed to scan subscriber's ID for feed ID=%d: %w", f.ID, err)
		}
		subscribers = append(subscribers, uid)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed while scanning subscribers for feed ID=%d: %w", f.ID, err)
	}
	if len(subscribers) == 0 {
		return nil
	}

	var errs []error
	for _, uid := range subscribers {
		errs = append(errs, writeStoriesForUser(ctx, svc, uid, f, es))
	}
	return errors.Join(errs...)
}

func writeStoriesForUser(
	ctx context.Context,
	svc *newspaper.Service,
	uid user.UserID,
	f FeedRecord,
	es []entryRecord,
) error {
	// TODO: apply user-defined filters to feed entries
	var ss []newspaper.Story
	for _, e := range es {
		var pubDate time.Time
		if e.publishedAt.Valid {
			pubDate = e.publishedAt.Time
		}
		var desc string
		if e.description.Valid {
			desc = e.description.String
		}
		s, err := newspaper.DraftStory(uid, e.title, desc, f.Title, e.id, pubDate)
		if err != nil {
			fmt.Printf("Cannot write a story from this entry ID=%d. Skipping.\n", e.id)
		} else {
			ss = append(ss, s)
		}
	}
	if len(ss) == 0 {
		return nil
	}
	return svc.SubmitStories(ctx, ss)
}

const contentTemplate = `
<!DOCTYPE html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
%s
</body>
</html>
`

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

	buf := bluemonday.UGCPolicy().SanitizeReader(res.Body)
	if buf.Len() == 0 {
		return fmt.Errorf("the body is empty")
	}

	baseUrl := new(*entryUrl)
	baseUrl.Path = ""
	baseUrl.RawPath = ""
	baseUrl.Fragment = ""
	baseUrl.RawFragment = ""
	baseUrl.RawQuery = ""
	baseUrl.User = nil

	article, err := readability.FromReader(buf, baseUrl)
	if err != nil {
		return err
	}

	buf.Reset()
	err = article.RenderHTML(buf)
	if err != nil {
		return err
	}
	if buf.Len() == 0 {
		return fmt.Errorf("failed to extract content")
	}
	content := fmt.Sprintf(contentTemplate, buf)

	_, err = db.ExecContext(ctx, `
		UPDATE feed_entries
		SET content = $1, snapshot_at = $2
		WHERE id = $3;
	`, content, time.Now(), entry.id)
	if err != nil {
		return fmt.Errorf("failed to fetch content")
	}

	return nil
}
