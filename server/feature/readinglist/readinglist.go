package readinglist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/feature/user"
)

// ErrItemNotFound is returned when the item doesn't exist or doesn't belong
// to the calling user.
var ErrItemNotFound = errors.New("item not found")

// DeleteItem removes an item from the list.
// Reports false with no error if the id does not exist, true otherwise.
//
// TODO: Garbage-collect orphaned web clips in the background. Deleting the
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
	// TODO: Return a dedicated error for the case where the id doesn't exist
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
	// TODO: Return a dedicated error for the case where the id doesn't exist
	return it, err
}

// SaveWebClip adds a web clip specified by the URL to the reading list.
// Reports nil if succeeds.
//
// Note that this function immediately returns after creating a placeholder reading list item.
// It then tries fetching the clip itself asynchronously, and fills the
// placeholders with actual metadata.
func (s *Service) SaveWebClip(ctx context.Context, uid user.UserID, ln url.URL, title string) (SavedItem, error) {
	// TODO: Cleanup URL
	// TODO: Validate URL (schema, host)
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
		// TODO: Recover from panic
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

// fetchWebClip fills in the clip's metadata and content from the page.
// The metadata is saved even if the content can't be extracted, in which
// case an error is reported.
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

	page, err := scraper.Extract(res.Body, ln)
	if err != nil {
		return err
	}

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
	// Only overwrite the title and description when the page actually has
	// them; otherwise keep the existing (placeholder) values rather than
	// clobbering them with ''.
	_, err = tx.ExecContext(ctx, `
		UPDATE reading_list_items
		SET title = COALESCE(NULLIF($1, ''), title),
			description = COALESCE(NULLIF($2, ''), description)
		WHERE id = $3;
	`, page.Title, page.Description, rID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE web_clips
		SET title = COALESCE(NULLIF($1, ''), title),
			description = COALESCE(NULLIF($2, ''), description)
		WHERE id = $3;
	`, page.Title, page.Description, clipID)
	if err != nil {
		return err
	}
	if page.Content == "" {
		// Save the metadata anyway; tryFetchWebClip marks the clip as failed.
		if err := tx.Commit(); err != nil {
			return err
		}
		return fmt.Errorf("failed to extract content")
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE web_clips
		SET content = $1, fetch_status = 'done'
		WHERE id = $2;
	`, page.Content, clipID)
	if err != nil {
		return err
	}
	return tx.Commit()
}
