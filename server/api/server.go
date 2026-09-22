package api

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fujidaiti/paperdoll/server"
	"github.com/fujidaiti/paperdoll/server/cfg"
	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/fujidaiti/paperdoll/server/feature/readinglist"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/fujidaiti/paperdoll/server/feature/user"
	"github.com/fujidaiti/paperdoll/server/infra"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func StartServer(ctx context.Context) {
	dsn := os.Getenv("DB_DSN")
	if len(dsn) == 0 {
		panic("DB_DSN is required.")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	defer func() {
		fmt.Println("Closing DB...")
		if err := db.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if err := db.Ping(); err != nil {
		panic(err)
	}

	config, err := cfg.Read()
	if err != nil {
		panic(err)
	}

	emailSender, err := infra.NewEmailSenderFrom(config)
	if err != nil {
		panic(err)
	}

	srv := NewServer(db, nil, emailSender)
	srv.BaseContext = func(_ net.Listener) context.Context { return ctx }
	defer func() {
		fmt.Println("Shutting down API server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			fmt.Println("Graceful shutdown failed:")
			fmt.Println(err)
		}
	}()

	c := make(chan error, 1)
	go func() {
		fmt.Printf("API server started on %s\n", srv.Addr)
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			c <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-c:
		fmt.Println(err)
	}
}

func NewServer(db *sql.DB, httpProxy *url.URL, emailSender infra.EmailSender) *http.Server {
	scrp := scraper.NewService(httpProxy)
	h := &Handler{
		DB:                 db,
		UserService:        user.NewService(db, emailSender),
		ReadingListService: readinglist.NewService(db, scrp),
		FeedService:        feed.NewService(db, scrp),
		ScraperService:     scrp,
	}

	authorized := http.NewServeMux()
	authorized.HandleFunc("GET /newspapers/today", h.getTodaysNewspaper)
	authorized.HandleFunc("GET /feeds", h.getFeeds)
	authorized.HandleFunc("PUT /feeds", h.subscribeToFeed)
	authorized.HandleFunc("GET /feeds/search", h.searchFeeds)
	authorized.HandleFunc("GET /feeds/{id}", h.getFeed)
	authorized.HandleFunc("GET /feeds/{id}/timeline", h.getFeedTimeline)
	authorized.HandleFunc("GET /feed-entries/{id}", h.getFeedEntry)
	authorized.HandleFunc("GET /web-clips/{id}", h.getWebClip)
	authorized.HandleFunc("POST /reading-list", h.saveToReadingList)
	authorized.HandleFunc("GET /reading-list", h.getReadingList)
	authorized.HandleFunc("GET /reading-list/archived", h.getArchivedReadingList)
	authorized.HandleFunc("DELETE /reading-list/{id}", h.deleteReadingListItem)
	authorized.HandleFunc("PATCH /reading-list/{id}", h.setReadingListItemArchivedStatus)
	authorized.HandleFunc("POST /signout", h.signOut)

	mux := http.NewServeMux()
	mux.Handle("/", AuthMiddleware(authorized, h.UserService))
	mux.HandleFunc("GET /health", h.getHealth)
	mux.HandleFunc("POST /signup", h.signUp)
	mux.HandleFunc("POST /signup/verify-email", h.verifySignUpEmail)
	mux.HandleFunc("POST /signup/resend-verification", h.resendSignUpVerification)
	mux.HandleFunc("POST /signin", h.signIn)

	return &http.Server{
		Addr:    ":8080",
		Handler: mux,
		// Slowloris attack prevention
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// TODO: rename to handler (lowercase)
type Handler struct {
	DB                 *sql.DB
	UserService        *user.Service
	ReadingListService *readinglist.Service
	FeedService        *feed.Service
	ScraperService     *scraper.Service
}

type getHealthResponse struct {
	Version string `json:"version"`
}

func (h *Handler) getHealth(w http.ResponseWriter, _ *http.Request) {
	jres, err := json.Marshal(getHealthResponse{Version: server.Version})
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct a JSON response.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

type getTodaysNewspaperResponse struct {
	ID          int       `json:"id"`
	PublishedAt time.Time `json:"published_at"`
	Stories     []stories `json:"stories"`
	NextCursor  *string   `json:"next_cursor,omitempty"`
}

// readLater describes an item's presence in the reading list. It is nil (and
// omitted from responses) when the item is not saved.
type readLater struct {
	ID       int  `json:"id"`
	Archived bool `json:"archived"`
}

type stories struct {
	ID          int        `json:"id"`
	ResourceID  int        `json:"resource_id"`
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Source      *string    `json:"source,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	ReadLater   *readLater `json:"read_later,omitempty"`
}

func (h *Handler) getTodaysNewspaper(w http.ResponseWriter, r *http.Request) {
	// TODO: Design a better cursor (e.g. priority)
	var cursor int
	if c := r.URL.Query().Get("after"); c != "" {
		var err error
		cursor, err = strconv.Atoi(c)
		if err != nil {
			serverError(w, http.StatusBadRequest, "Malformed cursor")
			return
		}
	}

	ctx := r.Context()
	uid, ok := UserIDFromContext(ctx)
	if !ok {
		serverError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res := getTodaysNewspaperResponse{}
	err := h.DB.QueryRowContext(ctx, `
		SELECT id, published_at
		FROM newspapers
		WHERE user_id = $1
		ORDER BY published_at DESC
		LIMIT 1;
	`, uid).Scan(&res.ID, &res.PublishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		serverError(w, http.StatusNotFound, "No newspaper found.")
		return
	} else if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to fetch today's newspaper.")
		return
	}

	// TODO: Replace the correlated subquery with a LEFT JOIN on
	// reading_list_items once reading_list_items.feed_entry_id is unique. The
	// subquery + LIMIT 1 is a workaround for the currently non-unique column.
	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, feed_entry_id, title, description, source, published_at,
			(SELECT id FROM reading_list_items
				WHERE feed_entry_id = stories.feed_entry_id AND user_id = $4
				LIMIT 1) as reading_list_item_id,
			(SELECT archived FROM reading_list_items
				WHERE feed_entry_id = stories.feed_entry_id AND user_id = $4
				LIMIT 1)
		FROM stories
		WHERE newspaper_id = $1 AND id > $2
		ORDER BY id ASC
		LIMIT $3;
	`, res.ID, cursor, paginationSize+1, uid)
	if err != nil {
		fmt.Println(err)
		serverError(
			w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to fetch stories for the newspaper (ID=%d).", res.ID),
		)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	for rows.Next() {
		a := stories{Kind: "feed_entry"}
		var rlID *int
		var rlArchived *bool
		err := rows.Scan(
			&a.ID,
			&a.ResourceID,
			&a.Title,
			&a.Description,
			&a.Source,
			&a.PublishedAt,
			&rlID,
			&rlArchived,
		)
		if err != nil {
			serverError(w, http.StatusInternalServerError, "Failed to parse a story.")
			return
		}
		if rlID != nil {
			a.ReadLater = &readLater{ID: *rlID, Archived: rlArchived != nil && *rlArchived}
		}
		res.Stories = append(res.Stories, a)
	}
	if err := rows.Err(); err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to parse stories.")
		return
	}

	if len(res.Stories) > paginationSize {
		res.Stories = res.Stories[:paginationSize]
		last := res.Stories[len(res.Stories)-1]
		res.NextCursor = new(strconv.Itoa(last.ID))
	}

	jres, err := json.Marshal(res)
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct a JSON response.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

type feedEntry struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	FeedID      int        `json:"feed_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Content     *string    `json:"content,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	SnapshotAt  time.Time  `json:"snapshot_at"`
}

type getFeedEntryResponse struct {
	feedEntry
	ReadLater *readLater `json:"read_later,omitempty"`
}

func (h *Handler) getFeedEntry(w http.ResponseWriter, r *http.Request) {
	rawId := r.PathValue("id")
	id, err := strconv.Atoi(rawId)
	if err != nil {
		serverError(w, http.StatusBadRequest, fmt.Sprintf("Invalid entry ID: %s", rawId))
		return
	}

	ctx := r.Context()
	uid, ok := UserIDFromContext(ctx)
	if !ok {
		serverError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res := getFeedEntryResponse{}
	var rlID *int
	var rlArchived *bool
	err = h.DB.QueryRowContext(ctx, `
		SELECT id, feed_id, url, title, description, content, snapshot_at, published_at,
			(SELECT id FROM reading_list_items
				WHERE feed_entry_id = feed_entries.id AND user_id = $2
				LIMIT 1),
			(SELECT archived FROM reading_list_items
				WHERE feed_entry_id = feed_entries.id AND user_id = $2
				LIMIT 1)
		FROM feed_entries
		WHERE id = $1;
	`, id, uid).Scan(
		&res.ID, &res.FeedID, &res.URL, &res.Title, &res.Description, &res.Content,
		&res.SnapshotAt, &res.PublishedAt, &rlID, &rlArchived,
	)
	if errors.Is(err, sql.ErrNoRows) {
		serverError(w, http.StatusNotFound, "Entry not found.")
		return
	} else if err != nil {
		serverError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to fetch entry by ID=%d", id),
		)
		return
	}
	if rlID != nil {
		res.ReadLater = &readLater{ID: *rlID, Archived: rlArchived != nil && *rlArchived}
	}

	jres, err := json.Marshal(res)
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct a JSON response.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

type feedAttrsSchema struct {
	URL         string `json:"url"`
	SiteURL     string `json:"site_url,omitempty"`
	IconURL     string `json:"icon_url,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	// PostGroups is only set by the search, and only for an HTML page that
	// publishes no feed.
	PostGroups []postGroupSchema `json:"post_groups,omitempty"`
}

type postGroupSchema struct {
	ID       int                        `json:"id"`
	Selector string                     `json:"selector"`
	Count    int                        `json:"count"`
	Sampled  int                        `json:"sampled"`
	Links    []attributeCandidateSchema `json:"links"`
	Texts    []attributeCandidateSchema `json:"texts"`
	Images   []attributeCandidateSchema `json:"images"`
}

type attributeCandidateSchema struct {
	Selector string                 `json:"selector"`
	Values   []attributeValueSchema `json:"values"`
}

type attributeValueSchema struct {
	Value string `json:"value,omitempty"`
	Alt   string `json:"alt,omitempty"`
}

// sampleSize is how many items of a group the response describes, before the
// rule in postGroups adds more.
const sampleSize = 3

// postGroups turns the enumeration, which answers per item, into the response,
// which answers per selector. One row of the response is one key, and it holds
// the value that key produces in each sampled item, so the client can show a
// preview of item i by reading index i of every row.
func postGroups(gs []feed.Group) []postGroupSchema {
	out := make([]postGroupSchema, 0, len(gs))
	for i, g := range gs {
		// The sample starts with the first items and then grows to cover every
		// key of the group. Without the second part, a key that only the eighth
		// item carries, such as a description that most posts leave out, would
		// be a row of empty values, and the user could neither tell what it is
		// nor decide whether to pick it.
		var sample []int
		carried := map[string]bool{}
		for j, p := range g.Posts {
			take := j < sampleSize
			for _, a := range attributesOf(p) {
				if !carried[a.Selector] {
					take = true
				}
			}
			if !take {
				continue
			}
			for _, a := range attributesOf(p) {
				carried[a.Selector] = true
			}
			sample = append(sample, j)
		}

		s := postGroupSchema{
			ID:       i,
			Selector: g.Selector,
			Count:    len(g.Posts),
			Sampled:  len(sample),
		}
		for _, f := range []struct {
			pick func(feed.PostCandidate) []feed.Attribute
			into *[]attributeCandidateSchema
		}{
			{func(p feed.PostCandidate) []feed.Attribute { return p.Links }, &s.Links},
			{func(p feed.PostCandidate) []feed.Attribute { return p.Texts }, &s.Texts},
			{func(p feed.PostCandidate) []feed.Attribute { return p.Images }, &s.Images},
		} {
			// The rows follow the order in which the keys first appear over
			// all the items, not only the sampled ones.
			var keys []string
			seen := map[string]bool{}
			for _, p := range g.Posts {
				for _, a := range f.pick(p) {
					if !seen[a.Selector] {
						seen[a.Selector] = true
						keys = append(keys, a.Selector)
					}
				}
			}
			rows := make([]attributeCandidateSchema, 0, len(keys))
			for _, k := range keys {
				row := attributeCandidateSchema{
					Selector: k,
					Values:   make([]attributeValueSchema, 0, len(sample)),
				}
				for _, j := range sample {
					// An item that does not carry the key produces an empty
					// entry rather than a shorter array, so that every row has
					// the same length and index i is always item i.
					var v attributeValueSchema
					for _, a := range f.pick(g.Posts[j]) {
						if a.Selector == k {
							v = attributeValueSchema{Value: a.Value, Alt: a.Alt}
							break
						}
					}
					row.Values = append(row.Values, v)
				}
				rows = append(rows, row)
			}
			*f.into = rows
		}
		out = append(out, s)
	}
	return out
}

func attributesOf(p feed.PostCandidate) []feed.Attribute {
	return slices.Concat(p.Links, p.Texts, p.Images)
}

type feedSchema struct {
	ID int `json:"id"`
	feedAttrsSchema
}

type getFeedsResBody struct {
	Feeds      []feedSchema `json:"feeds"`
	NextCursor *string      `json:"next_cursor,omitempty"`
}

const getFeedsSortKeyLen = 5

func (h *Handler) getFeeds(w http.ResponseWriter, r *http.Request) {
	var cursor *paginationCusor[string]
	if c := r.URL.Query().Get("after"); c != "" {
		var err error
		if cursor, err = decodeCursor[string](c); err != nil {
			serverError(w, http.StatusBadRequest, "Malformed cursor")
			return
		}
	}

	ctx := r.Context()
	uid, ok := UserIDFromContext(ctx)
	if !ok {
		serverError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var res getFeedsResBody
	var where string
	args := []any{uid}
	if cursor != nil {
		where = "WHERE (sort_key, id) > ($2, $3)"
		args = append(args, cursor.Key, cursor.Tiebreaker)
	}
	rows, err := h.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT *
		FROM (
			SELECT f.id, f.url, f.site_url, f.icon_url, f.title, f.description, LEFT(title, %d) AS sort_key
			FROM feed_subscriptions s JOIN feeds f
			ON s.user_id = $1 AND s.feed_id = f.id
		)
		%s
		ORDER BY sort_key ASC, id ASC
		LIMIT %d;
	`, getFeedsSortKeyLen, where, paginationSize+1), args...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fmt.Print(err)
		serverError(w, http.StatusInternalServerError, "Failed to fetch feeds")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	for rows.Next() {
		var f feedSchema
		var su, iu, desc sql.NullString
		err := rows.Scan(&f.ID, &f.URL, &su, &iu, &f.Title, &desc, new(string))
		if err != nil {
			fmt.Print(err)
			serverError(w, http.StatusInternalServerError, "Failed to fetch feeds")
			return
		}
		if su.Valid {
			f.SiteURL = su.String
		}
		if iu.Valid {
			f.IconURL = iu.String
		}
		if desc.Valid {
			f.Description = desc.String
		}
		res.Feeds = append(res.Feeds, f)
	}
	if rows.Err() != nil {
		fmt.Print(err)
		serverError(w, http.StatusInternalServerError, "Failed to fetch feeds")
		return
	}

	if len(res.Feeds) > paginationSize {
		res.Feeds = res.Feeds[:paginationSize]
		last := res.Feeds[len(res.Feeds)-1]
		var sortKey string
		// TODO: Use a better sort key
		if t := last.Title; len(t) < getFeedsSortKeyLen {
			sortKey = t
		} else {
			sortKey = string([]rune(t)[:getFeedsSortKeyLen])
		}
		c := paginationCusor[string]{Key: sortKey, Tiebreaker: last.ID}
		if res.NextCursor, err = encodeCursor(c); err != nil {
			fmt.Println(err)
			serverError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
	}

	jres, err := json.Marshal(res)
	if err != nil {
		fmt.Print(err)
		serverError(w, http.StatusInternalServerError, "Failed to construct JSON")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

type getFeedResBody struct {
	feedSchema
}

func (h *Handler) getFeed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		fmt.Print(err)
		serverError(w, http.StatusBadRequest, "Invalid feed id")
		return
	}

	ctx := r.Context()
	var res getFeedResBody
	var su, iu, desc sql.NullString
	err = h.DB.QueryRowContext(ctx, `
		SELECT id, url, site_url, icon_url, title, description
		FROM feeds
		WHERE id = $1;
	`, id).Scan(&res.ID, &res.URL, &su, &iu, &res.Title, &desc)
	if errors.Is(err, sql.ErrNoRows) {
		fmt.Print(err)
		serverError(w, http.StatusNotFound, "No feed found")
		return
	} else if err != nil {
		fmt.Print(err)
		serverError(w, http.StatusInternalServerError, "Failed to fetch feed")
		return
	}
	if su.Valid {
		res.SiteURL = su.String
	}
	if iu.Valid {
		res.IconURL = iu.String
	}
	if desc.Valid {
		res.Description = desc.String
	}

	jres, err := json.Marshal(res)
	if err != nil {
		fmt.Print(err)
		serverError(w, http.StatusInternalServerError, "Failed to construct JSON")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jres)
}

type feedTimelineEntry struct {
	feedEntry
	ReadLater *readLater `json:"read_later,omitempty"`
}

type getFeedTimelineResBody struct {
	Entries    []feedTimelineEntry `json:"entries"`
	NextCursor *string             `json:"next_cursor,omitempty"`
}

func (h *Handler) getFeedTimeline(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		fmt.Print(err)
		serverError(w, http.StatusBadRequest, "Invalid feed id")
		return
	}
	var cursor *paginationCusor[time.Time]
	if c := r.URL.Query().Get("after"); c != "" {
		if cursor, err = decodeCursor[time.Time](c); err != nil {
			serverError(w, http.StatusBadRequest, "Malformed cursor")
			return
		}
	}

	ctx := r.Context()
	uid, ok := UserIDFromContext(ctx)
	if !ok {
		serverError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var where string
	args := []any{id, uid}
	if cursor != nil {
		where = "AND (COALESCE(published_at, snapshot_at), id) < ($3, $4)"
		args = append(args, cursor.Key, cursor.Tiebreaker)
	}
	// TODO: Replace this correlated subquery with a LEFT JOIN once
	//       a UNIQUE (feed_entry_id) constraint exists on reading_list_items.
	// TODO: Store sort keys in DB and create an index
	rows, err := h.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, feed_id, url, title, description, published_at, snapshot_at,
			(SELECT id FROM reading_list_items
				WHERE feed_entry_id = feed_entries.id AND user_id = $2
				LIMIT 1) as reading_list_item_id,
			(SELECT archived FROM reading_list_items
				WHERE feed_entry_id = feed_entries.id AND user_id = $2
				LIMIT 1)
		FROM feed_entries
		WHERE feed_id = $1 %s
		ORDER BY COALESCE(published_at, snapshot_at) DESC, id DESC
		LIMIT %d;
	`, where, paginationSize+1), args...)
	if err != nil {
		fmt.Print(err)
		serverError(w, http.StatusInternalServerError, "Failed to fetch entries")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	res := getFeedTimelineResBody{Entries: []feedTimelineEntry{}}
	for rows.Next() {
		var e feedTimelineEntry
		var rlID *int
		var rlArchived *bool
		err := rows.Scan(
			&e.ID, &e.FeedID, &e.URL, &e.Title, &e.Description,
			&e.PublishedAt, &e.SnapshotAt, &rlID, &rlArchived,
		)
		if err != nil {
			fmt.Print(err)
			serverError(w, http.StatusInternalServerError, "Failed to fetch entry")
			return
		}
		if rlID != nil {
			e.ReadLater = &readLater{ID: *rlID, Archived: rlArchived != nil && *rlArchived}
		}
		res.Entries = append(res.Entries, e)
	}
	if err := rows.Err(); err != nil {
		fmt.Print(err)
		serverError(w, http.StatusInternalServerError, "Failed to fetch entries")
		return
	}

	if len(res.Entries) > paginationSize {
		res.Entries = res.Entries[:paginationSize]
		last := res.Entries[len(res.Entries)-1]
		sortKey := last.PublishedAt
		if sortKey == nil {
			sortKey = &last.SnapshotAt
		}
		c := paginationCusor[time.Time]{Key: *sortKey, Tiebreaker: last.ID}
		if res.NextCursor, err = encodeCursor(c); err != nil {
			fmt.Println(err)
			serverError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
	}

	jres, err := json.Marshal(res)
	if err != nil {
		fmt.Print(err)
		serverError(w, http.StatusInternalServerError, "Failed to construct JSON")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

type searchFeedsResBody struct {
	Feeds []feedAttrsSchema `json:"feeds"`
}

func (h *Handler) searchFeeds(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	fs, err := h.FeedService.SearchFeeds(r.Context(), q)
	if err != nil {
		fmt.Println(err)
		serverError(w, http.StatusNotFound, "Failed to search feeds")
		return
	}

	res := searchFeedsResBody{Feeds: []feedAttrsSchema{}}
	for _, f := range fs {
		a := feedAttrsSchema{
			URL:        f.URL.String(),
			Title:      f.Title,
			PostGroups: postGroups(f.Groups),
		}
		if u := f.SiteURL; u != nil {
			a.SiteURL = u.String()
		}
		if u := f.IconURL; u != nil {
			a.IconURL = u.String()
		}
		if d := f.Description; d != nil {
			a.Description = *d
		}
		res.Feeds = append(res.Feeds, a)
	}

	jres, err := json.Marshal(res)
	if err != nil {
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Failed to construct JSON")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jres)
}

type subscribeToFeedReqBody struct {
	URL string `json:"url"`
	// Selectors is set only for an HTML page that publishes no feed, one entry
	// per group the user ticked.
	Selectors []postSelectorsSchema `json:"selectors"`
}

type postSelectorsSchema struct {
	Root        string `json:"root"`
	Link        string `json:"link"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Timestamp   string `json:"timestamp"`
}

type subscribeToFeedResBody struct {
	feedSchema
}

func (h *Handler) subscribeToFeed(w http.ResponseWriter, r *http.Request) {
	var b subscribeToFeedReqBody
	// TODO: Limit request body size (http.MaxBytesReader)
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		fmt.Print(err)
		serverError(w, http.StatusBadRequest, "Failed to parse request body")
		return
	}
	u, err := url.Parse(b.URL)
	if err != nil {
		fmt.Print(err)
		serverError(w, http.StatusBadRequest, "Failed to parse URL")
		return
	}
	ctx := r.Context()
	uid, ok := UserIDFromContext(ctx)
	if !ok {
		serverError(w, http.StatusUnauthorized, "unauthorized")
	}

	sets := make([]feed.Selectors, 0, len(b.Selectors))
	for _, s := range b.Selectors {
		if s.Root == "" || s.Link == "" {
			serverError(w, http.StatusBadRequest, "A selector set needs both a root and a link")
			return
		}
		sets = append(sets, feed.Selectors{
			Root:        s.Root,
			Link:        s.Link,
			Title:       s.Title,
			Description: s.Description,
			Image:       s.Image,
			Timestamp:   s.Timestamp,
		})
	}

	fd, err := h.FeedService.Subscribe(ctx, uid, *u, sets)
	if err != nil {
		fmt.Print(err)
		// The selectors were read against the page the server just fetched, so
		// a set that produces no post is the client's mistake rather than a
		// server fault. Saving it would create a feed that stays empty forever.
		if errors.Is(err, feed.ErrSelectors) {
			serverError(w, http.StatusBadRequest, err.Error())
			return
		}
		serverError(w, http.StatusInternalServerError, "Failed to subscribe to feed")
		return
	}

	res := subscribeToFeedResBody{ID: fd.ID, URL: fd.URL.String(), Title: fd.Title}
	if u := fd.SiteURL; u != nil {
		res.SiteURL = u.String()
	}
	if u := fd.IconURL; u != nil {
		res.IconURL = u.String()
	}
	if d := fd.Description; d != nil {
		res.Description = *d
	}
	jres, err := json.Marshal(res)
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct a JSON")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

type saveToReadingListReqBody struct {
	URL *string `json:"url"`

	// Title is the optional placeholder, e.g. the page title shared by the browser.
	// This is only honored on the URL path; it is ignored for feed entries.
	Title *string `json:"title"`

	FeedEntryID *int `json:"feed_entry_id"`
	WebClipID   *int `json:"web_clip_id"`
}

func (h *Handler) saveToReadingList(w http.ResponseWriter, r *http.Request) {
	var b saveToReadingListReqBody
	// TODO: Limit request body size (http.MaxBytesReader)
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		fmt.Println(err)
		serverError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	argn := 0
	if b.URL != nil {
		argn++
	}
	if b.FeedEntryID != nil {
		argn++
	}
	if b.WebClipID != nil {
		argn++
	}
	if argn != 1 {
		serverError(w, http.StatusBadRequest, "Specify exactly one item to save")
		return
	}

	ctx := r.Context()
	uid, ok := UserIDFromContext(ctx)
	if !ok {
		serverError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var saved readinglist.SavedItem
	switch {
	case b.URL != nil:
		// TODO: Validate URL (schema and host)
		u, err := url.Parse(*b.URL)
		if err != nil || *b.URL == "" {
			serverError(w, http.StatusBadRequest, "Invalid URL")
			return
		}
		// TODO: trim and length-cap the client-supplied title
		var title string
		if b.Title != nil {
			title = *b.Title
		}
		saved, err = h.ReadingListService.SaveWebClip(ctx, uid, *u, title)
		if err != nil {
			fmt.Println(err)
			serverError(w, http.StatusInternalServerError, "Failed to save clip")
			return
		}

	case b.FeedEntryID != nil:
		// TODO: Return 404 instead of 500 when the given ID doesn't exist
		var err error
		saved, err = h.ReadingListService.SaveFeedEntry(ctx, uid, *b.FeedEntryID)
		if err != nil {
			fmt.Println(err)
			serverError(w, http.StatusInternalServerError, "Failed to save feed entry")
			return
		}

	case b.WebClipID != nil:
		// TODO: Return 404 instead of 500 when the given ID doesn't exist
		var err error
		saved, err = h.ReadingListService.SaveWebClipByID(ctx, uid, *b.WebClipID)
		if err != nil {
			fmt.Println(err)
			serverError(w, http.StatusInternalServerError, "Failed to save web clip")
			return
		}
	}

	jres, err := json.Marshal(readingListItem{
		ID:          saved.ID,
		ResourceID:  saved.ResourceID,
		Kind:        saved.Kind,
		Title:       saved.Title,
		Description: saved.Description,
		SavedAt:     saved.SavedAt,
	})
	if err != nil {
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Failed to construct JSON response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(jres)
}

type getReadingListResBody struct {
	Items      []readingListItem `json:"items"`
	NextCursor *string           `json:"next_cursor,omitempty"`
}

type readingListItem struct {
	ID          int       `json:"id"`
	ResourceID  int       `json:"resource_id"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"` // Make this optional
	Description *string   `json:"description,omitempty"`
	SavedAt     time.Time `json:"saved_at"`
}

func (h *Handler) getReadingList(w http.ResponseWriter, r *http.Request) {
	h.writeReadingList(w, r, false)
}

// getArchivedReadingList returns the archived reading list items, newest first.
func (h *Handler) getArchivedReadingList(w http.ResponseWriter, r *http.Request) {
	h.writeReadingList(w, r, true)
}

// writeReadingList fetches the reading list items filtered by archive status,
// paginated newest-first, and writes them as the JSON response.
func (h *Handler) writeReadingList(w http.ResponseWriter, r *http.Request, archived bool) {
	var cursor *paginationCusor[time.Time]
	if c := r.URL.Query().Get("after"); c != "" {
		var err error
		if cursor, err = decodeCursor[time.Time](c); err != nil {
			serverError(w, http.StatusBadRequest, "Malformed cursor")
			return
		}
	}

	ctx := r.Context()
	uid, ok := UserIDFromContext(ctx)
	if !ok {
		serverError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	args := []any{archived, uid}
	var where string
	if cursor != nil {
		args = append(args, cursor.Key, cursor.Tiebreaker)
		where = "AND (saved_at, id) < ($3, $4)"
	}
	rows, err := h.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, kind, title, description, saved_at, web_clip_id, feed_entry_id
		FROM reading_list_items
		WHERE archived = $1 AND user_id = $2 %s
		ORDER BY saved_at DESC, id DESC
		LIMIT %d;
	`, where, paginationSize+1), args...)
	if err != nil {
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Failed to fetch reading list")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	res := getReadingListResBody{Items: []readingListItem{}}
	for rows.Next() {
		var li readingListItem
		var wcID, feID *int
		err := rows.Scan(&li.ID, &li.Kind, &li.Title, &li.Description, &li.SavedAt, &wcID, &feID)
		if err != nil {
			fmt.Println(err)
			serverError(w, http.StatusInternalServerError, "Failed to fetch reading list item")
			return
		}
		switch li.Kind {
		case "web_clip":
			if wcID == nil {
				fmt.Println(
					"Malformed data: a reading list item of kind 'web_clip' " +
						"is expected to have a web clip ID, but it doesn't.",
				)
				serverError(w, http.StatusInternalServerError, "Failed to fetch reading list item")
				return
			}
			li.ResourceID = *wcID

		case "feed_entry":
			if feID == nil {
				fmt.Println(
					"Malformed data: a reading list item of kind 'feed_entry' " +
						"is expected to have a feed entry ID, but it doesn't.",
				)
				serverError(w, http.StatusInternalServerError, "Failed to fetch reading list item")
				return
			}
			li.ResourceID = *feID

		default:
			fmt.Printf("Unknown reading list item kind: %s\n", li.Kind)
			serverError(w, http.StatusInternalServerError, "Failed to fetch reading list item")
			return
		}
		res.Items = append(res.Items, li)
	}
	if err := rows.Err(); err != nil {
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Failed to fetch reading list")
		return
	}

	if len(res.Items) > paginationSize {
		res.Items = res.Items[:paginationSize]
		last := res.Items[len(res.Items)-1]
		c := paginationCusor[time.Time]{Key: last.SavedAt, Tiebreaker: last.ID}
		if res.NextCursor, err = encodeCursor(c); err != nil {
			serverError(w, http.StatusInternalServerError, "Something went wrong")
			return
		}
	}

	jres, err := json.Marshal(res)
	if err != nil {
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Failed to construct JSON response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

type getWebClipResBody struct {
	ID          int     `json:"id"`
	URL         string  `json:"url"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Content     *string `json:"content,omitempty"`

	// ReadLater describes the reading list item backing this clip, if it is
	// saved in the reading list (regardless of archive status). Nil (and omitted)
	// when the clip is not saved.
	ReadLater *readLater `json:"read_later,omitempty"`
}

func (h *Handler) getWebClip(w http.ResponseWriter, r *http.Request) {
	rawId := r.PathValue("id")
	id, err := strconv.Atoi(rawId)
	if err != nil {
		serverError(w, http.StatusBadRequest, fmt.Sprintf("Invalid web clip ID: %s", rawId))
		return
	}

	ctx := r.Context()
	uid, ok := UserIDFromContext(ctx)
	if !ok {
		serverError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var res getWebClipResBody
	var rlID *int
	var rlArchived *bool
	err = h.DB.QueryRowContext(ctx, `
		SELECT id, url, title, description, content,
			(SELECT id FROM reading_list_items
				WHERE web_clip_id = web_clips.id AND user_id = $2
				LIMIT 1),
			(SELECT archived FROM reading_list_items
				WHERE web_clip_id = web_clips.id AND user_id = $2
				LIMIT 1)
		FROM web_clips
		WHERE id = $1;
	`, id, uid).Scan(&res.ID, &res.URL, &res.Title, &res.Description, &res.Content, &rlID, &rlArchived)
	if errors.Is(err, sql.ErrNoRows) {
		serverError(w, http.StatusNotFound, "Web clip not found.")
		return
	} else if err != nil {
		fmt.Println(err)
		serverError(
			w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch web clip by ID=%d", id),
		)
		return
	}
	if rlID != nil {
		res.ReadLater = &readLater{ID: *rlID, Archived: rlArchived != nil && *rlArchived}
	}

	jres, err := json.Marshal(res)
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct a JSON response.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

func (h *Handler) deleteReadingListItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		serverError(w, http.StatusBadRequest, "Malformed ID")
		return
	}
	uid, authOK := UserIDFromContext(r.Context())
	if !authOK {
		serverError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ok, err := h.ReadingListService.DeleteItem(r.Context(), uid, id)
	if err != nil {
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Something went wrong.")
		return
	}
	if !ok {
		serverError(w, http.StatusNotFound, "Item not found")
		return
	}
	// TODO: DRY JSON response creation
	jres, err := json.Marshal(map[string]string{})
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct JSON response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	_, _ = w.Write(jres)
}

type setReadingListItemArchivedStatusReqBody struct {
	Archived *bool `json:"archived"`
}

func (h *Handler) setReadingListItemArchivedStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		serverError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		serverError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var b setReadingListItemArchivedStatusReqBody
	err = json.NewDecoder(r.Body).Decode(&b)
	if err != nil || b.Archived == nil {
		serverError(w, http.StatusBadRequest, "Malformed request body")
		return
	}
	if *b.Archived {
		err = h.ReadingListService.ArchiveItem(r.Context(), uid, id)
	} else {
		err = h.ReadingListService.UnarchiveItem(r.Context(), uid, id)
	}
	if errors.Is(err, readinglist.ErrItemNotFound) {
		serverError(w, http.StatusNotFound, "Item not found")
		return
	} else if err != nil {
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	// TODO: DRY JSON response creation
	jres, _ := json.Marshal(map[string]string{})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	_, _ = w.Write(jres)
}

type signUpReqBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signUpResBody struct {
	Ticket string `json:"ticket"`
}

func (h *Handler) signUp(w http.ResponseWriter, r *http.Request) {
	var req signUpReqBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serverError(w, http.StatusBadRequest, "Malformed request body")
		return
	}

	email, err := user.ParseEmail(req.Email)
	switch {
	case errors.Is(err, user.ErrEmailInvalid):
		serverError(w, http.StatusBadRequest, "Email has invalid format")
		return
	case err != nil:
		serverError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	pswd, err := user.ValidatePassword(req.Password)
	switch {
	case errors.Is(err, user.ErrPswdInvalid):
		serverError(w, http.StatusBadRequest, "Password has invalid format")
		return
	case err != nil:
		serverError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	ticket, err := h.UserService.SignUp(r.Context(), email, pswd)
	switch {
	case errors.Is(err, user.ErrEmailTaken):
		serverError(w, http.StatusConflict, "Email already exists")
		return
	case errors.Is(err, user.ErrTooManyAttempts):
		serverError(w, http.StatusTooManyRequests, tooManyVerificationEmailsMsg)
		return
	case err != nil:
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	// TODO: DRY JSON response creation
	jres, err := json.Marshal(signUpResBody{Ticket: ticket.Encode()})
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct a JSON response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write(jres)
}

// Both ErrTokenInvalid and ErrTokenExpired become a 404: the client is not told
// whether the ticket is unknown, expired, or out of guesses.
const noPendingSignUpMsg = "No pending sign-up found for this ticket"

const tooManyVerificationEmailsMsg = "Too many verification emails sent to this address"

type verifySignUpEmailReqBody struct {
	Ticket           string `json:"ticket"`
	VerificationCode string `json:"verification_code"`
	Device           string `json:"device"`
}

type verifySignUpEmailResBody struct {
	Token string `json:"token"`
}

func (h *Handler) verifySignUpEmail(w http.ResponseWriter, r *http.Request) {
	var req verifySignUpEmailReqBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serverError(w, http.StatusBadRequest, "Malformed request body")
		return
	}

	token, err := h.UserService.VerifySignUpEmailAddress(
		r.Context(), req.Ticket, req.VerificationCode, req.Device,
	)
	switch {
	case errors.Is(err, user.ErrCodeInvalid):
		serverError(w, http.StatusBadRequest, "Verification code has invalid format")
		return
	case errors.Is(err, user.ErrDeviceEmpty):
		serverError(w, http.StatusBadRequest, "Device is empty")
		return
	case errors.Is(err, user.ErrEmailVerifyFailed):
		serverError(w, http.StatusUnauthorized, "Verification code is incorrect")
		return
	case errors.Is(err, user.ErrTokenInvalid), errors.Is(err, user.ErrTokenExpired):
		serverError(w, http.StatusNotFound, noPendingSignUpMsg)
		return
	case errors.Is(err, user.ErrEmailTaken):
		serverError(w, http.StatusConflict, "Email already exists")
		return
	case err != nil:
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	// TODO: DRY JSON response creation
	jres, err := json.Marshal(verifySignUpEmailResBody{Token: token.Encode()})
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct a JSON response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

type resendSignUpVerificationReqBody struct {
	Ticket string `json:"ticket"`
}

func (h *Handler) resendSignUpVerification(w http.ResponseWriter, r *http.Request) {
	var req resendSignUpVerificationReqBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serverError(w, http.StatusBadRequest, "Malformed request body")
		return
	}

	ticket, err := h.UserService.ResendSignUpVerificationEmail(r.Context(), req.Ticket)
	switch {
	case errors.Is(err, user.ErrTokenInvalid), errors.Is(err, user.ErrTokenExpired):
		serverError(w, http.StatusNotFound, noPendingSignUpMsg)
		return
	case errors.Is(err, user.ErrEmailTaken):
		serverError(w, http.StatusConflict, "Email already exists")
		return
	case errors.Is(err, user.ErrTooManyAttempts):
		serverError(w, http.StatusTooManyRequests, tooManyVerificationEmailsMsg)
		return
	case err != nil:
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	// TODO: DRY JSON response creation
	jres, err := json.Marshal(signUpResBody{Ticket: ticket.Encode()})
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct a JSON response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write(jres)
}

type signInReqBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Device   string `json:"device"`
}

type signInResBody struct {
	Token string `json:"token"`
}

func (h *Handler) signIn(w http.ResponseWriter, r *http.Request) {
	var req signInReqBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		serverError(w, http.StatusBadRequest, "Malformed request body")
		return
	}

	email, err := user.ParseEmail(req.Email)
	switch {
	case err != nil:
		serverError(w, http.StatusUnauthorized, "Email or password is incorrect")
		return
	}

	token, err := h.UserService.SignIn(r.Context(), email, req.Password, req.Device)
	switch {
	case errors.Is(err, user.ErrDeviceEmpty):
		serverError(w, http.StatusBadRequest, "Device is empty")
		return
	case errors.Is(err, user.ErrAuthFailed):
		serverError(w, http.StatusUnauthorized, "Email or password is incorrect")
		return
	case err != nil:
		fmt.Println(err)
		serverError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	// TODO: DRY JSON response creation
	jres, err := json.Marshal(signInResBody{Token: token.Encode()})
	if err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to construct a JSON response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

func (h *Handler) signOut(w http.ResponseWriter, r *http.Request) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		// TODO: DRY JSON response creation
		jres, _ := json.Marshal(map[string]string{})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jres)
		return
	}
	if err := h.UserService.SignOut(r.Context(), token); err != nil {
		serverError(w, http.StatusInternalServerError, "Failed to sign out")
		return
	}
	// TODO: DRY JSON response creation
	jres, _ := json.Marshal(map[string]string{})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jres)
}

func serverError(w http.ResponseWriter, statusCode int, msg string) {
	res, err := json.Marshal(map[string]any{
		"message": msg,
	})
	if err != nil {
		http.Error(w, msg, http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write(res)
	}
}

const paginationSize = 50

type paginationCusor[T any] struct {
	Key        T   `json:"key"`
	Tiebreaker int `json:"tiebreaker"`
}

func encodeCursor[T any](cursor paginationCusor[T]) (*string, error) {
	c, err := json.Marshal(cursor)
	if err != nil {
		return nil, err
	}
	return new(base64.RawURLEncoding.EncodeToString(c)), nil
}

func decodeCursor[T any](cursor string) (*paginationCusor[T], error) {
	d, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	c := new(paginationCusor[T])
	err = json.Unmarshal(d, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
