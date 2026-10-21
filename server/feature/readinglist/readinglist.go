package readinglist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"codeberg.org/readeck/go-readability/v2"
	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/microcosm-cc/bluemonday"
)

// ErrItemNotFound is returned when the item doesn't exist or doesn't belong
// to the calling user.
var ErrItemNotFound = errors.New("item not found")

// DeleteItem removes an item from the list.
// Reports false with no error if the id does not exist, true otherwise.
//
// TODO(#139): Garbage-collect orphaned web clips in the background. Deleting the
// join row here leaves the backing web_clips row unreachable (it can only be
// reached via a reading_list_items row), so a background job should periodically
// delete web_clips that have no referencing reading_list_items.
func (s *Service) DeleteItem(ctx context.Context, uid user.UserID, id int) (bool, error) {
	res, err := s.DB.ExecContext(ctx, `
		DELETE FROM reading_list_items
		WHERE id = $1 AND user_id = $2;
	`, id, uid)
	if err != nil {
		return false, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return false, err
	} else {
		return n != 0, nil
	}
}

func (s *Service) ArchiveItem(ctx context.Context, uid user.UserID, id int) error {
	return s.setItemArchivedStatus(ctx, uid, id, true)
}

func (s *Service) UnarchiveItem(ctx context.Context, uid user.UserID, id int) error {
	return s.setItemArchivedStatus(ctx, uid, id, false)
}

func (s *Service) setItemArchivedStatus(ctx context.Context, uid user.UserID, id int, archived bool) error {
	res, err := s.DB.ExecContext(ctx, `
		UPDATE reading_list_items
		SET archived = $1
		WHERE id = $2 AND user_id = $3;
	`, archived, id, uid)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrItemNotFound
	}
	return nil
}

// SavedItem is the reading list item created by a save operation.
type SavedItem struct {
	ID          int
	ResourceID  int
	Kind        string
	Title       string
	Description *string
	SavedAt     time.Time
}

func (s *Service) SaveFeedEntry(ctx context.Context, uid user.UserID, id int) (SavedItem, error) {
	var it SavedItem
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO reading_list_items (kind, feed_entry_id, title, description, user_id)
		SELECT 'feed_entry', id, title, description, $2
		FROM feed_entries
		WHERE id = $1
		RETURNING id, feed_entry_id, kind, title, description, saved_at;
	`, id, uid).Scan(&it.ID, &it.ResourceID, &it.Kind, &it.Title, &it.Description, &it.SavedAt)
	// TODO(#114): Return a dedicated error for the case where the id doesn't exist
	return it, err
}

func (s *Service) SaveWebClipByID(ctx context.Context, uid user.UserID, id int) (SavedItem, error) {
	// reading_list_items.title is NOT NULL but web_clips.title is nullable,
	// so coalesce to keep the placeholder well-formed.
	var it SavedItem
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO reading_list_items (kind, web_clip_id, title, description, user_id)
		SELECT 'web_clip', id, COALESCE(title, ''), description, $2
		FROM web_clips
		WHERE id = $1
		RETURNING id, web_clip_id, kind, title, description, saved_at;
	`, id, uid).Scan(&it.ID, &it.ResourceID, &it.Kind, &it.Title, &it.Description, &it.SavedAt)
	// TODO(#114): Return a dedicated error for the case where the id doesn't exist
	return it, err
}

// SaveWebClip adds a web clip specified by the URL to the reading list.
// Reports nil if succeeds.
//
// Note that this function immediately returns after creating a placeholder reading list item.
// It then tries fetching the clip itself asynchronously, and fills the
// placeholders with actual metadata.
func (s *Service) SaveWebClip(ctx context.Context, uid user.UserID, ln url.URL, title string) (SavedItem, error) {
	// TODO(#115): Cleanup URL
	// TODO(#115): Validate URL (schema, host)
	var it SavedItem
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return it, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Println(err)
		}
	}()
	var clipID int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO web_clips (url)
		VALUES ($1)
		RETURNING id;
	`, ln.String()).Scan(&clipID)
	if err != nil {
		return it, err
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO reading_list_items (kind, web_clip_id, title, user_id)
		VALUES ('web_clip', $1, $2, $3)
		RETURNING id, web_clip_id, kind, title, description, saved_at;
	`, clipID, title, uid).Scan(&it.ID, &it.ResourceID, &it.Kind, &it.Title, &it.Description, &it.SavedAt)
	if err != nil {
		return it, err
	}
	if err = tx.Commit(); err != nil {
		return it, err
	}
	go func() {
		// TODO(#136): Recover from panic
		err := s.tryFetchWebClip(it.ID, clipID, ln)
		if err != nil {
			fmt.Println(err)
		}
	}()
	return it, nil
}

func (s *Service) tryFetchWebClip(rID, clipID int, ln url.URL) error {
	err := s.fetchWebClip(rID, clipID, ln)
	if err == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, dbErr := s.DB.ExecContext(ctx, `
		UPDATE web_clips
		SET fetch_status = 'failed'
		WHERE id = $1;
	`, clipID)
	if dbErr != nil {
		return errors.Join(err, dbErr)
	}
	return err
}

// TODO(#132): DRY scraping logic
func (s *Service) fetchWebClip(rID, clipID int, ln url.URL) error {
	fmt.Printf("Fetching reading list clip from %s\n", ln.String())
	res, err := s.scraper.Fetch(context.Background(), ln)
	if err != nil {
		return err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP GET failed with status code %d", res.StatusCode)
	}

	buf := bluemonday.UGCPolicy().AllowElements("head", "title").SanitizeReader(res.Body)
	if buf.Len() == 0 {
		return fmt.Errorf("the body is empty")
	}

	baseUrl := new(ln)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Println(err)
		}
	}()
	// Only overwrite the title when the fetch actually extracted one; otherwise
	// keep the existing (placeholder) title rather than clobbering it with ''.
	if t := article.Title(); t != "" {
		_, err = tx.ExecContext(ctx, `
			UPDATE reading_list_items
			SET title = $1
			WHERE id = $2;
		`, t, rID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE web_clips
			SET title = $1, content = $2, fetch_status = 'done'
			WHERE id = $3;
		`, t, content, clipID)
		if err != nil {
			return err
		}
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE web_clips
			SET content = $1, fetch_status = 'done'
			WHERE id = $2;
		`, content, clipID)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
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
