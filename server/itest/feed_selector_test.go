//go:build integration

package itest

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/itest/testenv"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgtype"
)

// The simplified page of testdata/feed/page_without_feed.html with node IDs
// is:
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
//
// testdata/feed/page_without_feed_selectors.json holds the extraction rules
// that are generated from firstPostMarks, and
// testdata/feed/page_without_feed_state.json holds what they find.
// testdata/feed/page_without_feed_excluded_node_state.json holds what the rules
// generated from excludedNodeMarks find.

// firstPostMarks labels the first article, and excludes the link of the
// second one.
var firstPostMarks = feed.Marks{
	LabeledPosts:  []feed.LabeledPostIDs{{"box": 2, "title": 3, "link": 3}},
	ExcludedLinks: []string{"blog.example.com/posts/second"},
	ExcludedNodes: []int{},
}

// excludedNodeMarks labels both articles, and excludes the second one as a
// node, which wins over its label.
var excludedNodeMarks = feed.Marks{
	LabeledPosts: []feed.LabeledPostIDs{
		{"box": 2, "title": 3, "link": 3},
		{"box": 5, "title": 6, "link": 6},
	},
	ExcludedLinks: []string{},
	ExcludedNodes: []int{5},
}

// checkJSON compares the JSON of got with the JSON file at path.
func checkJSON(t *testing.T, path string, got any) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(must(json.Marshal(got)), &g); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(must(os.ReadFile(path)), &w); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if d := cmp.Diff(w, g); d != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", path, d)
	}
}

// editorRowExists reports whether the selector editor id is in the database.
func editorRowExists(t *testing.T, id int) bool {
	t.Helper()
	return scanValOrFatal[bool](t, `SELECT EXISTS (SELECT 1 FROM selector_editors WHERE id = $1)`, id)
}

func TestFeed_OpenSelectorEditor(t *testing.T) {
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
		wantPage := `<!DOCTYPE html><html><head><title>Example Blog</title>` +
			`<link rel="canonical" href="http://blog.example.com/"/></head>` +
			`<body data-id="0"><main data-id="1">` +
			`<article data-id="2"><h2 href="http://blog.example.com/posts/first" data-id="3">First post</h2>` +
			`<p data-id="4">The first post.</p></article>` +
			`<article data-id="5"><h2 href="http://blog.example.com/posts/second" data-id="6">Second post</h2>` +
			`<p data-id="7">The second post.</p></article>` +
			`</main></body></html>`
		if e.Page != wantPage {
			t.Errorf("page = %s, want %s", e.Page, wantPage)
		}
		if d := cmp.Diff(
			feed.Marks{LabeledPosts: []feed.LabeledPostIDs{}, ExcludedLinks: []string{}, ExcludedNodes: []int{}},
			e.Marks,
		); d != "" {
			t.Errorf("marks mismatch:\n%s", d)
		}
		wantState := feed.EditorState{
			MatchedPosts:  []feed.MatchedPost{},
			ExcludedPosts: []feed.ExcludedPost{},
			Preview:       []feed.ExtractedPost{},
		}
		if d := cmp.Diff(wantState, e.EditorState); d != "" {
			t.Errorf("state mismatch:\n%s", d)
		}
		if ttl := time.Until(e.ExpiresAt); ttl < 47*time.Hour || ttl > 49*time.Hour {
			t.Errorf("expires in %v, want 48 hours", ttl)
		}
		if !editorRowExists(t, e.ID) {
			t.Error("the editor is not stored")
		}
	})

	t.Run("feed with selectors", func(t *testing.T) {
		saved := must(os.ReadFile("./testdata/feed/page_without_feed_selectors.json"))
		execOrFatal(t, `UPDATE feeds SET extraction_rules = $1 WHERE id = $2`, string(saved), page.ID)

		e, err := s.OpenSelectorEditor(t.Context(), uid, page.ID)
		if err != nil {
			t.Fatalf("OpenSelectorEditor: %v", err)
		}
		// The posts that the saved selectors find are labeled.
		if d := cmp.Diff(firstPostMarks, e.Marks); d != "" {
			t.Errorf("marks mismatch:\n%s", d)
		}
		checkJSON(t, "./testdata/feed/page_without_feed_state.json", e.EditorState)
	})

	t.Run("feed with excluded nodes", func(t *testing.T) {
		e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
		must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, excludedNodeMarks))
		if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); err != nil {
			t.Fatalf("CommitSelectorEditor: %v", err)
		}

		e, err := s.OpenSelectorEditor(t.Context(), uid, page.ID)
		if err != nil {
			t.Fatalf("OpenSelectorEditor: %v", err)
		}
		// The excluded node is excluded again. The second article was labeled, but
		// the saved selectors do not find it, so it is not labeled.
		want := feed.Marks{
			LabeledPosts:  []feed.LabeledPostIDs{{"box": 2, "title": 3, "link": 3}},
			ExcludedLinks: []string{},
			ExcludedNodes: []int{5},
		}
		if d := cmp.Diff(want, e.Marks); d != "" {
			t.Errorf("marks mismatch:\n%s", d)
		}
		checkJSON(t, "./testdata/feed/page_without_feed_excluded_node_state.json", e.EditorState)
	})

	t.Run("saved path that selects no node", func(t *testing.T) {
		saved := must(os.ReadFile("./testdata/feed/page_without_feed_selectors_missing_node.json"))
		execOrFatal(t, `UPDATE feeds SET extraction_rules = $1 WHERE id = $2`, string(saved), page.ID)

		e, err := s.OpenSelectorEditor(t.Context(), uid, page.ID)
		if err != nil {
			t.Fatalf("OpenSelectorEditor: %v", err)
		}
		if d := cmp.Diff(firstPostMarks, e.Marks); d != "" {
			t.Errorf("marks mismatch:\n%s", d)
		}
		if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); err != nil {
			t.Fatalf("CommitSelectorEditor: %v", err)
		}
		// The path is dropped from the saved rules.
		var got, want any
		_ = json.Unmarshal(
			[]byte(scanValOrFatal[string](t, `SELECT extraction_rules FROM feeds WHERE id = $1`, page.ID)),
			&got,
		)
		_ = json.Unmarshal(must(os.ReadFile("./testdata/feed/page_without_feed_selectors.json")), &want)
		if d := cmp.Diff(want, got); d != "" {
			t.Errorf("saved rules mismatch:\n%s", d)
		}
	})

	t.Run("no feed", func(t *testing.T) {
		if _, err := s.OpenSelectorEditor(t.Context(), uid, 1_000_000); !errors.Is(err, feed.ErrFeedNotFound) {
			t.Errorf("got %v, want ErrFeedNotFound", err)
		}
	})

	t.Run("RSS feed", func(t *testing.T) {
		rss := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://www.nasa.gov/news-release/feed/"))))
		if _, err := s.OpenSelectorEditor(t.Context(), uid, rss.ID); !errors.Is(err, feed.ErrNotUnstructured) {
			t.Errorf("got %v, want ErrNotUnstructured", err)
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
	})
}

func TestFeed_UpdateSelectorEditorMarks(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	other, _ := provisionTestAccount(t, "other@example.com", "test#password$1234", "Pixel9a/Android", time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))
	e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))

	state, err := s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks)
	if err != nil {
		t.Fatalf("UpdateSelectorEditorMarks: %v", err)
	}
	checkJSON(t, "./testdata/feed/page_without_feed_state.json", state)
	var labeled string
	var excluded []string
	var excludedNodes []int
	types := pgtype.NewMap()
	scanRowOrFatal(t, `SELECT labeled_posts, excluded_links, excluded_nodes FROM selector_editors WHERE id = $1`,
		[]any{e.ID}, &labeled, types.SQLScanner(&excluded), types.SQLScanner(&excludedNodes))
	if d := cmp.Diff(`[{"box":2,"link":3,"title":3}]`, labeled); d != "" {
		t.Errorf("stored labeled posts mismatch:\n%s", d)
	}
	if d := cmp.Diff([]string{"blog.example.com/posts/second"}, excluded); d != "" {
		t.Errorf("stored excluded links mismatch:\n%s", d)
	}
	if d := cmp.Diff([]int{}, excludedNodes); d != "" {
		t.Errorf("stored excluded nodes mismatch:\n%s", d)
	}

	t.Run("excluded node with a labeled post", func(t *testing.T) {
		state, err := s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, excludedNodeMarks)
		if err != nil {
			t.Fatalf("UpdateSelectorEditorMarks: %v", err)
		}
		// The second article is not a post, but stays labeled.
		checkJSON(t, "./testdata/feed/page_without_feed_excluded_node_state.json", state)
		got := must(s.GetSelectorEditor(t.Context(), uid, page.ID, e.ID))
		if d := cmp.Diff(excludedNodeMarks, got.Marks); d != "" {
			t.Errorf("stored marks mismatch:\n%s", d)
		}
	})

	for _, tt := range []struct {
		name  string
		marks feed.Marks
	}{
		{"no node", feed.Marks{LabeledPosts: []feed.LabeledPostIDs{{"box": 2, "title": 3, "link": 99}}}},
		{"body as a post", feed.Marks{LabeledPosts: []feed.LabeledPostIDs{{"box": 0, "title": 3, "link": 3}}}},
		{"no box", feed.Marks{LabeledPosts: []feed.LabeledPostIDs{{"title": 3, "link": 3}}}},
		{"field outside the post", feed.Marks{LabeledPosts: []feed.LabeledPostIDs{{"box": 2, "title": 6, "link": 3}}}},
		{"field that does not accept the node", feed.Marks{
			LabeledPosts: []feed.LabeledPostIDs{{"box": 2, "title": 2, "link": 3}},
		}},
		{"no link", feed.Marks{LabeledPosts: []feed.LabeledPostIDs{{"box": 2, "title": 3}}}},
		{"no title", feed.Marks{LabeledPosts: []feed.LabeledPostIDs{{"box": 2, "link": 3}}}},
		{"unknown field", feed.Marks{LabeledPosts: []feed.LabeledPostIDs{{"box": 2, "title": 3, "link": 3, "author": 4}}}},
		{"excluded link of a post", feed.Marks{
			LabeledPosts:  []feed.LabeledPostIDs{{"box": 2, "title": 3, "link": 3}},
			ExcludedLinks: []string{"blog.example.com/posts/first"},
		}},
		{"excluded node that is not on the page", feed.Marks{ExcludedNodes: []int{99}}},
		{"excluded body", feed.Marks{ExcludedNodes: []int{0}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := s.UpdateSelectorEditorMarks(
				t.Context(),
				uid,
				page.ID,
				e.ID,
				tt.marks,
			); !errors.Is(
				err,
				feed.ErrInvalidMarks,
			) {
				t.Errorf("got %v, want ErrInvalidMarks", err)
			}
		})
	}

	t.Run("editor of another user", func(t *testing.T) {
		if _, err := s.UpdateSelectorEditorMarks(
			t.Context(),
			other,
			page.ID,
			e.ID,
			firstPostMarks,
		); !errors.Is(
			err,
			feed.ErrEditorNotFound,
		) {
			t.Errorf("got %v, want ErrEditorNotFound", err)
		}
	})

	t.Run("expired editor", func(t *testing.T) {
		execOrFatal(t, `UPDATE selector_editors SET expires_at = now() - interval '1 minute' WHERE id = $1`, e.ID)
		if _, err := s.UpdateSelectorEditorMarks(
			t.Context(),
			uid,
			page.ID,
			e.ID,
			firstPostMarks,
		); !errors.Is(
			err,
			feed.ErrEditorExpired,
		) {
			t.Errorf("got %v, want ErrEditorExpired", err)
		}
	})
}

func TestFeed_GetSelectorEditor(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	other, _ := provisionTestAccount(t, "other@example.com", "test#password$1234", "Pixel9a/Android", time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))
	opened := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
	must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, opened.ID, firstPostMarks))

	e, err := s.GetSelectorEditor(t.Context(), uid, page.ID, opened.ID)
	if err != nil {
		t.Fatalf("GetSelectorEditor: %v", err)
	}
	if e.ID != opened.ID || !e.ExpiresAt.Equal(opened.ExpiresAt) || e.Page != opened.Page {
		t.Errorf("got the editor %d expiring at %v, want the opened one %d expiring at %v with its page",
			e.ID, e.ExpiresAt, opened.ID, opened.ExpiresAt)
	}
	if d := cmp.Diff(firstPostMarks, e.Marks); d != "" {
		t.Errorf("marks mismatch:\n%s", d)
	}
	checkJSON(t, "./testdata/feed/page_without_feed_state.json", e.EditorState)

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
		execOrFatal(t, `UPDATE selector_editors SET expires_at = now() - interval '1 minute' WHERE id = $1`, opened.ID)
		if _, err := s.GetSelectorEditor(
			t.Context(),
			uid,
			page.ID,
			opened.ID,
		); !errors.Is(
			err,
			feed.ErrEditorExpired,
		) {
			t.Errorf("got %v, want ErrEditorExpired", err)
		}
	})
}

func TestFeed_CommitSelectorEditor(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	other, _ := provisionTestAccount(t, "other@example.com", "test#password$1234", "Pixel9a/Android", time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))

	t.Run("editor without selectors", func(t *testing.T) {
		e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
		if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); !errors.Is(err, feed.ErrNoSelectors) {
			t.Errorf("got %v, want ErrNoSelectors", err)
		}
	})

	t.Run("editor of another user", func(t *testing.T) {
		e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
		must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks))
		if err := s.CommitSelectorEditor(t.Context(), other, page.ID, e.ID); !errors.Is(err, feed.ErrEditorNotFound) {
			t.Errorf("got %v, want ErrEditorNotFound", err)
		}
	})

	t.Run("expired editor", func(t *testing.T) {
		e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
		must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks))
		execOrFatal(t, `UPDATE selector_editors SET expires_at = now() - interval '1 minute' WHERE id = $1`, e.ID)
		if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); !errors.Is(err, feed.ErrEditorExpired) {
			t.Errorf("got %v, want ErrEditorExpired", err)
		}
		if !editorRowExists(t, e.ID) {
			t.Error("the expired editor is deleted, want it kept")
		}
	})

	if page.HasSelectors {
		t.Fatal("HasSelectors = true before a commit, want false")
	}

	e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
	must(s.UpdateSelectorEditorMarks(t.Context(), uid, page.ID, e.ID, firstPostMarks))
	if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); err != nil {
		t.Fatalf("CommitSelectorEditor: %v", err)
	}
	var got, want any
	_ = json.Unmarshal(
		[]byte(scanValOrFatal[string](t, `SELECT extraction_rules FROM feeds WHERE id = $1`, page.ID)),
		&got,
	)
	_ = json.Unmarshal(must(os.ReadFile("./testdata/feed/page_without_feed_selectors.json")), &want)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("saved rules mismatch:\n%s", d)
	}
	if editorRowExists(t, e.ID) {
		t.Error("the committed editor is kept, want it deleted")
	}
	if again := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/")))); !again.HasSelectors {
		t.Error("HasSelectors = false after a commit, want true")
	}
	if err := s.CommitSelectorEditor(t.Context(), uid, page.ID, e.ID); !errors.Is(err, feed.ErrEditorNotFound) {
		t.Errorf("committing again: got %v, want ErrEditorNotFound", err)
	}
}

func TestFeed_CloseSelectorEditor(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	other, _ := provisionTestAccount(t, "other@example.com", "test#password$1234", "Pixel9a/Android", time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))
	e := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))

	if err := s.CloseSelectorEditor(t.Context(), other, page.ID, e.ID); !errors.Is(err, feed.ErrEditorNotFound) {
		t.Errorf("closing the editor of another user: got %v, want ErrEditorNotFound", err)
	}
	if !editorRowExists(t, e.ID) {
		t.Fatal("another user deleted the editor")
	}

	if err := s.CloseSelectorEditor(t.Context(), uid, page.ID, e.ID); err != nil {
		t.Fatalf("CloseSelectorEditor: %v", err)
	}
	if editorRowExists(t, e.ID) {
		t.Error("the closed editor is kept, want it deleted")
	}
	if err := s.CloseSelectorEditor(t.Context(), uid, page.ID, e.ID); !errors.Is(err, feed.ErrEditorNotFound) {
		t.Errorf("closing again: got %v, want ErrEditorNotFound", err)
	}
}

func TestFeed_DeleteExpiredSelectorEditors(t *testing.T) {
	t.Cleanup(testenv.TearDown)
	testenv.StubHTTP("blog.example.com", "/", "./testdata/feed/page_without_feed.html")
	uid := provisionDefaultTestAccount(t, time.Now())
	s := feed.NewService(testenv.DB(), scraper.NewService(stubServerAddr))
	page := must(s.Subscribe(t.Context(), uid, *must(url.Parse("http://blog.example.com/"))))
	expired := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
	valid := must(s.OpenSelectorEditor(t.Context(), uid, page.ID))
	execOrFatal(t, `UPDATE selector_editors SET expires_at = now() - interval '1 minute' WHERE id = $1`, expired.ID)

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
