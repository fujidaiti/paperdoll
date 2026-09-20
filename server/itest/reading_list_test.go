//go:build integration

package itest

import (
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/readinglist"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
)

// seedFeedEntry inserts a minimal feed and feed entry, returning the entry ID.
// Feed entries are normally created by feed polling, which is far more setup
// than these tests need, so this bypasses it the same way newspaper_test.go
// does.
func seedFeedEntry(t *testing.T, seed, title, description string) int {
	t.Helper()
	feedID := scanValOrFatal[int](t, `
		INSERT INTO feeds (url, title) VALUES ($1, $2) RETURNING id
	`, fmt.Sprintf("http://%s.test/rss", seed), title+" Feed")
	return scanValOrFatal[int](t, `
		INSERT INTO feed_entries (dedup_key, feed_id, url, title, description, snapshot_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, seed, feedID, fmt.Sprintf("http://%s.test/entry", seed), title, description, time.Now())
}

// SaveFeedEntry copies the entry's metadata into a new reading list item owned
// by the calling user.
func TestReadingList_SaveFeedEntry(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := mustTimeUTC("2026-07-15 10:00:00")
	uid := provisionDefaultTestAccount(t, now)
	entryID := seedFeedEntry(t, "save-me", "Save Me", "An entry worth saving.")

	s := readinglist.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	item, err := s.SaveFeedEntry(t.Context(), uid, entryID)
	if err != nil {
		t.Fatalf("SaveFeedEntry returned an unexpected error: %v", err)
	}

	if item.ID == 0 {
		t.Error("item ID must be assigned, got 0")
	}
	if item.ResourceID != entryID {
		t.Errorf("got resource ID %d, want the seeded entry %d", item.ResourceID, entryID)
	}
	if item.Kind != "feed_entry" {
		t.Errorf("got kind %q, want %q", item.Kind, "feed_entry")
	}
	if item.Title != "Save Me" {
		t.Errorf("got title %q, want the entry's own title %q", item.Title, "Save Me")
	}
	if item.Description == nil {
		t.Error("got nil description, want the entry's own description")
	} else if *item.Description != "An entry worth saving." {
		t.Errorf("got description %q, want %q", *item.Description, "An entry worth saving.")
	}
	if item.SavedAt.IsZero() {
		t.Error("saved_at must be assigned, got the zero time")
	}

	// The saved row itself, column by column: the returned item is only a
	// report of what was written, not proof of it.
	if n := scanValOrFatal[int](t, `SELECT count(*) FROM reading_list_items`); n != 1 {
		t.Fatalf("got %d reading list items, want exactly 1", n)
	}
	var (
		gotKind        string
		gotFeedEntryID int
		gotWebClipID   *int
		gotTitle       string
		gotDescription *string
		gotArchived    bool
		gotSavedAt     time.Time
		gotUID         user.UserID
	)
	scanRowOrFatal(t, `
		SELECT kind, feed_entry_id, web_clip_id, title, description, archived, saved_at, user_id
		FROM reading_list_items WHERE id = $1
	`, []any{item.ID},
		&gotKind, &gotFeedEntryID, &gotWebClipID, &gotTitle,
		&gotDescription, &gotArchived, &gotSavedAt, &gotUID,
	)
	if gotKind != "feed_entry" {
		t.Errorf("got stored kind %q, want %q", gotKind, "feed_entry")
	}
	if gotFeedEntryID != entryID {
		t.Errorf("got stored feed_entry_id %d, want the seeded entry %d", gotFeedEntryID, entryID)
	}
	if gotWebClipID != nil {
		t.Errorf("got stored web_clip_id %d, want NULL for a feed entry item", *gotWebClipID)
	}
	if gotTitle != "Save Me" {
		t.Errorf("got stored title %q, want the entry's own title %q", gotTitle, "Save Me")
	}
	if gotDescription == nil {
		t.Error("got stored description NULL, want the entry's own description")
	} else if *gotDescription != "An entry worth saving." {
		t.Errorf("got stored description %q, want %q", *gotDescription, "An entry worth saving.")
	}
	if gotArchived {
		t.Error("a newly saved item must not be archived")
	}
	if !gotSavedAt.Equal(item.SavedAt) {
		t.Errorf("got stored saved_at %v, want the returned %v", gotSavedAt, item.SavedAt)
	}
	if gotUID != uid {
		t.Errorf("got stored user_id %d, want alice's id %d", gotUID, uid)
	}
}

// SaveWebClip returns a placeholder item immediately, then fills in the clip's
// real metadata from the fetched page in the background.
func TestReadingList_SaveWebClip(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := mustTimeUTC("2026-07-15 10:00:00")
	uid := provisionDefaultTestAccount(t, now)

	clipURL := must(url.Parse("http://clip.test/article"))
	testenv.StubHTTP(clipURL.Host, clipURL.Path, "./testdata/reading_list/web_clip.html")

	s := readinglist.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	item, err := s.SaveWebClip(t.Context(), uid, *clipURL, "Placeholder Title")
	if err != nil {
		t.Fatalf("SaveWebClip returned an unexpected error: %v", err)
	}

	// The synchronous part: a placeholder item and a not-yet-fetched clip.
	if item.ID == 0 {
		t.Error("item ID must be assigned, got 0")
	}
	if item.Kind != "web_clip" {
		t.Errorf("got kind %q, want %q", item.Kind, "web_clip")
	}
	if item.Title != "Placeholder Title" {
		t.Errorf("got title %q, want the caller-supplied placeholder %q", item.Title, "Placeholder Title")
	}
	gotURL := scanValOrFatal[string](t, `SELECT url FROM web_clips WHERE id = $1`, item.ResourceID)
	if gotURL != clipURL.String() {
		t.Errorf("got clip URL %q, want %q", gotURL, clipURL.String())
	}

	// The saved row itself. Its title is deliberately left out here: the
	// background fetch may already have replaced the placeholder, so the title
	// is only asserted once the fetch is known to be finished, below.
	if n := scanValOrFatal[int](t, `SELECT count(*) FROM reading_list_items`); n != 1 {
		t.Fatalf("got %d reading list items, want exactly 1", n)
	}
	var (
		gotKind        string
		gotWebClipID   int
		gotFeedEntryID *int
		gotDescription *string
		gotArchived    bool
		gotSavedAt     time.Time
		gotUID         user.UserID
	)
	scanRowOrFatal(t, `
		SELECT kind, web_clip_id, feed_entry_id, description, archived, saved_at, user_id
		FROM reading_list_items WHERE id = $1
	`, []any{item.ID},
		&gotKind, &gotWebClipID, &gotFeedEntryID,
		&gotDescription, &gotArchived, &gotSavedAt, &gotUID,
	)
	if gotKind != "web_clip" {
		t.Errorf("got stored kind %q, want %q", gotKind, "web_clip")
	}
	if gotWebClipID != item.ResourceID {
		t.Errorf("got stored web_clip_id %d, want the returned resource ID %d", gotWebClipID, item.ResourceID)
	}
	if gotFeedEntryID != nil {
		t.Errorf("got stored feed_entry_id %d, want NULL for a web clip item", *gotFeedEntryID)
	}
	if gotDescription != nil {
		t.Errorf("got stored description %q, want NULL (the clip isn't fetched yet)", *gotDescription)
	}
	if gotArchived {
		t.Error("a newly saved item must not be archived")
	}
	if !gotSavedAt.Equal(item.SavedAt) {
		t.Errorf("got stored saved_at %v, want the returned %v", gotSavedAt, item.SavedAt)
	}
	if gotUID != uid {
		t.Errorf("got stored user_id %d, want alice's id %d", gotUID, uid)
	}

	// TODO: make this better
	// The asynchronous part. Waiting here is not optional: the background fetch
	// runs on context.Background(), so returning before it finishes would let it
	// write to the DB while testenv.TearDown is restoring the snapshot.
	var status string
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		status = scanValOrFatal[string](t, `SELECT fetch_status FROM web_clips WHERE id = $1`, item.ResourceID)
		if status != "pending" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if status != "done" {
		t.Fatalf("got fetch_status %q, want %q after the background fetch", status, "done")
	}

	const wantTitle = "Reading List Test Clip"
	var clipTitle string
	var clipContent string
	scanRowOrFatal(t, `
		SELECT title, content FROM web_clips WHERE id = $1
	`, []any{item.ResourceID}, &clipTitle, &clipContent)
	if clipTitle != wantTitle {
		t.Errorf("got clip title %q, want the fetched title %q", clipTitle, wantTitle)
	}
	if clipContent == "" {
		t.Error("clip content must be filled in by the background fetch, got an empty string")
	}
	// The placeholder title is replaced once the real one is known.
	itemTitle := scanValOrFatal[string](t, `SELECT title FROM reading_list_items WHERE id = $1`, item.ID)
	if itemTitle != wantTitle {
		t.Errorf("got item title %q, want the fetched title %q", itemTitle, wantTitle)
	}
}

// SaveWebClipByID copies an existing clip's metadata into a new reading list
// item owned by the calling user.
func TestReadingList_SaveWebClipByID(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := mustTimeUTC("2026-07-15 10:00:00")
	uid := provisionDefaultTestAccount(t, now)
	// Insert the clip directly: nothing in the domain creates a bare clip
	// without also saving it to the caller's own reading list.
	clipID := scanValOrFatal[int](t, `
		INSERT INTO web_clips (url, title, description) VALUES ($1, $2, $3) RETURNING id
	`, "http://clip.test/existing", "Existing Clip", "An existing clip.")

	s := readinglist.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	item, err := s.SaveWebClipByID(t.Context(), uid, clipID)
	if err != nil {
		t.Fatalf("SaveWebClipByID returned an unexpected error: %v", err)
	}

	if item.ResourceID != clipID {
		t.Errorf("got resource ID %d, want the seeded clip %d", item.ResourceID, clipID)
	}
	if item.Kind != "web_clip" {
		t.Errorf("got kind %q, want %q", item.Kind, "web_clip")
	}
	if item.Title != "Existing Clip" {
		t.Errorf("got title %q, want the clip's own title %q", item.Title, "Existing Clip")
	}
	if item.Description == nil {
		t.Error("got nil description, want the clip's own description")
	} else if *item.Description != "An existing clip." {
		t.Errorf("got description %q, want %q", *item.Description, "An existing clip.")
	}

	// The saved row itself, column by column.
	if n := scanValOrFatal[int](t, `SELECT count(*) FROM reading_list_items`); n != 1 {
		t.Fatalf("got %d reading list items, want exactly 1", n)
	}
	var (
		gotKind        string
		gotWebClipID   int
		gotFeedEntryID *int
		gotTitle       string
		gotDescription *string
		gotArchived    bool
		gotSavedAt     time.Time
		gotUID         user.UserID
	)
	scanRowOrFatal(t, `
		SELECT kind, web_clip_id, feed_entry_id, title, description, archived, saved_at, user_id
		FROM reading_list_items WHERE id = $1
	`, []any{item.ID},
		&gotKind, &gotWebClipID, &gotFeedEntryID, &gotTitle,
		&gotDescription, &gotArchived, &gotSavedAt, &gotUID,
	)
	if gotKind != "web_clip" {
		t.Errorf("got stored kind %q, want %q", gotKind, "web_clip")
	}
	if gotWebClipID != clipID {
		t.Errorf("got stored web_clip_id %d, want the seeded clip %d", gotWebClipID, clipID)
	}
	if gotFeedEntryID != nil {
		t.Errorf("got stored feed_entry_id %d, want NULL for a web clip item", *gotFeedEntryID)
	}
	if gotTitle != "Existing Clip" {
		t.Errorf("got stored title %q, want the clip's own title %q", gotTitle, "Existing Clip")
	}
	if gotDescription == nil {
		t.Error("got stored description NULL, want the clip's own description")
	} else if *gotDescription != "An existing clip." {
		t.Errorf("got stored description %q, want %q", *gotDescription, "An existing clip.")
	}
	if gotArchived {
		t.Error("a newly saved item must not be archived")
	}
	if !gotSavedAt.Equal(item.SavedAt) {
		t.Errorf("got stored saved_at %v, want the returned %v", gotSavedAt, item.SavedAt)
	}
	if gotUID != uid {
		t.Errorf("got stored user_id %d, want bob's id %d", gotUID, uid)
	}
}

// web_clips.title is nullable but reading_list_items.title is NOT NULL, so a
// clip that has no title yet is saved with an empty title rather than failing.
func TestReadingList_SaveWebClipByID_UntitledClip(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := mustTimeUTC("2026-07-15 10:00:00")
	uid := provisionDefaultTestAccount(t, now)
	clipID := scanValOrFatal[int](t, `
		INSERT INTO web_clips (url) VALUES ($1) RETURNING id
	`, "http://clip.test/untitled")

	s := readinglist.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	item, err := s.SaveWebClipByID(t.Context(), uid, clipID)
	if err != nil {
		t.Fatalf("SaveWebClipByID returned an unexpected error: %v", err)
	}

	if item.Title != "" {
		t.Errorf("got title %q, want an empty string for an untitled clip", item.Title)
	}
	gotTitle := scanValOrFatal[string](t, `SELECT title FROM reading_list_items WHERE id = $1`, item.ID)
	if gotTitle != "" {
		t.Errorf("got stored title %q, want an empty string", gotTitle)
	}
}

// DeleteItem only removes items owned by the calling user, and reports whether
// anything was removed instead of failing on an unknown id.
func TestReadingList_DeleteRequiresOwnership(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := mustTimeUTC("2026-07-15 10:00:00")
	uidAlice, _ := provisionTestAccount(t,
		"alice@example.com", "alice#password$123", "Pixel9a", now)
	uidBob, _ := provisionTestAccount(t,
		"bob@example.com", "bob#password$123", "Pixel9a", now)
	entryID := seedFeedEntry(t, "delete-me", "Delete Me", "An entry worth deleting.")

	s := readinglist.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	item := must(s.SaveFeedEntry(t.Context(), uidAlice, entryID))

	deleted, err := s.DeleteItem(t.Context(), uidBob, item.ID)
	if err != nil {
		t.Fatalf("DeleteItem returned an unexpected error for bob: %v", err)
	}
	if deleted {
		t.Error("got deleted=true for bob, want false (he must not delete alice's item)")
	}
	if n := scanValOrFatal[int](t, `SELECT count(*) FROM reading_list_items WHERE id = $1`, item.ID); n != 1 {
		t.Errorf("got %d rows, want the item to still exist", n)
	}

	deleted, err = s.DeleteItem(t.Context(), uidAlice, item.ID)
	if err != nil {
		t.Fatalf("DeleteItem returned an unexpected error for alice: %v", err)
	}
	if !deleted {
		t.Error("got deleted=false for alice, want true (she owns the item)")
	}
	if n := scanValOrFatal[int](t, `SELECT count(*) FROM reading_list_items WHERE id = $1`, item.ID); n != 0 {
		t.Errorf("got %d rows, want the item to be gone", n)
	}

	// The item no longer exists, so deleting it again is a no-op, not an error.
	deleted, err = s.DeleteItem(t.Context(), uidAlice, item.ID)
	if err != nil {
		t.Fatalf("DeleteItem returned an unexpected error for a deleted item: %v", err)
	}
	if deleted {
		t.Error("got deleted=true for an id that no longer exists, want false")
	}
}

// ArchiveItem and UnarchiveItem only affect items owned by the calling user,
// and report ErrItemNotFound otherwise.
func TestReadingList_ArchiveAndUnarchiveRequireOwnership(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	now := mustTimeUTC("2026-07-15 10:00:00")
	uidAlice, _ := provisionTestAccount(t,
		"alice@example.com", "alice#password$123", "Pixel9a", now)
	uidBob, _ := provisionTestAccount(t,
		"bob@example.com", "bob#password$123", "Pixel9a", now)
	entryID := seedFeedEntry(t, "archive-me", "Archive Me", "An entry worth archiving.")

	s := readinglist.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	item := must(s.SaveFeedEntry(t.Context(), uidAlice, entryID))
	isArchived := func() bool {
		return scanValOrFatal[bool](t, `SELECT archived FROM reading_list_items WHERE id = $1`, item.ID)
	}

	if err := s.ArchiveItem(t.Context(), uidBob, item.ID); !errors.Is(err, readinglist.ErrItemNotFound) {
		t.Errorf("got %v for bob, want %v (he must not archive alice's item)", err, readinglist.ErrItemNotFound)
	}
	if isArchived() {
		t.Error("item must remain unarchived after bob's attempt")
	}

	if err := s.ArchiveItem(t.Context(), uidAlice, item.ID); err != nil {
		t.Fatalf("ArchiveItem returned an unexpected error for alice: %v", err)
	}
	if !isArchived() {
		t.Error("item must be archived after alice archives it")
	}

	if err := s.UnarchiveItem(t.Context(), uidBob, item.ID); !errors.Is(err, readinglist.ErrItemNotFound) {
		t.Errorf("got %v for bob, want %v (he must not unarchive alice's item)", err, readinglist.ErrItemNotFound)
	}
	if !isArchived() {
		t.Error("item must remain archived after bob's attempt")
	}

	if err := s.UnarchiveItem(t.Context(), uidAlice, item.ID); err != nil {
		t.Fatalf("UnarchiveItem returned an unexpected error for alice: %v", err)
	}
	if isArchived() {
		t.Error("item must be unarchived after alice unarchives it")
	}
}
