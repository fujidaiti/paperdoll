//go:build integration

package itest

import (
	"database/sql"
	"testing"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/fujidaiti/paperdoll/server/feature/newspaper"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
	"github.com/google/go-cmp/cmp"
)

// seedSchedule registers a newspaper_schedules row at the given local
// clock time (only the time-of-day component matters; minute_of_date is
// relative to at's own local midnight, matching schedule.go's own calc).
//
// TODO(#155): refactor to use a domain function once per-user schedule is supported
func seedSchedule(t *testing.T, label string, at time.Time) {
	t.Helper()
	local := at.Local()
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
	execOrFatal(t, `
		INSERT INTO newspaper_schedules (label, minute_of_date) VALUES ($1, $2)
	`, label, int(local.Sub(midnight).Minutes()))
}

// CollectJobs creates exactly one newspaper (and job) per registered user,
// all at the same shared schedule time, since per-user schedules don't
// exist yet.
func TestNewspaperCollectJobs_OneJobPerUser(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	uidAlice, _ := provisionTestAccount(t,
		"alice@example.com", "alice#password$123", "Pixel9a", now)
	uidBob, _ := provisionTestAccount(t,
		"bob@example.com", "bob#password$123", "Pixel9a", now)
	seedSchedule(t, "before", now.Add(-2*time.Minute))
	seedSchedule(t, "after", now.Add(3*time.Minute))

	jobs, err := newspaper.CollectJobs(t.Context(), testenv.DB(), now)
	if err != nil {
		t.Fatalf("CollectJobs returned an unexpected error: %v", err)
	}
	if n := len(jobs); n != 2 {
		t.Fatalf("got %d jobs, want 2", n)
	}

	jobByUser := make(map[user.UserID]int, len(jobs))
	for _, j := range jobs {
		jobByUser[j.UserID] = j.NewspaperID
	}
	for _, uid := range []user.UserID{uidAlice, uidBob} {
		newspaperID, ok := jobByUser[uid]
		if !ok {
			t.Errorf("no job returned for user %d", uid)
			continue
		}
		want := scanValOrFatal[int](t, `SELECT id FROM newspapers WHERE user_id = $1`, uid)
		if newspaperID != want {
			t.Errorf("job for user %d has NewspaperID %d, want %d", uid, newspaperID, want)
		}
	}
}

// CollectJobs returns no jobs when there are no users to issue newspapers for.
func TestNewspaperCollectJobs_NoUsers(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	seedSchedule(t, "before", now.Add(-2*time.Minute))
	seedSchedule(t, "after", now.Add(3*time.Minute))

	jobs, err := newspaper.CollectJobs(t.Context(), testenv.DB(), now)
	if err != nil {
		t.Fatalf("CollectJobs returned an unexpected error: %v", err)
	}
	if n := len(jobs); n != 0 {
		t.Errorf("got %d jobs, want 0", n)
	}
}

// A user who already has a newspaper for the current schedule tick is
// skipped, without blocking newspaper creation for other users.
func TestNewspaperCollectJobs_SkipsUserWithExistingNewspaper(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	seedSchedule(t, "before", now.Add(-2*time.Minute))
	seedSchedule(t, "after", now.Add(3*time.Minute))

	// Alice already has a newspaper for the upcoming schedule tick: run
	// CollectJobs for her alone first, before bob is registered.
	uidAlice, _ := provisionTestAccount(t,
		"alice@example.com", "alice#password$123", "Pixel9a", now)
	if _, err := newspaper.CollectJobs(t.Context(), testenv.DB(), now); err != nil {
		t.Fatalf("CollectJobs returned an unexpected error: %v", err)
	}

	uidBob, _ := provisionTestAccount(t,
		"bob@example.com", "bob#password$123", "Pixel9a", now)
	jobs, err := newspaper.CollectJobs(t.Context(), testenv.DB(), now)
	if err != nil {
		t.Fatalf("CollectJobs returned an unexpected error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1 (only bob's)", len(jobs))
	}
	if jobs[0].UserID != uidBob {
		t.Errorf("got job for user %d, want bob's job (user %d)", jobs[0].UserID, uidBob)
	}
	if want := scanValOrFatal[int](
		t,
		`SELECT id FROM newspapers WHERE user_id = $1`,
		uidBob,
	); jobs[0].NewspaperID != want {
		t.Errorf("got job.NewspaperID %d, want %d (bob's actual newspaper row)", jobs[0].NewspaperID, want)
	}
	if n := scanValOrFatal[int](t, `SELECT count(*) FROM newspapers WHERE user_id = $1`, uidBob); n != 1 {
		t.Errorf("got %d newspapers for bob, want 1", n)
	}
	if n := scanValOrFatal[int](t, `SELECT count(*) FROM newspapers WHERE user_id = $1`, uidAlice); n != 1 {
		t.Errorf("got %d newspapers for alice, want still 1 (no duplicate)", n)
	}
}

// newspaperStoryRow is a story row of a newspaper, identifying the source
// entry by its dedup key so that expectations can name it explicitly.
type newspaperStoryRow struct {
	EntryKey    string
	NewspaperID int
	Title       string
	Description sql.NullString
	Source      sql.NullString
	PublishedAt sql.NullTime
}

func scanNewspaperStories(t *testing.T, uid user.UserID) []newspaperStoryRow {
	t.Helper()
	return scanRowsOrFatal(t, `
		SELECT e.dedup_key, s.newspaper_id, s.title, s.description, s.source, s.published_at
		FROM stories s JOIN feed_entries e ON e.id = s.feed_entry_id
		WHERE s.user_id = $1
		ORDER BY e.dedup_key
	`, []any{uid}, func(rows *sql.Rows, r *newspaperStoryRow) error {
		return rows.Scan(&r.EntryKey, &r.NewspaperID, &r.Title, &r.Description, &r.Source, &r.PublishedAt)
	})
}

// Assembling a user's newspaper drafts stories from the entries of the user's
// subscribed feeds that were published in the newspaper's editorial interval.
// Other users' stories and newspapers are left untouched.
func TestNewspaperAssemble_DraftsStoriesFromSubscribedFeeds(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := mustTimeUTC("2026-07-15 10:00:00").Local()
	uidAlice, _ := provisionTestAccount(t,
		"alice@example.com", "alice#password$123", "Pixel9a", now)
	uidBob, _ := provisionTestAccount(t,
		"bob@example.com", "bob#password$123", "Pixel9a", now)
	seedSchedule(t, "before", now.Add(-2*time.Minute))
	seedSchedule(t, "after", now.Add(3*time.Minute))

	feedAID := seedFeed(t, uidAlice, "http://feed-a.test/rss", "./testdata/newspaper/feed_a.xml")
	feedBID := seedFeed(t, uidBob, "http://feed-b.test/rss", "./testdata/newspaper/feed_b.xml")
	for _, f := range []feed.FeedRecord{
		{ID: feedAID, URL: "http://feed-a.test/rss", Title: "Feed A"},
		{ID: feedBID, URL: "http://feed-b.test/rss", Title: "Feed B"},
	} {
		j := &feed.Job{DB: testenv.DB(), Feed: f, ScrpSvc: scraper.NewService(stubServerAddr)}
		if err := j.Do(t.Context()); err != nil {
			t.Fatalf("polling feed %d: job.Do returned an unexpected error: %v", f.ID, err)
		}
	}

	jobs, err := newspaper.CollectJobs(t.Context(), testenv.DB(), now)
	if err != nil {
		t.Fatalf("CollectJobs returned an unexpected error: %v", err)
	}
	var aliceJob *newspaper.Job
	for i := range jobs {
		if jobs[i].UserID == uidAlice {
			aliceJob = &jobs[i]
		}
	}
	if aliceJob == nil {
		t.Fatalf("got no job for alice among %d jobs", len(jobs))
	}
	if err := aliceJob.Do(t.Context()); err != nil {
		t.Fatalf("job.Do returned an unexpected error: %v", err)
	}

	want := []newspaperStoryRow{
		{
			EntryKey:    "a-current",
			NewspaperID: aliceJob.NewspaperID,
			Title:       "Current item",
			Description: sql.NullString{String: "Current desc", Valid: true},
			Source:      sql.NullString{String: "Feed A", Valid: true},
			PublishedAt: sql.NullTime{Time: mustTimeUTC("2026-07-15 10:00:00"), Valid: true},
		},
	}
	if d := cmp.Diff(want, scanNewspaperStories(t, uidAlice)); d != "" {
		t.Errorf("alice's stories mismatch:\n%s", d)
	}
	if got := scanNewspaperStories(t, uidBob); len(got) != 0 {
		t.Errorf("got %d stories for bob, want 0 (his job did not run)", len(got))
	}
	if draft := scanValOrFatal[bool](t, `
		SELECT draft FROM newspapers WHERE id = $1
	`, aliceJob.NewspaperID); draft {
		t.Error("alice's newspaper is still a draft, want it published")
	}
	if draft := scanValOrFatal[bool](t, `
		SELECT draft FROM newspapers WHERE user_id = $1
	`, uidBob); !draft {
		t.Error("bob's newspaper is published, want it to stay a draft")
	}
}

// A user who subscribes to a feed after its entries were stored still gets
// stories from them, the same as the feed's earlier subscriber.
func TestNewspaperAssemble_LaterSubscriberGetsStories(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := mustTimeUTC("2026-07-15 10:00:00").Local()
	uidAlice, _ := provisionTestAccount(t,
		"alice@example.com", "alice#password$123", "Pixel9a", now)
	uidBob, _ := provisionTestAccount(t,
		"bob@example.com", "bob#password$123", "Pixel9a", now)
	seedSchedule(t, "before", now.Add(-2*time.Minute))
	seedSchedule(t, "after", now.Add(3*time.Minute))

	feedID := seedFeed(t, uidAlice, "http://feed-b.test/rss", "./testdata/newspaper/feed_b.xml")
	j := &feed.Job{
		DB:      testenv.DB(),
		Feed:    feed.FeedRecord{ID: feedID, URL: "http://feed-b.test/rss", Title: "Feed B"},
		ScrpSvc: scraper.NewService(stubServerAddr),
	}
	if err := j.Do(t.Context()); err != nil {
		t.Fatalf("job.Do returned an unexpected error: %v", err)
	}
	seedFeed(t, uidBob, "http://feed-b.test/rss", "./testdata/newspaper/feed_b.xml")

	jobs, err := newspaper.CollectJobs(t.Context(), testenv.DB(), now)
	if err != nil {
		t.Fatalf("CollectJobs returned an unexpected error: %v", err)
	}
	if n := len(jobs); n != 2 {
		t.Fatalf("got %d jobs, want 2", n)
	}
	for _, j := range jobs {
		if err := j.Do(t.Context()); err != nil {
			t.Fatalf("job.Do for user %d returned an unexpected error: %v", j.UserID, err)
		}
	}

	for _, uid := range []user.UserID{uidAlice, uidBob} {
		got := scanNewspaperStories(t, uid)
		if len(got) != 1 || got[0].EntryKey != "b-current" {
			t.Errorf("got stories %+v for user %d, want exactly one from b-current", got, uid)
		}
	}
}

// Each newspaper covers only its own editorial interval, so consecutive
// newspapers get stories from different entries.
func TestNewspaperAssemble_CoversOwnIntervalOnly(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := mustTimeUTC("2026-07-15 10:00:00").Local()
	uid := provisionDefaultTestAccount(t, now)
	seedSchedule(t, "before", now.Add(-2*time.Minute))
	seedSchedule(t, "after", now.Add(3*time.Minute))

	feedID := seedFeed(t, uid, "http://feed-a.test/rss", "./testdata/newspaper/feed_a.xml")
	j := &feed.Job{
		DB:      testenv.DB(),
		Feed:    feed.FeedRecord{ID: feedID, URL: "http://feed-a.test/rss", Title: "Feed A"},
		ScrpSvc: scraper.NewService(stubServerAddr),
	}
	if err := j.Do(t.Context()); err != nil {
		t.Fatalf("job.Do returned an unexpected error: %v", err)
	}

	// The first newspaper is published at 10:03Z. The second one is published
	// at 09:58Z on the next day, and its interval starts at 10:03Z.
	var newspaperIDs []int
	for _, at := range []time.Time{
		now,
		mustTimeUTC("2026-07-16 09:50:00").Local(),
	} {
		jobs, err := newspaper.CollectJobs(t.Context(), testenv.DB(), at)
		if err != nil {
			t.Fatalf("CollectJobs at %s returned an unexpected error: %v", at, err)
		}
		if n := len(jobs); n != 1 {
			t.Fatalf("got %d jobs at %s, want 1", n, at)
		}
		if err := jobs[0].Do(t.Context()); err != nil {
			t.Fatalf("job.Do at %s returned an unexpected error: %v", at, err)
		}
		newspaperIDs = append(newspaperIDs, jobs[0].NewspaperID)
	}

	type storyRef struct {
		EntryKey    string
		NewspaperID int
	}
	var got []storyRef
	for _, s := range scanNewspaperStories(t, uid) {
		got = append(got, storyRef{s.EntryKey, s.NewspaperID})
	}
	want := []storyRef{
		{"a-current", newspaperIDs[0]},
		{"a-later", newspaperIDs[1]},
	}
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("stories mismatch:\n%s", d)
	}
}

// A user with no stories to draft at assembly time gets their draft
// newspaper row cleaned up rather than published empty.
func TestNewspaperAssemble_NoStoriesCleansUpNewspaper(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	provisionDefaultTestAccount(t, now)
	seedSchedule(t, "before", now.Add(-2*time.Minute))
	seedSchedule(t, "after", now.Add(3*time.Minute))

	jobs, err := newspaper.CollectJobs(t.Context(), testenv.DB(), now)
	if err != nil {
		t.Fatalf("CollectJobs returned an unexpected error: %v", err)
	}
	if n := len(jobs); n != 1 {
		t.Fatalf("got %d jobs, want 1", n)
	}

	if err := jobs[0].Do(t.Context()); err != nil {
		t.Fatalf("job.Do returned an unexpected error: %v", err)
	}

	if n := scanValOrFatal[int](t, `SELECT count(*) FROM newspapers`); n != 0 {
		t.Errorf("got %d newspapers, want 0 (cleaned up after finding no stories)", n)
	}
}
