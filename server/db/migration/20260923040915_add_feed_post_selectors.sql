-- +goose Up
-- One saved recipe for reading posts out of an HTML page that publishes no
-- feed. A feed that has rows here is a page feed; a feed that has none is an
-- RSS/Atom feed, so no column on feeds is needed to tell the two apart.
--
-- The columns hold the keys the user picked on the subscription screen. They
-- read as CSS selectors, but the server never compiles them: it enumerates the
-- page again and compares them with the keys that enumeration writes. See
-- "What a selector is" in server/feature/feed/PLAN.md.
--
-- published_at rather than timestamp, because timestamp is a type name in
-- Postgres.
CREATE TABLE feed_post_selectors (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    feed_id bigint NOT NULL REFERENCES feeds (id),
    root text NOT NULL,
    link text NOT NULL,
    title text,
    description text,
    image text,
    published_at text
);

CREATE INDEX idx_feed_post_selectors_feed_id ON feed_post_selectors (feed_id);

-- +goose Down
DROP TABLE feed_post_selectors;
