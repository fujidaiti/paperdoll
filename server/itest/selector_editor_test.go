//go:build integration

package itest

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/tools/txtar"
)

// firstPostMarks marks the first article of
// testdata/feed/page_without_feed.html, whose nodes are:
//
//	<body data-id="0">
//	  <main data-id="1">
//	    <article data-id="2">
//	      <h2 href="http://blog.example.com/posts/first" data-id="3">First post</h2>
//	      <p data-id="4">The first post.</p>
//	    </article>
//	    <article data-id="5">
//	      <h2 href="http://blog.example.com/posts/second" data-id="6">Second post</h2>
//	      <p data-id="7">The second post.</p>
//	    </article>
//	  </main>
//	</body>
var firstPostMarks = feed.Marks{
	MarkedPosts:   []feed.MarkedPostIDs{{"box": 2, "title": 3, "link": 3}},
	ExcludedLinks: []string{},
	ExcludedNodes: []int{},
}

// editorRowExists reports whether the selector editor id is in the database.
func editorRowExists(t *testing.T, id int) bool {
	t.Helper()
	return scanValOrFatal[bool](t, `SELECT EXISTS (SELECT 1 FROM selector_editors WHERE id = $1)`, id)
}

// editorRowCount returns the number of selector editors of the feed feedID.
func editorRowCount(t *testing.T, feedID int) int {
	t.Helper()
	return scanValOrFatal[int](t, `SELECT count(*) FROM selector_editors WHERE feed_id = $1`, feedID)
}

// storedMarks returns the marks that the database stores for the editor id.
func storedMarks(t *testing.T, id int) feed.Marks {
	t.Helper()
	var marked string
	var marks feed.Marks
	types := pgtype.NewMap()
	scanRowOrFatal(t, `SELECT marked_posts, excluded_links, excluded_nodes FROM selector_editors WHERE id = $1`,
		[]any{id}, &marked, types.SQLScanner(&marks.ExcludedLinks), types.SQLScanner(&marks.ExcludedNodes))
	if err := json.Unmarshal([]byte(marked), &marks.MarkedPosts); err != nil {
		t.Fatal(err)
	}
	return marks
}

// savedRules returns the rules that the feed feedID stores, or "" without rules.
func savedRules(t *testing.T, feedID int) string {
	t.Helper()
	return scanValOrFatal[string](t, `SELECT COALESCE(extraction_rules::text, '') FROM feeds WHERE id = $1`, feedID)
}

func expireEditor(t *testing.T, id int) {
	t.Helper()
	execOrFatal(t, `UPDATE selector_editors SET expires_at = now() - interval '1 minute' WHERE id = $1`, id)
}

func TestSelectorEditor_Open(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	testenv.StubHTTP("www.nasa.gov", "/news-release/feed/", "./testdata/feed/nasa_news_release.xml")
	uid := provisionDefaultTestAccount(t, time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))

	t.Run("feed without selectors", func(t *testing.T) {
		e, err := s.OpenSelectorEditor(t.Context(), uid, page.ID)
		if err != nil {
			t.Fatalf("OpenSelectorEditor: %v", err)
		}
		if e.ID == 0 {
			t.Error("the editor has no ID")
		}
		if ttl := time.Until(e.ExpiresAt); ttl < 47*time.Hour || ttl > 49*time.Hour {
			t.Errorf("expires in %v, want 48 hours", ttl)
		}
		if e.Page == "" {
			t.Error("the page is empty")
		}
		if e.MarkedPosts == nil || len(e.MarkedPosts) != 0 ||
			e.ExcludedLinks == nil || len(e.ExcludedLinks) != 0 ||
			e.ExcludedNodes == nil || len(e.ExcludedNodes) != 0 {
			t.Errorf("marks = %+v, want empty lists", e.Marks)
		}
		if e.FoundPosts == nil || len(e.FoundPosts) != 0 ||
			e.ExcludedPosts == nil || len(e.ExcludedPosts) != 0 ||
			e.Preview == nil || len(e.Preview) != 0 {
			t.Errorf("state = %+v, want empty lists", e.EditorState)
		}
		if n := scanValOrFatal[int](t,
			`SELECT count(*) FROM selector_editors WHERE id = $1 AND user_id = $2 AND feed_id = $3`,
			e.ID, uid, page.ID,
		); n != 1 {
			t.Errorf("%d rows for the user and the feed, want 1", n)
		}
	})

	t.Run("feed with selectors", func(t *testing.T) {
		saved := must(os.ReadFile("./testdata/feed/page_without_feed_selectors.json"))
		execOrFatal(t, `UPDATE feeds SET extraction_rules = $1 WHERE id = $2`, string(saved), page.ID)

		e, err := s.OpenSelectorEditor(t.Context(), uid, page.ID)
		if err != nil {
			t.Fatalf("OpenSelectorEditor: %v", err)
		}
		if len(e.MarkedPosts) == 0 {
			t.Error("no marked posts")
		}
		if len(e.FoundPosts) == 0 {
			t.Error("no found posts")
		}
		for _, p := range e.FoundPosts {
			for _, k := range []string{"box", "title", "link"} {
				if _, ok := p.IDs[k]; !ok {
					t.Errorf("a found post has no %s: %v", k, p.IDs)
				}
			}
		}
	})

	t.Run("saved path that selects no node", func(t *testing.T) {
		saved := must(os.ReadFile("./testdata/feed/page_without_feed_selectors_missing_node.json"))
		execOrFatal(t, `UPDATE feeds SET extraction_rules = $1 WHERE id = $2`, string(saved), page.ID)

		e, err := s.OpenSelectorEditor(t.Context(), uid, page.ID)
		if err != nil {
			t.Fatalf("OpenSelectorEditor: %v", err)
		}
		if len(e.ExcludedNodes) != 0 {
			t.Errorf("excluded nodes = %v, want none", e.ExcludedNodes)
		}
	})

	t.Run("no feed", func(t *testing.T) {
		const missing = 1_000_000
		if _, err := s.OpenSelectorEditor(t.Context(), uid, missing); !errors.Is(err, feed.ErrFeedNotFound) {
			t.Errorf("got %v, want ErrFeedNotFound", err)
		}
		if n := editorRowCount(t, missing); n != 0 {
			t.Errorf("%d rows, want 0", n)
		}
	})

	t.Run("RSS feed", func(t *testing.T) {
		rss := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://www.nasa.gov/news-release/feed/"))))
		if _, err := s.OpenSelectorEditor(t.Context(), uid, rss.ID); !errors.Is(err, feed.ErrNotUnstructured) {
			t.Errorf("got %v, want ErrNotUnstructured", err)
		}
		if n := editorRowCount(t, rss.ID); n != 0 {
			t.Errorf("%d rows, want 0", n)
		}
	})

	t.Run("page that cannot be fetched", func(t *testing.T) {
		id := scanValOrFatal[int](t, `
			INSERT INTO feeds (url, site_url, title)
			VALUES ('http://gone.example.com/', 'http://gone.example.com/', 'Gone')
			RETURNING id`)
		_, err := s.OpenSelectorEditor(t.Context(), uid, id)
		if err == nil || errors.Is(err, feed.ErrFeedNotFound) || errors.Is(err, feed.ErrNotUnstructured) {
			t.Errorf("got %v, want a fetch error", err)
		}
		if n := editorRowCount(t, id); n != 0 {
			t.Errorf("%d rows, want 0", n)
		}
	})

	t.Run("second open of the same feed", func(t *testing.T) {
		first := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
		second, err := s.OpenSelectorEditor(t.Context(), uid, page.ID)
		if err != nil {
			t.Fatalf("OpenSelectorEditor: %v", err)
		}
		if second.ID == first.ID {
			t.Errorf("both editors have the ID %d", first.ID)
		}
		if !editorRowExists(t, first.ID) || !editorRowExists(t, second.ID) {
			t.Error("want both editors in the database")
		}
	})
}

func TestSelectorEditor_UpdateMarks(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	other, _ := provisionTestAccount(t, "other@example.com", "test#password$1234", "Pixel9a/Android", time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))
	e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))

	t.Run("valid marks", func(t *testing.T) {
		state, err := s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks)
		if err != nil {
			t.Fatalf("UpdateSelectorEditorMarks: %v", err)
		}
		if d := cmp.Diff(firstPostMarks, storedMarks(t, e.ID)); d != "" {
			t.Errorf("stored marks mismatch:\n%s", d)
		}
		if state.FoundPosts == nil || state.ExcludedPosts == nil || state.Preview == nil {
			t.Errorf("state = %+v, want non-nil lists", state)
		}
	})

	t.Run("invalid marks", func(t *testing.T) {
		invalid := feed.Marks{MarkedPosts: []feed.MarkedPostIDs{{"box": 99, "title": 3, "link": 3}}}
		_, err := s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, invalid)
		if !errors.Is(err, feed.ErrInvalidMarks) {
			t.Errorf("got %v, want ErrInvalidMarks", err)
		}
		if d := cmp.Diff(firstPostMarks, storedMarks(t, e.ID)); d != "" {
			t.Errorf("stored marks changed:\n%s", d)
		}
	})

	t.Run("excluded node with a marked post", func(t *testing.T) {
		marks := feed.Marks{
			MarkedPosts: []feed.MarkedPostIDs{
				{"box": 2, "title": 3, "link": 3},
				{"box": 5, "title": 6, "link": 6},
			},
			ExcludedLinks: []string{},
			ExcludedNodes: []int{5},
		}
		if _, err := s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, marks); err != nil {
			t.Fatalf("UpdateSelectorEditorMarks: %v", err)
		}
		if d := cmp.Diff(marks, storedMarks(t, e.ID)); d != "" {
			t.Errorf("stored marks mismatch:\n%s", d)
		}
		must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks))
	})

	t.Run("editor of another user", func(t *testing.T) {
		_, err := s.UpdateSelectorEditorMarks(t.Context(), other, page.ID, e.ID, feed.Marks{})
		if !errors.Is(err, feed.ErrEditorNotFound) {
			t.Errorf("got %v, want ErrEditorNotFound", err)
		}
		if d := cmp.Diff(firstPostMarks, storedMarks(t, e.ID)); d != "" {
			t.Errorf("stored marks changed:\n%s", d)
		}
	})

	t.Run("expired editor", func(t *testing.T) {
		expireEditor(t, e.ID)
		_, err := s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, feed.Marks{})
		if !errors.Is(err, feed.ErrEditorExpired) {
			t.Errorf("got %v, want ErrEditorExpired", err)
		}
		if d := cmp.Diff(firstPostMarks, storedMarks(t, e.ID)); d != "" {
			t.Errorf("stored marks changed:\n%s", d)
		}
	})
}

func TestSelectorEditor_Get(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	other, _ := provisionTestAccount(t, "other@example.com", "test#password$1234", "Pixel9a/Android", time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))
	opened := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
	must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, opened.ID, firstPostMarks))

	t.Run("own editor", func(t *testing.T) {
		e, err := s.GetSelectorEditor(t.Context(), uid, page.ID, opened.ID)
		if err != nil {
			t.Fatalf("GetSelectorEditor: %v", err)
		}
		if d := cmp.Diff(firstPostMarks, e.Marks); d != "" {
			t.Errorf("marks mismatch:\n%s", d)
		}
		if e.Page != opened.Page {
			t.Errorf("page = %s, want the page of the open: %s", e.Page, opened.Page)
		}
	})

	t.Run("editor of another user", func(t *testing.T) {
		if _, err := s.GetSelectorEditor(
			t.Context(),
			other,
			page.ID,
			opened.ID,
		); !errors.Is(
			err,
			feed.ErrEditorNotFound,
		) {
			t.Errorf("got %v, want ErrEditorNotFound", err)
		}
	})

	t.Run("expired editor", func(t *testing.T) {
		expireEditor(t, opened.ID)
		if _, err := s.GetSelectorEditor(t.Context(), uid, page.ID, opened.ID); !errors.Is(err, feed.ErrEditorExpired) {
			t.Errorf("got %v, want ErrEditorExpired", err)
		}
	})
}

func TestSelectorEditor_Commit(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	other, _ := provisionTestAccount(t, "other@example.com", "test#password$1234", "Pixel9a/Android", time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))
	entries := func(t *testing.T) int {
		t.Helper()
		return scanValOrFatal[int](t, `SELECT count(*) FROM feed_entries WHERE feed_id = $1`, page.ID)
	}

	t.Run("editor without selectors", func(t *testing.T) {
		e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
		if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); !errors.Is(err, feed.ErrNoSelectors) {
			t.Errorf("got %v, want ErrNoSelectors", err)
		}
		if got := savedRules(t, page.ID); got != "" {
			t.Errorf("the feed has the rules %s, want none", got)
		}
		if !editorRowExists(t, e.ID) {
			t.Error("the editor is deleted, want it kept")
		}
	})

	t.Run("editor of another user", func(t *testing.T) {
		e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
		must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks))
		if err := s.CommitSelectorEditor(t.Context(), other, page.ID, e.ID); !errors.Is(err, feed.ErrEditorNotFound) {
			t.Errorf("got %v, want ErrEditorNotFound", err)
		}
		if got := savedRules(t, page.ID); got != "" {
			t.Errorf("the feed has the rules %s, want none", got)
		}
		if n := entries(t); n != 0 {
			t.Errorf("the feed has %d entries, want 0", n)
		}
		if !editorRowExists(t, e.ID) {
			t.Error("the editor is deleted, want it kept")
		}
	})

	t.Run("expired editor", func(t *testing.T) {
		e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
		must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks))
		expireEditor(t, e.ID)
		if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); !errors.Is(err, feed.ErrEditorExpired) {
			t.Errorf("got %v, want ErrEditorExpired", err)
		}
		if got := savedRules(t, page.ID); got != "" {
			t.Errorf("the feed has the rules %s, want none", got)
		}
		if n := entries(t); n != 0 {
			t.Errorf("the feed has %d entries, want 0", n)
		}
		if !editorRowExists(t, e.ID) {
			t.Error("the editor is deleted, want it kept")
		}
	})

	t.Run("editor with marks", func(t *testing.T) {
		e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
		must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks))
		if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); err != nil {
			t.Fatalf("CommitSelectorEditor: %v", err)
		}
		if editorRowExists(t, e.ID) {
			t.Error("the committed editor is kept, want it deleted")
		}
		if savedRules(t, page.ID) == "" {
			t.Error("the feed has no rules")
		}
		if n := entries(t); n == 0 {
			t.Error("the feed has no entries")
		}
	})
}

func TestSelectorEditor_Close(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	other, _ := provisionTestAccount(t, "other@example.com", "test#password$1234", "Pixel9a/Android", time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))
	e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
	must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks))

	t.Run("editor of another user", func(t *testing.T) {
		if err := s.CloseSelectorEditor(t.Context(), other, page.ID, e.ID); !errors.Is(err, feed.ErrEditorNotFound) {
			t.Errorf("got %v, want ErrEditorNotFound", err)
		}
		if !editorRowExists(t, e.ID) {
			t.Error("the editor is deleted, want it kept")
		}
	})

	t.Run("own editor", func(t *testing.T) {
		if err := s.CloseSelectorEditor(t.Context(), uid, page.ID, e.ID); err != nil {
			t.Fatalf("CloseSelectorEditor: %v", err)
		}
		if editorRowExists(t, e.ID) {
			t.Error("the closed editor is kept, want it deleted")
		}
		if got := savedRules(t, page.ID); got != "" {
			t.Errorf("the feed has the rules %s, want none", got)
		}
	})
}

func TestSelectorEditor_DeleteExpired(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))
	expired := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
	valid := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
	expireEditor(t, expired.ID)

	n, err := feed.DeleteExpiredSelectorEditors(t.Context(), testenv.DB())
	if err != nil {
		t.Fatalf("DeleteExpiredSelectorEditors: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted %d editors, want 1", n)
	}
	if editorRowExists(t, expired.ID) {
		t.Error("the expired editor is kept")
	}
	if !editorRowExists(t, valid.ID) {
		t.Error("the editor that has not expired is deleted")
	}
}

// TestSelectorEditor_BuildSelectorsForNewPage runs the main use case: a person
// subscribes to a page without a feed, marks a post, sees what the editor finds,
// and saves the selectors.
func TestSelectorEditor_BuildSelectorsForNewPage(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	archive := must(txtar.ParseFile("./testdata/selector_editor/build-selectors-for-new-page.txtar"))
	sections := map[string][]byte{}
	for _, f := range archive.Files {
		sections[f.Name] = f.Data
	}
	// The stub server serves files, so the page of the case is written to one.
	pagePath := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(pagePath, sections["page.html"], 0o644); err != nil {
		t.Fatal(err)
	}
	testenv.StubHTTP("blog.example.com", "/", pagePath)
	uid := provisionDefaultTestAccount(t, time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))

	page, err := s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/")))
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	e, err := s.OpenSelectorEditor(t.Context(), uid, page.ID)
	if err != nil {
		t.Fatalf("OpenSelectorEditor: %v", err)
	}
	var marks feed.Marks
	if err := json.Unmarshal(sections["marks.json"], &marks); err != nil {
		t.Fatal(err)
	}
	state, err := s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, marks)
	if err != nil {
		t.Fatalf("UpdateSelectorEditorMarks: %v", err)
	}
	var got, want any
	if err := json.Unmarshal(must(json.Marshal(state)), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(sections["state.json"], &want); err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("state mismatch (-want +got):\n%s", d)
	}
	if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); err != nil {
		t.Fatalf("CommitSelectorEditor: %v", err)
	}
}
