package feed

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/net/html"
)

// A selector editor is a temporary editor on the server in which a user edits
// the post selectors of an unstructured feed. It holds the simplified page of
// the feed, the marks of the user, and the selectors generated from them. The
// client sends all marks with each update, and the server generates the
// selectors again and returns what they find, so that the client never runs
// selectors. An editor belongs to the user who opened it, and expires 48
// hours after it opens, so that selectors built on a stale layout of the page
// are not saved.

var (
	ErrEditorNotFound = errors.New("selector editor not found")
	ErrEditorExpired  = errors.New("selector editor expired")
	ErrInvalidMarks   = errors.New("invalid marks")
	ErrNoSelectors    = errors.New("no selectors")
)

// LabeledPostIDs is a labeled post by node IDs (see withNodeIDs): the ID of
// the post node under "box", and the ID of the node of each labeled field
// under the name of the field.
type LabeledPostIDs map[string]int

// Marks are the marks of a user in a selector editor: the posts that the user
// labeled, and the host and path of each link that the user excluded (see
// generateSelectors).
type Marks struct {
	LabeledPosts  []LabeledPostIDs `json:"labeled_posts"`
	ExcludedLinks []string         `json:"excluded_links"`
}

// MatchedPost is a post that the selectors of an editor find: the node IDs of
// the post and of its fields, as in LabeledPostIDs, and the field values.
type MatchedPost struct {
	IDs    LabeledPostIDs
	Values ExtractedPost
}

func (p MatchedPost) MarshalJSON() ([]byte, error) {
	j := map[string]any{"values": p.Values}
	for k, id := range p.IDs {
		j[k] = id
	}
	return json.Marshal(j)
}

// ExcludedPost is a node that would be a post if its link were not excluded,
// with the excluded link that excludes it.
type ExcludedPost struct {
	Box  int    `json:"box"`
	Link string `json:"link"`
}

// EditorState is what the selectors of an editor find on its page: the
// matched posts and the excluded posts in document order, and the posts that
// ExtractPosts extracts, which the polling job will save.
type EditorState struct {
	MatchedPosts  []MatchedPost   `json:"matched_posts"`
	ExcludedPosts []ExcludedPost  `json:"excluded_posts"`
	Preview       []ExtractedPost `json:"preview"`
}

// SelectorEditor is a selector editor as the client sees it: its page with
// node IDs, the marks, and what the selectors generated from them find.
type SelectorEditor struct {
	ID        int       `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
	Page      string    `json:"page"`
	Marks
	EditorState
}

// withNodeIDs returns page, a simplified page, with the node ID of each
// element in its data-id attribute. The node ID of a node is its position in
// the page nodes of the page in document order (see pageIndex). A text node
// cannot carry an attribute, so a reader of the page computes the IDs of the
// texts as it walks the page nodes.
func withNodeIDs(page string) (string, error) {
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return "", err
	}
	t := newPageIndex(pageBody(doc))
	for i, n := range t.nodes {
		if n.Type == html.ElementNode {
			n.Attr = append(n.Attr, html.Attribute{Key: "data-id", Val: strconv.Itoa(i)})
		}
	}
	var b bytes.Buffer
	if err := html.Render(&b, doc); err != nil {
		return "", err
	}
	return b.String(), nil
}

// parseEditorPage parses page, a simplified page with node IDs (see
// withNodeIDs). It fails when the data-id of an element differs from the
// position of the element, so that an ID never resolves to a node other than
// the one that it was given to.
func parseEditorPage(page string) (*html.Node, *pageIndex, error) {
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return nil, nil, err
	}
	t := newPageIndex(pageBody(doc))
	for i, n := range t.nodes {
		if n.Type == html.ElementNode && attr(n, "data-id") != strconv.Itoa(i) {
			return nil, nil, fmt.Errorf("node %d has the data-id %q", i, attr(n, "data-id"))
		}
	}
	return doc, t, nil
}

// labeledPosts returns the labeled posts of marks as nodes of t, and the
// excluded links as a set. It fails with ErrInvalidMarks when an ID is not a
// node of t, when a post is the body or not an element, when a field node is
// outside its post or is a node that the field does not accept, when a post
// has no title or link, or when an excluded link is the link of a post.
func (t *pageIndex) labeledPosts(marks Marks) ([]labeledPost, map[string]bool, error) {
	nodeOf := func(id int) (*html.Node, error) {
		if id < 0 || id >= len(t.nodes) {
			return nil, fmt.Errorf("%w: no node %d", ErrInvalidMarks, id)
		}
		return t.nodes[id], nil
	}
	excluded := map[string]bool{}
	for _, l := range marks.ExcludedLinks {
		excluded[l] = true
	}
	var posts []labeledPost
	for _, ids := range marks.LabeledPosts {
		box, ok := ids["box"]
		if !ok {
			return nil, nil, fmt.Errorf("%w: a post has no box", ErrInvalidMarks)
		}
		post, err := nodeOf(box)
		if err != nil {
			return nil, nil, err
		}
		if post == t.body || post.Type != html.ElementNode {
			return nil, nil, fmt.Errorf("%w: node %d cannot be a post", ErrInvalidMarks, box)
		}
		p := labeledPost{node: post, fields: make(map[PostField]*html.Node)}
		for k, id := range ids {
			f := PostField(k)
			switch f {
			case "box":
				continue
			case PostTitle, PostBody, PostLink, PostThumbnail, PostPubDate:
			default:
				return nil, nil, fmt.Errorf("%w: unknown field %q", ErrInvalidMarks, k)
			}
			n, err := nodeOf(id)
			if err != nil {
				return nil, nil, err
			}
			if !t.contains(post, n) || !f.accepts(n) {
				return nil, nil, fmt.Errorf("%w: node %d cannot be the %s of post %d", ErrInvalidMarks, id, f, box)
			}
			p.fields[f] = n
		}
		if p.fields[PostTitle] == nil || p.fields[PostLink] == nil {
			return nil, nil, fmt.Errorf("%w: post %d has no title or link", ErrInvalidMarks, box)
		}
		u, err := url.Parse(linkOf(p.fields[PostLink]))
		if err == nil && excluded[u.Host+u.Path] {
			return nil, nil, fmt.Errorf("%w: the link of post %d is excluded", ErrInvalidMarks, box)
		}
		posts = append(posts, p)
	}
	return posts, excluded, nil
}

// editorState returns what selectors find in doc, the page of t. A node that
// several selectors find takes its fields from the first one.
func (t *pageIndex) editorState(doc *html.Node, selectors []PostSelector) EditorState {
	s := EditorState{MatchedPosts: []MatchedPost{}, ExcludedPosts: []ExcludedPost{}}
	matched := map[*html.Node]bool{}
	excluded := map[*html.Node]bool{}
	for i := range selectors {
		m := &selectors[i]
		exclude := m.compileExclude()
		for _, n := range m.candidates(t.body) {
			if link, ok := m.excludedBy(n, exclude); ok {
				if !excluded[n] {
					excluded[n] = true
					s.ExcludedPosts = append(s.ExcludedPosts, ExcludedPost{Box: t.index[n], Link: link})
				}
				continue
			}
			if matched[n] {
				continue
			}
			matched[n] = true
			p := MatchedPost{IDs: LabeledPostIDs{"box": t.index[n]}, Values: make(ExtractedPost)}
			for f := range m.Fields() {
				if v := m.resolveField(n, f); v != nil {
					p.IDs[string(f)] = t.index[v]
					if val, ok := f.valueOf(v); ok {
						p.Values[f] = val
					}
				}
			}
			s.MatchedPosts = append(s.MatchedPosts, p)
		}
	}
	slices.SortFunc(s.MatchedPosts, func(a, b MatchedPost) int { return a.IDs["box"] - b.IDs["box"] })
	slices.SortFunc(s.ExcludedPosts, func(a, b ExcludedPost) int { return a.Box - b.Box })
	s.Preview = ExtractPosts(doc, selectors)
	if s.Preview == nil {
		s.Preview = []ExtractedPost{}
	}
	return s
}

// generate returns the selectors generated from posts and excluded, as JSON,
// and what they find in doc, the page of t.
func (t *pageIndex) generate(
	doc *html.Node,
	posts []labeledPost,
	excluded map[string]bool,
) (string, EditorState, error) {
	selectors := t.generateSelectors(posts, excluded)
	if selectors == nil {
		selectors = []PostSelector{}
	}
	raw, err := json.Marshal(selectors)
	if err != nil {
		return "", EditorState{}, err
	}
	return string(raw), t.editorState(doc, selectors), nil
}

// idsOf returns posts by node IDs.
func (t *pageIndex) idsOf(posts []labeledPost) []LabeledPostIDs {
	ids := []LabeledPostIDs{}
	for _, p := range posts {
		j := LabeledPostIDs{"box": t.index[p.node]}
		for f, n := range p.fields {
			j[string(f)] = t.index[n]
		}
		ids = append(ids, j)
	}
	return ids
}

// OpenSelectorEditor opens a new selector editor of the unstructured feed
// feedID for the user uid. It fetches the page of the feed, and labels the
// posts that the saved selectors of the feed find, which keeps them found when
// the user adds new marks, since the saved selectors do not keep the marks of
// the user. The excluded links start as the ones of the saved selectors.
func (s *Service) OpenSelectorEditor(ctx context.Context, uid user.UserID, feedID int) (SelectorEditor, error) {
	var fu string
	var saved sql.NullString
	var unstructured bool
	err := s.DB.QueryRowContext(ctx, `
		SELECT url, selectors, (url = site_url) IS TRUE FROM feeds WHERE id = $1;
	`, feedID).Scan(&fu, &saved, &unstructured)
	if errors.Is(err, sql.ErrNoRows) {
		return SelectorEditor{}, ErrFeedNotFound
	} else if err != nil {
		return SelectorEditor{}, err
	}
	if !unstructured {
		return SelectorEditor{}, ErrNotUnstructured
	}
	var selectors []PostSelector
	if saved.Valid {
		if err := json.Unmarshal([]byte(saved.String), &selectors); err != nil {
			return SelectorEditor{}, err
		}
	}
	u, err := url.Parse(fu)
	if err != nil {
		return SelectorEditor{}, err
	}
	raw, _, err := s.fetch(ctx, *u)
	if err != nil {
		return SelectorEditor{}, err
	}
	simplified, err := SimplifyPage(raw, fu)
	if err != nil {
		return SelectorEditor{}, err
	}
	page, err := withNodeIDs(simplified)
	if err != nil {
		return SelectorEditor{}, err
	}
	doc, t, err := parseEditorPage(page)
	if err != nil {
		return SelectorEditor{}, err
	}

	posts := t.labelFoundPosts(selectors)
	excluded := map[string]bool{}
	for _, m := range selectors {
		for _, l := range m.Exclude {
			excluded[l] = true
		}
	}
	e := SelectorEditor{
		Page:          page,
		LabeledPosts:  t.idsOf(posts),
		ExcludedLinks: slices.Sorted(maps.Keys(excluded)),
	}
	if e.ExcludedLinks == nil {
		e.ExcludedLinks = []string{}
	}
	generated, state, err := t.generate(doc, posts, excluded)
	if err != nil {
		return SelectorEditor{}, err
	}
	e.EditorState = state
	labeled, err := json.Marshal(e.LabeledPosts)
	if err != nil {
		return SelectorEditor{}, err
	}
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO selector_editors
			(user_id, feed_id, simplified_page, labeled_posts, excluded_links, selectors, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, now() + interval '48 hours')
		RETURNING id, expires_at;
	`, uid, feedID, page, string(labeled), e.ExcludedLinks, generated).Scan(&e.ID, &e.ExpiresAt)
	if err != nil {
		return SelectorEditor{}, err
	}
	return e, nil
}

// loadSelectorEditor returns the page and the marks of the editor id of the
// user uid on the feed feedID, and when the editor expires. It fails with
// ErrEditorNotFound when there is no such editor, and with ErrEditorExpired
// when it expired.
func (s *Service) loadSelectorEditor(
	ctx context.Context,
	uid user.UserID,
	feedID, id int,
) (string, Marks, time.Time, error) {
	var page, labeled string
	var marks Marks
	var expiresAt time.Time
	var expired bool
	err := s.DB.QueryRowContext(ctx, `
		SELECT simplified_page, labeled_posts, excluded_links, expires_at, expires_at <= now()
		FROM selector_editors
		WHERE id = $1 AND user_id = $2 AND feed_id = $3;
	`, id, uid, feedID).Scan(
		&page, &labeled, pgtype.NewMap().SQLScanner(&marks.ExcludedLinks), &expiresAt, &expired,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return "", Marks{}, time.Time{}, ErrEditorNotFound
	} else if err != nil {
		return "", Marks{}, time.Time{}, err
	}
	if expired {
		return "", Marks{}, time.Time{}, ErrEditorExpired
	}
	if err := json.Unmarshal([]byte(labeled), &marks.LabeledPosts); err != nil {
		return "", Marks{}, time.Time{}, err
	}
	return page, marks, expiresAt, nil
}

// GetSelectorEditor returns the editor id of the user uid on the feed
// feedID with its stored marks, such as after the app was killed while the
// editor was open. The page is not fetched again, since the marks refer to the
// nodes of the stored page. Getting it does not extend the expiry.
func (s *Service) GetSelectorEditor(ctx context.Context, uid user.UserID, feedID, id int) (SelectorEditor, error) {
	page, marks, expiresAt, err := s.loadSelectorEditor(ctx, uid, feedID, id)
	if err != nil {
		return SelectorEditor{}, err
	}
	doc, t, err := parseEditorPage(page)
	if err != nil {
		return SelectorEditor{}, err
	}
	posts, excluded, err := t.labeledPosts(marks)
	if err != nil {
		return SelectorEditor{}, err
	}
	_, state, err := t.generate(doc, posts, excluded)
	if err != nil {
		return SelectorEditor{}, err
	}
	return SelectorEditor{ID: id, ExpiresAt: expiresAt, Page: page, Marks: marks, EditorState: state}, nil
}

// UpdateSelectorEditorMarks replaces the marks of the editor id of the user
// uid on the feed feedID with marks, generates the selectors again, and
// returns what they find. It fails with ErrInvalidMarks when marks are not
// valid on the page of the editor (see labeledPosts).
func (s *Service) UpdateSelectorEditorMarks(
	ctx context.Context,
	uid user.UserID,
	feedID, id int,
	marks Marks,
) (EditorState, error) {
	page, _, _, err := s.loadSelectorEditor(ctx, uid, feedID, id)
	if err != nil {
		return EditorState{}, err
	}
	// The page is parsed on each update rather than cached, since even a
	// large page parses in a few milliseconds.
	doc, t, err := parseEditorPage(page)
	if err != nil {
		return EditorState{}, err
	}
	posts, excluded, err := t.labeledPosts(marks)
	if err != nil {
		return EditorState{}, err
	}
	generated, state, err := t.generate(doc, posts, excluded)
	if err != nil {
		return EditorState{}, err
	}
	if marks.LabeledPosts == nil {
		marks.LabeledPosts = []LabeledPostIDs{}
	}
	if marks.ExcludedLinks == nil {
		marks.ExcludedLinks = []string{}
	}
	labeled, err := json.Marshal(marks.LabeledPosts)
	if err != nil {
		return EditorState{}, err
	}
	res, err := s.DB.ExecContext(ctx, `
		UPDATE selector_editors
		SET labeled_posts = $4, excluded_links = $5, selectors = $6
		WHERE id = $1 AND user_id = $2 AND feed_id = $3;
	`, id, uid, feedID, string(labeled), marks.ExcludedLinks, generated)
	if err != nil {
		return EditorState{}, err
	}
	// The editor was deleted after it was loaded.
	if n, err := res.RowsAffected(); err != nil {
		return EditorState{}, err
	} else if n == 0 {
		return EditorState{}, ErrEditorNotFound
	}
	return state, nil
}

// CommitSelectorEditor saves the selectors of the editor id of the user uid
// to the feed feedID, and deletes the editor. The selectors are shared by all
// subscribers, as the feed is, and the last commit wins. It fails with
// ErrNoSelectors when the editor has no selectors. An expired editor is kept,
// as it is until DeleteExpiredSelectorEditors deletes it.
func (s *Service) CommitSelectorEditor(ctx context.Context, uid user.UserID, feedID, id int) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var selectors string
	var expired bool
	err = tx.QueryRowContext(ctx, `
		SELECT selectors, expires_at <= now()
		FROM selector_editors
		WHERE id = $1 AND user_id = $2 AND feed_id = $3
		FOR UPDATE;
	`, id, uid, feedID).Scan(&selectors, &expired)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrEditorNotFound
	} else if err != nil {
		return err
	}
	if expired {
		return ErrEditorExpired
	}
	var ms []json.RawMessage
	if err := json.Unmarshal([]byte(selectors), &ms); err != nil {
		return err
	}
	if len(ms) == 0 {
		return ErrNoSelectors
	}
	// The open checked that the feed is unstructured, and a feed never
	// changes its kind.
	if _, err := tx.ExecContext(ctx, `UPDATE feeds SET selectors = $1 WHERE id = $2;`, selectors, feedID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM selector_editors WHERE id = $1;`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// CloseSelectorEditor deletes the editor id of the user uid on the feed
// feedID without saving its selectors. An expired editor is deleted too.
func (s *Service) CloseSelectorEditor(ctx context.Context, uid user.UserID, feedID, id int) error {
	res, err := s.DB.ExecContext(ctx, `
		DELETE FROM selector_editors WHERE id = $1 AND user_id = $2 AND feed_id = $3;
	`, id, uid, feedID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrEditorNotFound
	}
	return nil
}

// DeleteExpiredSelectorEditors deletes every expired selector editor, such as
// one whose close request never reached the server, and returns how many it
// deleted.
func DeleteExpiredSelectorEditors(ctx context.Context, db *sql.DB) (int64, error) {
	res, err := db.ExecContext(ctx, `DELETE FROM selector_editors WHERE expires_at < now();`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
